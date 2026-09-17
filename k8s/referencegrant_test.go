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
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
	gatewayv1beta1 "sigs.k8s.io/gateway-api/apis/v1beta1"

	"github.com/fabiocicerchia/go-proxy-cache/config"
)

// crossNamespaceRule - A rule whose backend deliberately points at another
// namespace, which is what a ReferenceGrant has to authorise.
func crossNamespaceRule(path string, namespace string, svc string, port int32) gatewayv1.HTTPRouteRule {
	pathType := gatewayv1.PathMatchPathPrefix
	pathValue := path
	svcPort := gatewayv1.PortNumber(port)
	svcNamespace := gatewayv1.Namespace(namespace)

	return gatewayv1.HTTPRouteRule{
		Matches: []gatewayv1.HTTPRouteMatch{{
			Path: &gatewayv1.HTTPPathMatch{Type: &pathType, Value: &pathValue},
		}},
		BackendRefs: []gatewayv1.HTTPBackendRef{{
			BackendRef: gatewayv1.BackendRef{
				BackendObjectReference: gatewayv1.BackendObjectReference{
					Name:      gatewayv1.ObjectName(svc),
					Namespace: &svcNamespace,
					Port:      &svcPort,
				},
			},
		}},
	}
}

func referenceGrant(namespace string, name string, fromKind string, fromNamespace string, toKind string, toName string) *gatewayv1beta1.ReferenceGrant {
	to := gatewayv1beta1.ReferenceGrantTo{Group: "", Kind: gatewayv1.Kind(toKind)}

	if toName != "" {
		objectName := gatewayv1.ObjectName(toName)
		to.Name = &objectName
	}

	return &gatewayv1beta1.ReferenceGrant{
		ObjectMeta: metav1.ObjectMeta{Namespace: namespace, Name: name},
		Spec: gatewayv1beta1.ReferenceGrantSpec{
			From: []gatewayv1beta1.ReferenceGrantFrom{{
				Group:     gatewayv1.GroupName,
				Kind:      gatewayv1.Kind(fromKind),
				Namespace: gatewayv1.Namespace(fromNamespace),
			}},
			To: []gatewayv1beta1.ReferenceGrantTo{to},
		},
	}
}

// Without a grant, a tenant could point an HTTPRoute at any Service in the
// cluster and republish it on the shared ingress.
func TestCrossNamespaceBackendRefIsRefusedWithoutAGrant(t *testing.T) {
	gw := gateway("default", "gw", "gpc", "")
	hr := httpRoute("tenant-a", "demo", "gw", []string{"demo.local"}, []gatewayv1.HTTPRouteRule{
		crossNamespaceRule("/", "platform", "internal-admin", 8080),
	})
	hr.Spec.ParentRefs[0].Namespace = namespacePtr("default")

	c, _, cancel := newTestController(t, gatewayOptions(), []runtime.Object{
		service("platform", "internal-admin", 8080, "http"),
		endpointSlice("platform", "internal-admin", "http", 8080, "10.9.9.9"),
	}, []runtime.Object{
		gatewayClass("gpc", ControllerName),
		gw,
		hr,
	})
	defer cancel()

	gw.Spec.Listeners[0].AllowedRoutes = allowedFromAll()

	routes := c.syncGatewayAPI(context.Background(), config.Config, emptyCertMap())

	assert.Empty(t, routes, "a cross-namespace backend must not resolve without a ReferenceGrant")
}

func TestCrossNamespaceBackendRefIsAllowedByAGrant(t *testing.T) {
	gw := gateway("default", "gw", "gpc", "")
	gw.Spec.Listeners[0].AllowedRoutes = allowedFromAll()

	hr := httpRoute("tenant-a", "demo", "gw", []string{"demo.local"}, []gatewayv1.HTTPRouteRule{
		crossNamespaceRule("/", "platform", "shared-api", 8080),
	})
	hr.Spec.ParentRefs[0].Namespace = namespacePtr("default")

	c, _, cancel := newTestController(t, gatewayOptions(), []runtime.Object{
		service("platform", "shared-api", 8080, "http"),
		endpointSlice("platform", "shared-api", "http", 8080, "10.9.9.9"),
	}, []runtime.Object{
		gatewayClass("gpc", ControllerName),
		gw,
		hr,
		referenceGrant("platform", "allow-tenant-a", "HTTPRoute", "tenant-a", "Service", "shared-api"),
	})
	defer cancel()

	routes := c.syncGatewayAPI(context.Background(), config.Config, emptyCertMap())

	assert.Len(t, routes, 1)
	assert.Equal(t, []string{"10.9.9.9:8080"}, routes[0].Backends[0].Endpoints)
}

// A grant naming no object covers every object of that kind in the namespace.
func TestGrantWithoutANameCoversEveryObject(t *testing.T) {
	grant := referenceGrant("platform", "allow-all", "HTTPRoute", "tenant-a", "Service", "")

	c, _, cancel := newTestController(t, gatewayOptions(), nil, []runtime.Object{grant})
	defer cancel()

	assert.True(t, c.permitted(reference{
		FromGroup: gatewayv1.GroupName, FromKind: "HTTPRoute", FromNamespace: "tenant-a",
		ToGroup: "", ToKind: "Service", ToName: "anything", ToNamespace: "platform",
	}))
}

func TestPermittedRules(t *testing.T) {
	grant := referenceGrant("platform", "allow", "HTTPRoute", "tenant-a", "Service", "shared-api")

	c, _, cancel := newTestController(t, gatewayOptions(), nil, []runtime.Object{grant})
	defer cancel()

	sameNamespace := reference{
		FromGroup: gatewayv1.GroupName, FromKind: "HTTPRoute", FromNamespace: "platform",
		ToGroup: "", ToKind: "Service", ToName: "whatever", ToNamespace: "platform",
	}
	assert.True(t, c.permitted(sameNamespace), "a same-namespace reference never needs a grant")

	wrongName := reference{
		FromGroup: gatewayv1.GroupName, FromKind: "HTTPRoute", FromNamespace: "tenant-a",
		ToGroup: "", ToKind: "Service", ToName: "other-api", ToNamespace: "platform",
	}
	assert.False(t, c.permitted(wrongName), "a grant naming one Service must not cover another")

	wrongNamespace := reference{
		FromGroup: gatewayv1.GroupName, FromKind: "HTTPRoute", FromNamespace: "tenant-b",
		ToGroup: "", ToKind: "Service", ToName: "shared-api", ToNamespace: "platform",
	}
	assert.False(t, c.permitted(wrongNamespace), "a grant naming one namespace must not cover another")

	wrongKind := reference{
		FromGroup: gatewayv1.GroupName, FromKind: "Gateway", FromNamespace: "tenant-a",
		ToGroup: "", ToKind: "Secret", ToName: "shared-api", ToNamespace: "platform",
	}
	assert.False(t, c.permitted(wrongKind), "a Service grant must not authorise a Secret")
}

// The certificate path is the more damaging of the two: without the check a
// tenant able to create a Gateway serves another namespace's private key.
func TestCrossNamespaceCertificateRefIsRefusedWithoutAGrant(t *testing.T) {
	gw := gatewayWithTLS("tenant-a", "gw", "gpc", "login.victim.example.com", "victim-prod", "victim-tls")

	c, _, cancel := newTestController(t, gatewayOptions(), []runtime.Object{
		namedTLSSecret(t, "victim-prod", "victim-tls", "login.victim.example.com"),
	}, []runtime.Object{
		gatewayClass("gpc", ControllerName),
		gw,
	})
	defer cancel()

	certs := emptyCertMap()
	refused := map[string]bool{}

	c.loadGatewayCertificates(gw, certs, refused)

	assert.Empty(t, certs, "another namespace's key must not be loaded without a ReferenceGrant")
	assert.True(t, refused[listenerKey(gw, "https")])
}

func TestCrossNamespaceCertificateRefIsAllowedByAGrant(t *testing.T) {
	gw := gatewayWithTLS("tenant-a", "gw", "gpc", "shared.example.com", "platform", "shared-tls")

	c, _, cancel := newTestController(t, gatewayOptions(), []runtime.Object{
		namedTLSSecret(t, "platform", "shared-tls", "shared.example.com"),
	}, []runtime.Object{
		gatewayClass("gpc", ControllerName),
		gw,
		referenceGrant("platform", "allow-tenant-a", "Gateway", "tenant-a", "Secret", "shared-tls"),
	})
	defer cancel()

	certs := emptyCertMap()
	refused := map[string]bool{}

	c.loadGatewayCertificates(gw, certs, refused)

	assert.Contains(t, certs, "shared.example.com")
	assert.Empty(t, refused)
}

// namedTLSSecret - A TLS Secret in a chosen namespace. The shared tlsSecret
// helper is hardcoded to default/tls, and these tests turn on which namespace
// the Secret lives in.
func namedTLSSecret(t *testing.T, namespace string, name string, host string) *corev1.Secret {
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

// gatewayWithTLS - A Gateway with one HTTPS listener whose certificate lives in
// the given namespace.
func gatewayWithTLS(namespace, name, className, hostname, secretNamespace, secretName string) *gatewayv1.Gateway {
	host := gatewayv1.Hostname(hostname)
	secretNS := gatewayv1.Namespace(secretNamespace)
	kind := gatewayv1.Kind("Secret")

	return &gatewayv1.Gateway{
		ObjectMeta: metav1.ObjectMeta{Namespace: namespace, Name: name},
		Spec: gatewayv1.GatewaySpec{
			GatewayClassName: gatewayv1.ObjectName(className),
			Listeners: []gatewayv1.Listener{{
				Name:     "https",
				Port:     443,
				Protocol: gatewayv1.HTTPSProtocolType,
				Hostname: &host,
				TLS: &gatewayv1.ListenerTLSConfig{
					CertificateRefs: []gatewayv1.SecretObjectReference{{
						Kind:      &kind,
						Name:      gatewayv1.ObjectName(secretName),
						Namespace: &secretNS,
					}},
				},
			}},
		},
	}
}

func namespacePtr(namespace string) *gatewayv1.Namespace {
	value := gatewayv1.Namespace(namespace)

	return &value
}

func allowedFromAll() *gatewayv1.AllowedRoutes {
	from := gatewayv1.NamespacesFromAll

	return &gatewayv1.AllowedRoutes{
		Namespaces: &gatewayv1.RouteNamespaces{From: &from},
	}
}
