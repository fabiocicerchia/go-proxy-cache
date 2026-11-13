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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"

	"github.com/fabiocicerchia/go-proxy-cache/config"
)

// An HTTPRoute with one working rule and one broken one used to report
// ResolvedRefs=True, because resolution was judged on whether *any* route came
// out. The dropped rule was then invisible in `kubectl describe`.
func TestResolvedRefsIsPerRuleNotPerRoute(t *testing.T) {
	gw := gateway("default", "gw", "gpc", "")
	hr := httpRoute("default", "demo", "gw", []string{"demo.local"}, []gatewayv1.HTTPRouteRule{
		backendRule("/ok", gatewayv1.PathMatchPathPrefix, "web", 80, nil),
		backendRule("/broken", gatewayv1.PathMatchPathPrefix, "missing", 80, nil),
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

	outcome := c.translateHTTPRoute(hr, gw, []*gatewayv1.Listener{&gw.Spec.Listeners[0]}, config.Config)

	assert.Len(t, outcome.Routes, 1, "only the resolvable rule produces a route")
	assert.Equal(t, 1, outcome.ResolvedRules)
	assert.NotEqual(t, len(hr.Spec.Rules), outcome.ResolvedRules,
		"one rule of two resolved, so the route is not fully resolved")
}

func TestResolvedRefsWhenEveryRuleResolves(t *testing.T) {
	gw := gateway("default", "gw", "gpc", "")
	hr := httpRoute("default", "demo", "gw", []string{"demo.local"}, []gatewayv1.HTTPRouteRule{
		backendRule("/a", gatewayv1.PathMatchPathPrefix, "web", 80, nil),
		backendRule("/b", gatewayv1.PathMatchPathPrefix, "web", 80, nil),
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

	outcome := c.translateHTTPRoute(hr, gw, []*gatewayv1.Listener{&gw.Spec.Listeners[0]}, config.Config)

	assert.Equal(t, len(hr.Spec.Rules), outcome.ResolvedRules)
	assert.False(t, outcome.Refused)
}

// The count used to be per Gateway and written onto every listener, so a
// listener serving nothing still reported a non-zero number.
func TestAttachedRoutesAreCountedPerListener(t *testing.T) {
	gw := gateway("default", "gw", "gpc", "")

	busy := gatewayv1.Hostname("busy.local")
	gw.Spec.Listeners[0].Name = "busy"
	gw.Spec.Listeners[0].Hostname = &busy

	idleHost := gatewayv1.Hostname("idle.local")
	gw.Spec.Listeners = append(gw.Spec.Listeners, gatewayv1.Listener{
		Name:     "idle",
		Port:     80,
		Protocol: gatewayv1.HTTPProtocolType,
		Hostname: &idleHost,
	})

	attached := map[string]int32{
		listenerKey(gw, "busy"): 3,
	}

	status := gatewayStatusFor(gw, []string{"10.0.0.1"}, attached, map[string]bool{})

	assert.Len(t, status.Listeners, 2)

	byName := map[gatewayv1.SectionName]int32{}
	for _, listener := range status.Listeners {
		byName[listener.Name] = listener.AttachedRoutes
	}

	assert.Equal(t, int32(3), byName["busy"])
	assert.Equal(t, int32(0), byName["idle"], "a listener with no routes must report zero")
}

func TestGatewayStatusKeepsObservedGeneration(t *testing.T) {
	gw := gateway("default", "gw", "gpc", "")
	gw.ObjectMeta = metav1.ObjectMeta{Namespace: "default", Name: "gw", Generation: 7}

	status := gatewayStatusFor(gw, nil, map[string]int32{}, map[string]bool{})

	assert.NotEmpty(t, status.Conditions)
	assert.Equal(t, int64(7), status.Conditions[0].ObservedGeneration)
}
