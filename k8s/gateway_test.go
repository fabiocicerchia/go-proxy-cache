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
	"crypto/tls"
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"

	"github.com/fabiocicerchia/go-proxy-cache/config"
	"github.com/fabiocicerchia/go-proxy-cache/server/router"
)

func gateway(namespace string, name string, className string, listenerHostname string) *gatewayv1.Gateway {
	listener := gatewayv1.Listener{
		Name:     "http",
		Port:     80,
		Protocol: gatewayv1.HTTPProtocolType,
	}

	if listenerHostname != "" {
		hostname := gatewayv1.Hostname(listenerHostname)
		listener.Hostname = &hostname
	}

	return &gatewayv1.Gateway{
		ObjectMeta: metav1.ObjectMeta{Namespace: namespace, Name: name},
		Spec: gatewayv1.GatewaySpec{
			GatewayClassName: gatewayv1.ObjectName(className),
			Listeners:        []gatewayv1.Listener{listener},
		},
	}
}

func httpRoute(namespace string, name string, gatewayName string, hostnames []string, rules []gatewayv1.HTTPRouteRule) *gatewayv1.HTTPRoute {
	hosts := make([]gatewayv1.Hostname, 0, len(hostnames))
	for _, h := range hostnames {
		hosts = append(hosts, gatewayv1.Hostname(h))
	}

	return &gatewayv1.HTTPRoute{
		ObjectMeta: metav1.ObjectMeta{Namespace: namespace, Name: name},
		Spec: gatewayv1.HTTPRouteSpec{
			CommonRouteSpec: gatewayv1.CommonRouteSpec{
				ParentRefs: []gatewayv1.ParentReference{{Name: gatewayv1.ObjectName(gatewayName)}},
			},
			Hostnames: hosts,
			Rules:     rules,
		},
	}
}

func backendRule(path string, pathType gatewayv1.PathMatchType, svc string, port int32, weight *int32) gatewayv1.HTTPRouteRule {
	pathValue := path
	svcPort := gatewayv1.PortNumber(port)

	return gatewayv1.HTTPRouteRule{
		Matches: []gatewayv1.HTTPRouteMatch{{
			Path: &gatewayv1.HTTPPathMatch{Type: &pathType, Value: &pathValue},
		}},
		BackendRefs: []gatewayv1.HTTPBackendRef{{
			BackendRef: gatewayv1.BackendRef{
				BackendObjectReference: gatewayv1.BackendObjectReference{
					Name: gatewayv1.ObjectName(svc),
					Port: &svcPort,
				},
				Weight: weight,
			},
		}},
	}
}

// emptyCertMap - The certificate sink the sync writes into; the Gateway
// translation tests only care about the routes it returns.
func emptyCertMap() map[string]*tls.Certificate {
	return map[string]*tls.Certificate{}
}

func gatewayOptions() Options {
	opts := testOptions()
	opts.EnableGatewayAPI = true

	return opts
}

func TestSyncGatewayAPITranslatesHTTPRoute(t *testing.T) {
	gw := gateway("default", "gw", "gpc", "")
	hr := httpRoute("default", "demo", "gw", []string{"demo.local"}, []gatewayv1.HTTPRouteRule{
		backendRule("/api", gatewayv1.PathMatchPathPrefix, "web", 80, nil),
	})

	c, _, cancel := newTestController(t, gatewayOptions(), []runtime.Object{
		service("default", "web", 80, "http"),
		endpointSlice("default", "web", "http", 8080, "10.1.0.1"),
	}, []runtime.Object{
		gatewayClass("gpc", ControllerName),
		gw,
		hr,
	})
	defer cancel()

	routes := c.syncGatewayAPI(context.Background(), config.Config, emptyCertMap())

	assert.Len(t, routes, 1)
	assert.Equal(t, "demo.local", routes[0].Host)
	assert.Equal(t, "/api", routes[0].Path)
	assert.Equal(t, router.PathPrefix, routes[0].PathType)
	assert.Equal(t, []string{"10.1.0.1:8080"}, routes[0].Backends[0].Endpoints)
}

func TestGatewayClassFilteringIgnoresOtherControllers(t *testing.T) {
	gw := gateway("default", "gw", "someone-else", "")
	hr := httpRoute("default", "demo", "gw", []string{"demo.local"}, []gatewayv1.HTTPRouteRule{
		backendRule("/", gatewayv1.PathMatchPathPrefix, "web", 80, nil),
	})

	c, _, cancel := newTestController(t, gatewayOptions(), []runtime.Object{
		service("default", "web", 80, "http"),
		endpointSlice("default", "web", "http", 8080, "10.1.0.1"),
	}, []runtime.Object{
		gatewayClass("someone-else", "example.com/other"),
		gw,
		hr,
	})
	defer cancel()

	routes := c.syncGatewayAPI(context.Background(), config.Config, emptyCertMap())

	assert.Empty(t, routes, "a Gateway owned by another controller must be ignored")
}

func TestHostnameIntersection(t *testing.T) {
	cases := []struct {
		route    string
		listener string
		want     string
		ok       bool
	}{
		{"demo.local", "", "demo.local", true},
		{"", "demo.local", "demo.local", true},
		{"demo.local", "demo.local", "demo.local", true},
		{"a.example.com", "*.example.com", "a.example.com", true},
		{"*.example.com", "a.example.com", "a.example.com", true},
		{"a.example.com", "b.example.com", "", false},
		{"a.b.example.com", "*.example.com", "", false},
		{"example.com", "*.example.com", "", false},
	}

	for _, tc := range cases {
		got, ok := intersectHostnames(tc.route, tc.listener)

		assert.Equal(t, tc.ok, ok, "%s vs %s", tc.route, tc.listener)
		assert.Equal(t, tc.want, got, "%s vs %s", tc.route, tc.listener)
	}
}

func TestListenerHostnameNarrowsRouteHostname(t *testing.T) {
	gw := gateway("default", "gw", "gpc", "*.example.com")
	hr := httpRoute("default", "demo", "gw", nil, []gatewayv1.HTTPRouteRule{
		backendRule("/", gatewayv1.PathMatchPathPrefix, "web", 80, nil),
	})

	c, _, cancel := newTestController(t, gatewayOptions(), []runtime.Object{
		service("default", "web", 80, "http"),
		endpointSlice("default", "web", "http", 8080, "10.1.0.1"),
	}, []runtime.Object{
		gatewayClass("gpc", ControllerName),
		gw,
		hr,
	})
	defer cancel()

	routes := c.syncGatewayAPI(context.Background(), config.Config, emptyCertMap())

	assert.Len(t, routes, 1)
	assert.Equal(t, "*.example.com", routes[0].Host, "a route with no hostnames inherits the listener's")
}

func TestWeightedBackendRefs(t *testing.T) {
	stable := int32(90)
	canary := int32(10)

	rule := backendRule("/", gatewayv1.PathMatchPathPrefix, "web", 80, &stable)
	canaryPort := gatewayv1.PortNumber(80)
	rule.BackendRefs = append(rule.BackendRefs, gatewayv1.HTTPBackendRef{
		BackendRef: gatewayv1.BackendRef{
			BackendObjectReference: gatewayv1.BackendObjectReference{
				Name: "web-canary",
				Port: &canaryPort,
			},
			Weight: &canary,
		},
	})

	gw := gateway("default", "gw", "gpc", "")
	hr := httpRoute("default", "demo", "gw", []string{"demo.local"}, []gatewayv1.HTTPRouteRule{rule})

	c, _, cancel := newTestController(t, gatewayOptions(), []runtime.Object{
		service("default", "web", 80, "http"),
		endpointSlice("default", "web", "http", 8080, "10.1.0.1"),
		service("default", "web-canary", 80, "http"),
		endpointSlice("default", "web-canary", "http", 8080, "10.1.0.9"),
	}, []runtime.Object{
		gatewayClass("gpc", ControllerName),
		gw,
		hr,
	})
	defer cancel()

	routes := c.syncGatewayAPI(context.Background(), config.Config, emptyCertMap())

	assert.Len(t, routes, 1)
	assert.Len(t, routes[0].Backends, 2)
	assert.Equal(t, int32(90), routes[0].Backends[0].Weight)
	assert.Equal(t, int32(10), routes[0].Backends[1].Weight)
}

func TestCrossNamespaceRouteRefusedByDefault(t *testing.T) {
	gw := gateway("gateways", "gw", "gpc", "")
	hr := httpRoute("apps", "demo", "gw", []string{"demo.local"}, []gatewayv1.HTTPRouteRule{
		backendRule("/", gatewayv1.PathMatchPathPrefix, "web", 80, nil),
	})

	namespace := gatewayv1.Namespace("gateways")
	hr.Spec.ParentRefs[0].Namespace = &namespace

	c, _, cancel := newTestController(t, gatewayOptions(), []runtime.Object{
		service("apps", "web", 80, "http"),
		endpointSlice("apps", "web", "http", 8080, "10.1.0.1"),
	}, []runtime.Object{
		gatewayClass("gpc", ControllerName),
		gw,
		hr,
	})
	defer cancel()

	routes := c.syncGatewayAPI(context.Background(), config.Config, emptyCertMap())

	assert.Empty(t, routes, "a listener that did not opt in must not admit routes from another namespace")
}

func TestCrossNamespaceRouteAllowedWhenListenerOptsIn(t *testing.T) {
	gw := gateway("gateways", "gw", "gpc", "")
	from := gatewayv1.NamespacesFromAll
	gw.Spec.Listeners[0].AllowedRoutes = &gatewayv1.AllowedRoutes{
		Namespaces: &gatewayv1.RouteNamespaces{From: &from},
	}

	hr := httpRoute("apps", "demo", "gw", []string{"demo.local"}, []gatewayv1.HTTPRouteRule{
		backendRule("/", gatewayv1.PathMatchPathPrefix, "web", 80, nil),
	})

	namespace := gatewayv1.Namespace("gateways")
	hr.Spec.ParentRefs[0].Namespace = &namespace

	c, _, cancel := newTestController(t, gatewayOptions(), []runtime.Object{
		service("apps", "web", 80, "http"),
		endpointSlice("apps", "web", "http", 8080, "10.1.0.1"),
	}, []runtime.Object{
		gatewayClass("gpc", ControllerName),
		gw,
		hr,
	})
	defer cancel()

	routes := c.syncGatewayAPI(context.Background(), config.Config, emptyCertMap())

	assert.Len(t, routes, 1)
}

func TestHTTPRouteFiltersAreTranslated(t *testing.T) {
	replacement := "/"
	rule := backendRule("/api", gatewayv1.PathMatchPathPrefix, "web", 80, nil)
	rule.Filters = []gatewayv1.HTTPRouteFilter{
		{
			Type: gatewayv1.HTTPRouteFilterRequestHeaderModifier,
			RequestHeaderModifier: &gatewayv1.HTTPHeaderFilter{
				Set:    []gatewayv1.HTTPHeader{{Name: "X-Tenant", Value: "acme"}},
				Remove: []string{"X-Internal"},
			},
		},
		{
			Type: gatewayv1.HTTPRouteFilterURLRewrite,
			URLRewrite: &gatewayv1.HTTPURLRewriteFilter{
				Path: &gatewayv1.HTTPPathModifier{
					Type:               gatewayv1.PrefixMatchHTTPPathModifier,
					ReplacePrefixMatch: &replacement,
				},
			},
		},
	}

	gw := gateway("default", "gw", "gpc", "")
	hr := httpRoute("default", "demo", "gw", []string{"demo.local"}, []gatewayv1.HTTPRouteRule{rule})

	c, _, cancel := newTestController(t, gatewayOptions(), []runtime.Object{
		service("default", "web", 80, "http"),
		endpointSlice("default", "web", "http", 8080, "10.1.0.1"),
	}, []runtime.Object{
		gatewayClass("gpc", ControllerName),
		gw,
		hr,
	})
	defer cancel()

	routes := c.syncGatewayAPI(context.Background(), config.Config, emptyCertMap())

	assert.Len(t, routes, 1)
	assert.Len(t, routes[0].Filters, 2)
	assert.Equal(t, router.FilterRequestHeaderModifier, routes[0].Filters[0].Type)
	assert.Equal(t, "acme", routes[0].Filters[0].RequestHeaders.Set["X-Tenant"])
	assert.Equal(t, []string{"X-Internal"}, routes[0].Filters[0].RequestHeaders.Remove)
	assert.Equal(t, router.FilterURLRewrite, routes[0].Filters[1].Type)
}

func TestHTTPRouteMethodAndHeaderMatches(t *testing.T) {
	method := gatewayv1.HTTPMethodPost
	regex := gatewayv1.HeaderMatchRegularExpression

	rule := backendRule("/", gatewayv1.PathMatchPathPrefix, "web", 80, nil)
	rule.Matches[0].Method = &method
	rule.Matches[0].Headers = []gatewayv1.HTTPHeaderMatch{
		{Name: "X-Version", Value: "^v2", Type: &regex},
	}

	gw := gateway("default", "gw", "gpc", "")
	hr := httpRoute("default", "demo", "gw", []string{"demo.local"}, []gatewayv1.HTTPRouteRule{rule})

	c, _, cancel := newTestController(t, gatewayOptions(), []runtime.Object{
		service("default", "web", 80, "http"),
		endpointSlice("default", "web", "http", 8080, "10.1.0.1"),
	}, []runtime.Object{
		gatewayClass("gpc", ControllerName),
		gw,
		hr,
	})
	defer cancel()

	routes := c.syncGatewayAPI(context.Background(), config.Config, emptyCertMap())

	assert.Len(t, routes, 1)
	assert.Equal(t, []string{"POST"}, routes[0].Methods)
	assert.Len(t, routes[0].Headers, 1)
	assert.Equal(t, router.MatchRegex, routes[0].Headers[0].Type)
}
