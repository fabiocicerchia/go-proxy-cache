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
	"testing"

	"github.com/stretchr/testify/assert"
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/fabiocicerchia/go-proxy-cache/config"
	"github.com/fabiocicerchia/go-proxy-cache/server/router"
)

func TestTranslateIngressResolvesPodIPs(t *testing.T) {
	ing := simpleIngress("default", "demo", "demo.local", "/", networkingv1.PathTypePrefix, "web", 80)

	c, _, cancel := newTestController(t, testOptions(), []runtime.Object{
		ingressClass(IngressClassName, ControllerName, false),
		ing,
		service("default", "web", 80, "http"),
		endpointSlice("default", "web", "http", 8080, "10.1.0.1", "10.1.0.2"),
	}, nil)
	defer cancel()

	routes, problems := c.translateIngress(ing, config.Config)

	assert.Empty(t, problems)
	assert.Len(t, routes, 1)

	route := routes[0]
	assert.Equal(t, "demo.local", route.Host)
	assert.Equal(t, "/", route.Path)
	assert.Equal(t, router.PathPrefix, route.PathType)
	assert.Len(t, route.Backends, 1)

	// The Service port is 80 but the pods listen on 8080: the EndpointSlice
	// target port is what has to be dialled.
	assert.Equal(t, []string{"10.1.0.1:8080", "10.1.0.2:8080"}, route.Backends[0].Endpoints)
	assert.True(t, route.PreserveHost, "Ingress preserves the client Host by default")
}

func TestTranslateIngressSkipsUnreadyAndTerminatingPods(t *testing.T) {
	slice := endpointSlice("default", "web", "http", 8080, "10.1.0.1", "10.1.0.2", "10.1.0.3")

	notReady := false
	slice.Endpoints[1].Conditions.Ready = &notReady

	terminating := true
	slice.Endpoints[2].Conditions.Terminating = &terminating

	ing := simpleIngress("default", "demo", "demo.local", "/", networkingv1.PathTypePrefix, "web", 80)

	c, _, cancel := newTestController(t, testOptions(), []runtime.Object{
		ingressClass(IngressClassName, ControllerName, false),
		ing,
		service("default", "web", 80, "http"),
		slice,
	}, nil)
	defer cancel()

	routes, _ := c.translateIngress(ing, config.Config)

	assert.Len(t, routes, 1)
	assert.Equal(t, []string{"10.1.0.1:8080"}, routes[0].Backends[0].Endpoints)
}

func TestTranslateIngressFallsBackToServiceDNSWithNoEndpoints(t *testing.T) {
	ing := simpleIngress("default", "demo", "demo.local", "/", networkingv1.PathTypePrefix, "web", 80)

	c, _, cancel := newTestController(t, testOptions(), []runtime.Object{
		ingressClass(IngressClassName, ControllerName, false),
		ing,
		service("default", "web", 80, "http"),
	}, nil)
	defer cancel()

	routes, problems := c.translateIngress(ing, config.Config)

	assert.Empty(t, problems)
	assert.Len(t, routes, 1)
	assert.Equal(t, []string{"web.default.svc:80"}, routes[0].Backends[0].Endpoints)
}

func TestTranslateIngressReportsMissingServiceButKeepsOtherPaths(t *testing.T) {
	pathType := networkingv1.PathTypePrefix
	ing := simpleIngress("default", "demo", "demo.local", "/ok", pathType, "web", 80)

	ing.Spec.Rules[0].HTTP.Paths = append(ing.Spec.Rules[0].HTTP.Paths, networkingv1.HTTPIngressPath{
		Path:     "/broken",
		PathType: &pathType,
		Backend: networkingv1.IngressBackend{
			Service: &networkingv1.IngressServiceBackend{
				Name: "missing",
				Port: networkingv1.ServiceBackendPort{Number: 80},
			},
		},
	})

	c, _, cancel := newTestController(t, testOptions(), []runtime.Object{
		ingressClass(IngressClassName, ControllerName, false),
		ing,
		service("default", "web", 80, "http"),
		endpointSlice("default", "web", "http", 8080, "10.1.0.1"),
	}, nil)
	defer cancel()

	routes, problems := c.translateIngress(ing, config.Config)

	// One broken backend must not take the working path out of service.
	assert.Len(t, routes, 1)
	assert.Equal(t, "/ok", routes[0].Path)
	assert.Len(t, problems, 1)
}

func TestTranslateIngressPathTypes(t *testing.T) {
	cases := []struct {
		name     string
		pathType networkingv1.PathType
		useRegex bool
		want     router.PathType
	}{
		{"exact", networkingv1.PathTypeExact, false, router.PathExact},
		{"prefix", networkingv1.PathTypePrefix, false, router.PathPrefix},
		{"implementation specific defaults to prefix", networkingv1.PathTypeImplementationSpecific, false, router.PathPrefix},
		{"implementation specific with use-regex", networkingv1.PathTypeImplementationSpecific, true, router.PathRegex},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ing := simpleIngress("default", "demo", "demo.local", "/api", tc.pathType, "web", 80)
			if tc.useRegex {
				ing.Annotations = map[string]string{AnnotationPrefix + AnnUsePathRegex: "true"}
			}

			c, _, cancel := newTestController(t, testOptions(), []runtime.Object{
				ingressClass(IngressClassName, ControllerName, false),
				ing,
				service("default", "web", 80, "http"),
				endpointSlice("default", "web", "http", 8080, "10.1.0.1"),
			}, nil)
			defer cancel()

			routes, _ := c.translateIngress(ing, config.Config)

			assert.Len(t, routes, 1)
			assert.Equal(t, tc.want, routes[0].PathType)
		})
	}
}

func TestTranslateIngressDefaultBackend(t *testing.T) {
	className := IngressClassName
	ing := &networkingv1.Ingress{}
	ing.Namespace = "default"
	ing.Name = "fallback"
	ing.Spec.IngressClassName = &className
	ing.Spec.DefaultBackend = &networkingv1.IngressBackend{
		Service: &networkingv1.IngressServiceBackend{
			Name: "web",
			Port: networkingv1.ServiceBackendPort{Number: 80},
		},
	}

	c, _, cancel := newTestController(t, testOptions(), []runtime.Object{
		ingressClass(IngressClassName, ControllerName, false),
		ing,
		service("default", "web", 80, "http"),
		endpointSlice("default", "web", "http", 8080, "10.1.0.1"),
	}, nil)
	defer cancel()

	routes, problems := c.translateIngress(ing, config.Config)

	assert.Empty(t, problems)
	assert.Len(t, routes, 1)
	assert.Equal(t, "", routes[0].Host, "a default backend is a hostless catch-all")
	assert.Equal(t, "/", routes[0].Path)
}

func TestClaimsIngress(t *testing.T) {
	opts := testOptions()
	other := "other-class"

	classes := []*networkingv1.IngressClass{
		ingressClass(IngressClassName, ControllerName, false),
		ingressClass(other, "example.com/other", false),
	}

	ours := simpleIngress("default", "a", "a.local", "/", networkingv1.PathTypePrefix, "web", 80)
	assert.True(t, claimsIngress(ours, classes, opts))

	theirs := simpleIngress("default", "b", "b.local", "/", networkingv1.PathTypePrefix, "web", 80)
	theirs.Spec.IngressClassName = &other
	assert.False(t, claimsIngress(theirs, classes, opts))

	unset := simpleIngress("default", "c", "c.local", "/", networkingv1.PathTypePrefix, "web", 80)
	unset.Spec.IngressClassName = nil
	assert.False(t, claimsIngress(unset, classes, opts), "no class and no default must not be claimed")

	withDefault := []*networkingv1.IngressClass{ingressClass(IngressClassName, ControllerName, true)}
	assert.True(t, claimsIngress(unset, withDefault, opts), "the default class claims classless Ingresses")

	legacy := simpleIngress("default", "d", "d.local", "/", networkingv1.PathTypePrefix, "web", 80)
	legacy.Spec.IngressClassName = nil
	legacy.Annotations = map[string]string{LegacyIngressClassAnnotation: IngressClassName}
	assert.True(t, claimsIngress(legacy, classes, opts), "the legacy annotation is still honoured")

	legacyOther := simpleIngress("default", "e", "e.local", "/", networkingv1.PathTypePrefix, "web", 80)
	legacyOther.Annotations = map[string]string{LegacyIngressClassAnnotation: "nginx"}
	assert.False(t, claimsIngress(legacyOther, classes, opts))
}

func TestClaimedIngressesAreFilteredAndSorted(t *testing.T) {
	other := "nginx"

	theirs := simpleIngress("default", "z-theirs", "z.local", "/", networkingv1.PathTypePrefix, "web", 80)
	theirs.Spec.IngressClassName = &other

	c, _, cancel := newTestController(t, testOptions(), []runtime.Object{
		ingressClass(IngressClassName, ControllerName, false),
		ingressClass(other, "k8s.io/ingress-nginx", false),
		simpleIngress("default", "b", "b.local", "/", networkingv1.PathTypePrefix, "web", 80),
		simpleIngress("default", "a", "a.local", "/", networkingv1.PathTypePrefix, "web", 80),
		theirs,
	}, nil)
	defer cancel()

	claimed := c.claimedIngresses()

	assert.Len(t, claimed, 2)
	assert.Equal(t, "a", claimed[0].Name)
	assert.Equal(t, "b", claimed[1].Name)
}

func TestRewriteTargetProducesPrefixRewrite(t *testing.T) {
	ing := simpleIngress("default", "demo", "demo.local", "/api", networkingv1.PathTypePrefix, "web", 80)
	ing.Annotations = map[string]string{AnnotationPrefix + AnnRewriteTarget: "/"}

	c, _, cancel := newTestController(t, testOptions(), []runtime.Object{
		ingressClass(IngressClassName, ControllerName, false),
		ing,
		service("default", "web", 80, "http"),
		endpointSlice("default", "web", "http", 8080, "10.1.0.1"),
	}, nil)
	defer cancel()

	routes, _ := c.translateIngress(ing, config.Config)

	assert.Len(t, routes, 1)
	assert.Len(t, routes[0].Filters, 1)
	assert.Equal(t, router.FilterURLRewrite, routes[0].Filters[0].Type)
	assert.NotNil(t, routes[0].Filters[0].Rewrite.ReplacePrefixMatch)
	assert.Equal(t, "/", *routes[0].Filters[0].Rewrite.ReplacePrefixMatch)
}
