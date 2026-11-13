//go:build all || unit
// +build all unit

package k8s

//                                                                         __
// .-----.-----.______.-----.----.-----.--.--.--.--.______.----.---.-.----|  |--.-----.
// |  _  |  _  |______|  _  |   _|  _  |_   _|  |  |______|  __|  _  |  __|     |  -__|
// |___  |_____|      |   __|__| |_____|__.__|___  |      |____|___._|____|__|__|_____|
// |_____|            |__|                   |_____|
//
// Copyright (c) 2023 Fabio Cicerchia. https://fabiocicerchia.it. MIT License
// Repo: https://github.com/fabiocicerchia/go-proxy-cache

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/fabiocicerchia/go-proxy-cache/server/router"
)

// selfSignedSecret - A minimal but genuinely valid kubernetes.io/tls Secret,
// generated once for the certificate tests.
func selfSignedSecret(t *testing.T, namespace string, name string, host string) *corev1.Secret {
	t.Helper()

	certPEM, keyPEM := generateSelfSigned(t, host)

	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Namespace: namespace, Name: name},
		Type:       corev1.SecretTypeTLS,
		Data: map[string][]byte{
			corev1.TLSCertKey:       certPEM,
			corev1.TLSPrivateKeyKey: keyPEM,
		},
	}
}

func TestSyncPublishesRoutingTableAndBalancers(t *testing.T) {
	defer router.Reset()

	ing := simpleIngress("default", "demo", "demo.local", "/api", networkingv1.PathTypePrefix, "web", 80)

	c, registry, cancel := newTestController(t, testOptions(), []runtime.Object{
		ingressClass(IngressClassName, ControllerName, false),
		ing,
		service("default", "web", 80, "http"),
		endpointSlice("default", "web", "http", 8080, "10.1.0.1", "10.1.0.2"),
	}, nil)
	defer cancel()

	router.Enable()
	c.sync(context.Background())

	table := router.Current()
	assert.Equal(t, 1, table.Len())

	req := httptest.NewRequest(http.MethodGet, "/api/v1/items", nil)
	req.Host = "demo.local"

	route, found := table.Match(req)
	assert.True(t, found)
	assert.Equal(t, []string{"10.1.0.1:8080", "10.1.0.2:8080"}, route.Backends[0].Endpoints)

	// The balancer registry must be told about exactly the backends in play.
	assert.Len(t, registry.last, 1)

	upstream, ok := registry.last[route.BalancerID(0)]
	assert.True(t, ok)
	assert.Equal(t, []string{"10.1.0.1:8080", "10.1.0.2:8080"}, upstream.Endpoints)
}

func TestSyncIgnoresIngressesOfOtherControllers(t *testing.T) {
	defer router.Reset()

	other := "nginx"
	theirs := simpleIngress("default", "theirs", "theirs.local", "/", networkingv1.PathTypePrefix, "web", 80)
	theirs.Spec.IngressClassName = &other

	c, _, cancel := newTestController(t, testOptions(), []runtime.Object{
		ingressClass(IngressClassName, ControllerName, false),
		ingressClass(other, "k8s.io/ingress-nginx", false),
		theirs,
		service("default", "web", 80, "http"),
		endpointSlice("default", "web", "http", 8080, "10.1.0.1"),
	}, nil)
	defer cancel()

	router.Enable()
	c.sync(context.Background())

	assert.Equal(t, 0, router.Current().Len())
}

func TestSyncLoadsCertificatesFromSecrets(t *testing.T) {
	defer router.Reset()

	ing := simpleIngress("default", "demo", "demo.local", "/", networkingv1.PathTypePrefix, "web", 80)
	ing.Spec.TLS = []networkingv1.IngressTLS{{
		Hosts:      []string{"demo.local"},
		SecretName: "demo-tls",
	}}

	c, _, cancel := newTestController(t, testOptions(), []runtime.Object{
		ingressClass(IngressClassName, ControllerName, false),
		ing,
		service("default", "web", 80, "http"),
		endpointSlice("default", "web", "http", 8080, "10.1.0.1"),
		selfSignedSecret(t, "default", "demo-tls", "demo.local"),
	}, nil)
	defer cancel()

	router.Enable()
	c.sync(context.Background())

	got, ok := c.certs.Get("demo.local")

	assert.True(t, ok)
	assert.NotNil(t, got)
}

func TestSyncSkipsMissingSecretWithoutDroppingTheRoute(t *testing.T) {
	defer router.Reset()

	ing := simpleIngress("default", "demo", "demo.local", "/", networkingv1.PathTypePrefix, "web", 80)
	ing.Spec.TLS = []networkingv1.IngressTLS{{
		Hosts:      []string{"demo.local"},
		SecretName: "does-not-exist",
	}}

	c, _, cancel := newTestController(t, testOptions(), []runtime.Object{
		ingressClass(IngressClassName, ControllerName, false),
		ing,
		service("default", "web", 80, "http"),
		endpointSlice("default", "web", "http", 8080, "10.1.0.1"),
	}, nil)
	defer cancel()

	router.Enable()
	c.sync(context.Background())

	assert.Equal(t, 1, router.Current().Len(), "a missing certificate must not remove the HTTP route")
	assert.Equal(t, 0, c.certs.Len())
}

func TestSyncRemovesBalancersForDeletedRoutes(t *testing.T) {
	defer router.Reset()

	ing := simpleIngress("default", "demo", "demo.local", "/", networkingv1.PathTypePrefix, "web", 80)

	c, registry, cancel := newTestController(t, testOptions(), []runtime.Object{
		ingressClass(IngressClassName, ControllerName, false),
		ing,
		service("default", "web", 80, "http"),
		endpointSlice("default", "web", "http", 8080, "10.1.0.1"),
	}, nil)
	defer cancel()

	router.Enable()
	c.sync(context.Background())
	assert.Len(t, registry.last, 1)

	err := c.factory.Networking().V1().Ingresses().Informer().GetIndexer().Delete(ing)
	assert.NoError(t, err)

	c.sync(context.Background())

	assert.Equal(t, 0, router.Current().Len())
	assert.Empty(t, registry.last, "the balancer for a deleted route must be reconciled away")
}

func TestRoutedDomainsExposesEachHost(t *testing.T) {
	routes := []*router.Route{
		{Host: "a.local"},
		{Host: "a.local"},
		{Host: "b.local"},
		{Host: ""},
	}

	domains := routedDomains(routes)

	assert.Len(t, domains, 2)
	assert.Equal(t, "a.local", domains["a.local"].Server.Upstream.Host)
	assert.Equal(t, "b.local", domains["b.local"].Server.Upstream.Host)
}
