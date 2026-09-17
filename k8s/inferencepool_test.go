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
	inferencev1 "sigs.k8s.io/gateway-api-inference-extension/api/v1"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"

	"github.com/fabiocicerchia/go-proxy-cache/config"
)

func inferencePool(namespace string, name string, port int32, matchLabels map[string]string) *inferencev1.InferencePool {
	selector := make(map[inferencev1.LabelKey]inferencev1.LabelValue, len(matchLabels))
	for k, v := range matchLabels {
		selector[inferencev1.LabelKey(k)] = inferencev1.LabelValue(v)
	}

	return &inferencev1.InferencePool{
		ObjectMeta: metav1.ObjectMeta{Namespace: namespace, Name: name},
		Spec: inferencev1.InferencePoolSpec{
			Selector:    inferencev1.LabelSelector{MatchLabels: selector},
			TargetPorts: []inferencev1.Port{{Number: inferencev1.PortNumber(port)}},
		},
	}
}

func modelPod(namespace string, name string, ip string, ready bool, podLabels map[string]string) *corev1.Pod {
	status := corev1.ConditionFalse
	if ready {
		status = corev1.ConditionTrue
	}

	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Namespace: namespace, Name: name, Labels: podLabels},
		Status: corev1.PodStatus{
			Phase:      corev1.PodRunning,
			PodIP:      ip,
			Conditions: []corev1.PodCondition{{Type: corev1.PodReady, Status: status}},
		},
	}
}

// inferencePoolRule - An HTTPRoute rule whose backend is an InferencePool
// rather than a Service.
func inferencePoolRule(path string, name string) gatewayv1.HTTPRouteRule {
	pathType := gatewayv1.PathMatchPathPrefix
	pathValue := path
	group := gatewayv1.Group(InferencePoolGroup)
	kind := gatewayv1.Kind(InferencePoolKind)

	return gatewayv1.HTTPRouteRule{
		Matches: []gatewayv1.HTTPRouteMatch{{
			Path: &gatewayv1.HTTPPathMatch{Type: &pathType, Value: &pathValue},
		}},
		BackendRefs: []gatewayv1.HTTPBackendRef{{
			BackendRef: gatewayv1.BackendRef{
				BackendObjectReference: gatewayv1.BackendObjectReference{
					Group: &group,
					Kind:  &kind,
					Name:  gatewayv1.ObjectName(name),
				},
			},
		}},
	}
}

func TestInferencePoolBackendResolvesSelectedPods(t *testing.T) {
	gw := gateway("default", "gw", "gpc", "")
	hr := httpRoute("default", "llm", "gw", []string{"llm.local"}, []gatewayv1.HTTPRouteRule{
		inferencePoolRule("/v1", "vllm"),
	})

	labelSet := map[string]string{"app": "vllm"}

	c, _, cancel := newTestController(t, gatewayOptions(), []runtime.Object{
		modelPod("default", "vllm-0", "10.2.0.1", true, labelSet),
		modelPod("default", "vllm-1", "10.2.0.2", true, labelSet),
		// Not ready, so not a candidate.
		modelPod("default", "vllm-2", "10.2.0.3", false, labelSet),
		// Ready, but not selected.
		modelPod("default", "other", "10.2.0.4", true, map[string]string{"app": "something-else"}),
	}, []runtime.Object{
		gatewayClass("gpc", ControllerName),
		gw,
		hr,
		inferencePool("default", "vllm", 8000, labelSet),
	})
	defer cancel()

	routes := c.syncGatewayAPI(context.Background(), config.Config, emptyCertMap())

	assert.Len(t, routes, 1)
	assert.Len(t, routes[0].Backends, 1)

	// The pool names its own port, so the Service port plays no part.
	assert.Equal(t, []string{"10.2.0.1:8000", "10.2.0.2:8000"}, routes[0].Backends[0].Endpoints)
	assert.Equal(t, "http", routes[0].Backends[0].Scheme)
}

// A pool whose Pods are all unready has nothing to route to, and the rule is
// dropped rather than being published with an empty backend.
func TestInferencePoolWithNoReadyPodsIsNotRouted(t *testing.T) {
	gw := gateway("default", "gw", "gpc", "")
	hr := httpRoute("default", "llm", "gw", []string{"llm.local"}, []gatewayv1.HTTPRouteRule{
		inferencePoolRule("/v1", "vllm"),
	})

	labelSet := map[string]string{"app": "vllm"}

	c, _, cancel := newTestController(t, gatewayOptions(), []runtime.Object{
		modelPod("default", "vllm-0", "10.2.0.1", false, labelSet),
	}, []runtime.Object{
		gatewayClass("gpc", ControllerName),
		gw,
		hr,
		inferencePool("default", "vllm", 8000, labelSet),
	})
	defer cancel()

	routes := c.syncGatewayAPI(context.Background(), config.Config, emptyCertMap())

	assert.Empty(t, routes)
}

// The CRDs are optional. Without them the pool cannot resolve, and that must
// cost only the routes that referenced it.
func TestInferencePoolDegradesWhenTheCRDIsAbsent(t *testing.T) {
	gw := gateway("default", "gw", "gpc", "")

	pooled := httpRoute("default", "llm", "gw", []string{"llm.local"}, []gatewayv1.HTTPRouteRule{
		inferencePoolRule("/v1", "vllm"),
	})
	ordinary := httpRoute("default", "web", "gw", []string{"web.local"}, []gatewayv1.HTTPRouteRule{
		backendRule("/", gatewayv1.PathMatchPathPrefix, "web", 80, nil),
	})

	c, _, cancel := newTestController(t, gatewayOptions(), []runtime.Object{
		service("default", "web", 80, "http"),
		endpointSlice("default", "web", "http", 8080, "10.1.0.1"),
	}, []runtime.Object{
		gatewayClass("gpc", ControllerName),
		gw,
		pooled,
		ordinary,
	})
	defer cancel()

	// What Run() does when the Inference Extension caches never sync.
	c.inferenceState = nil

	routes := c.syncGatewayAPI(context.Background(), config.Config, emptyCertMap())

	assert.Len(t, routes, 1, "the ordinary route must survive a missing InferencePool CRD")
	assert.Equal(t, "web.local", routes[0].Host)
}

func TestInferencePoolPortSelection(t *testing.T) {
	single := inferencePool("default", "one", 8000, map[string]string{"app": "vllm"})

	port, err := inferencePoolPort(single)
	assert.Nil(t, err)
	assert.Equal(t, int32(8000), port)

	// Choosing between several needs an endpoint picker, so the first is used.
	many := inferencePool("default", "many", 8000, map[string]string{"app": "vllm"})
	many.Spec.TargetPorts = append(many.Spec.TargetPorts, inferencev1.Port{Number: 8001})

	port, err = inferencePoolPort(many)
	assert.Nil(t, err)
	assert.Equal(t, int32(8000), port)

	none := inferencePool("default", "none", 8000, map[string]string{"app": "vllm"})
	none.Spec.TargetPorts = nil

	_, err = inferencePoolPort(none)
	assert.NotNil(t, err)
}

// A pool is not a way to reach another namespace's Pods: the spec scopes the
// selector to the pool's own namespace.
func TestInferencePoolIgnoresPodsInOtherNamespaces(t *testing.T) {
	labelSet := map[string]string{"app": "vllm"}

	c, _, cancel := newTestController(t, gatewayOptions(), []runtime.Object{
		modelPod("default", "mine", "10.2.0.1", true, labelSet),
		modelPod("other", "theirs", "10.9.9.9", true, labelSet),
	}, []runtime.Object{
		inferencePool("default", "vllm", 8000, labelSet),
	})
	defer cancel()

	endpoints, _, err := c.resolveInferencePool("default", "vllm")

	assert.Nil(t, err)
	assert.Equal(t, []string{"10.2.0.1:8000"}, endpoints)
}
