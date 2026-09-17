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
	"fmt"
	"net"
	"sort"
	"strconv"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/informers"
	inferencev1 "sigs.k8s.io/gateway-api-inference-extension/api/v1"
	inferenceclientset "sigs.k8s.io/gateway-api-inference-extension/client-go/clientset/versioned"
	inferenceinformers "sigs.k8s.io/gateway-api-inference-extension/client-go/informers/externalversions"
	inferencelisters "sigs.k8s.io/gateway-api-inference-extension/client-go/listers/api/v1"

	"github.com/fabiocicerchia/go-proxy-cache/logger"
	"github.com/fabiocicerchia/go-proxy-cache/server/router"
)

// inferenceSyncTimeout - How long to wait for the InferencePool caches.
//
// Long enough for a slow API server, short enough that a cluster without the
// CRDs is not held up: the wait cannot succeed there, so it is spent in full
// on every start.
const inferenceSyncTimeout = 10 * time.Second

// InferencePoolGroup - The API group the Gateway API Inference Extension owns.
const InferencePoolGroup = inferencev1.GroupName

// InferencePoolKind - The only kind of that group usable as a backend.
const InferencePoolKind = "InferencePool"

// inferenceState - The Inference Extension listers.
//
// Nil when the CRDs are absent, which is the usual case: the extension is
// optional and a cluster without it must still serve ordinary routes.
type inferenceState struct {
	pools inferencelisters.InferencePoolLister
	pods  corelistersPodLister
}

// corelistersPodLister - The slice of the Pod lister this file needs.
//
// An InferencePool selects Pods directly rather than going through a Service,
// so unlike every other backend here the endpoints come from Pods.
type corelistersPodLister interface {
	List(selector labels.Selector) ([]*corev1.Pod, error)
}

// resolveInferencePool - The ready Pod addresses backing a pool.
//
// The pool names its own port, so the addresses are pod IP plus that port
// rather than anything a Service decides. Only Pods in the pool's namespace
// are considered; the spec does not allow selecting across namespaces.
func (c *Controller) resolveInferencePool(namespace string, name string) ([]string, string, error) {
	if c.inferenceState == nil || c.inferenceState.pools == nil {
		return nil, "", fmt.Errorf("InferencePool support is not enabled")
	}

	pool, err := c.inferenceState.pools.InferencePools(namespace).Get(name)
	if err != nil {
		return nil, "", err
	}

	selector := inferencePoolSelector(pool)
	if selector == nil {
		return nil, "", fmt.Errorf("InferencePool %s/%s selects nothing", namespace, name)
	}

	pods, err := c.inferenceState.pods.List(selector)
	if err != nil {
		return nil, "", err
	}

	port, err := inferencePoolPort(pool)
	if err != nil {
		return nil, "", err
	}

	endpoints := make([]string, 0, len(pods))

	for _, pod := range pods {
		if pod.Namespace != namespace || pod.Status.PodIP == "" || !isPodReady(pod) {
			continue
		}

		endpoints = append(endpoints, net.JoinHostPort(pod.Status.PodIP, strconv.Itoa(int(port))))
	}

	// Stable order, so an unchanged pool does not look like a change and
	// rebuild every balancer on each resync.
	sort.Strings(endpoints)

	if len(endpoints) == 0 {
		return nil, "", fmt.Errorf("InferencePool %s/%s has no ready endpoints", namespace, name)
	}

	return endpoints, inferencePoolScheme(pool), nil
}

// inferencePoolSelector - The pool's Pod selector.
func inferencePoolSelector(pool *inferencev1.InferencePool) labels.Selector {
	if len(pool.Spec.Selector.MatchLabels) == 0 {
		return nil
	}

	matching := make(labels.Set, len(pool.Spec.Selector.MatchLabels))
	for k, v := range pool.Spec.Selector.MatchLabels {
		matching[string(k)] = string(v)
	}

	return labels.SelectorFromSet(matching)
}

// inferencePoolPort - The port to dial on the selected Pods.
//
// The spec allows up to eight; without an endpoint picker to choose between
// them there is nothing to base a choice on, so the first is used and the rest
// are reported.
func inferencePoolPort(pool *inferencev1.InferencePool) (int32, error) {
	if len(pool.Spec.TargetPorts) == 0 {
		return 0, fmt.Errorf("InferencePool %s/%s declares no target port", pool.Namespace, pool.Name)
	}

	if len(pool.Spec.TargetPorts) > 1 {
		logger.GetGlobal().Warnf(
			"InferencePool %s/%s declares %d target ports; using %d, as choosing between them needs an endpoint picker",
			pool.Namespace, pool.Name, len(pool.Spec.TargetPorts), pool.Spec.TargetPorts[0].Number,
		)
	}

	return int32(pool.Spec.TargetPorts[0].Number), nil
}

// inferencePoolScheme - How to speak to the pool's Pods.
//
// h2c is HTTP/2 over cleartext, which this proxy reaches over plain HTTP.
func inferencePoolScheme(pool *inferencev1.InferencePool) string {
	return "http"
}

// isPodReady - Whether a Pod is running and passing its readiness check.
func isPodReady(pod *corev1.Pod) bool {
	if pod.DeletionTimestamp != nil || pod.Status.Phase != corev1.PodRunning {
		return false
	}

	for i := range pod.Status.Conditions {
		if pod.Status.Conditions[i].Type == corev1.PodReady {
			return pod.Status.Conditions[i].Status == corev1.ConditionTrue
		}
	}

	return false
}

// inferencePoolBackend - An InferencePool backendRef turned into a backend.
func (c *Controller) inferencePoolBackend(namespace string, name string, weight int32) (router.Backend, error) {
	endpoints, scheme, err := c.resolveInferencePool(namespace, name)
	if err != nil {
		return router.Backend{}, err
	}

	pool, err := c.inferenceState.pools.InferencePools(namespace).Get(name)
	if err != nil {
		return router.Backend{}, err
	}

	picker, failOpen := c.resolveEndpointPicker(pool)

	return router.Backend{
		Name:                   fmt.Sprintf("%s/%s (InferencePool)", namespace, name),
		Endpoints:              endpoints,
		Scheme:                 scheme,
		Weight:                 weight,
		EndpointPicker:         picker,
		EndpointPickerFailOpen: failOpen,
	}, nil
}

// resolveEndpointPicker - Where to reach the pool's endpoint picker, and what
// to do when it cannot be reached.
//
// The reference is a Service by default, and the picker is addressed through
// it rather than through its pods: it is a singleton scheduler, so the
// ClusterIP's own balancing is exactly right.
func (c *Controller) resolveEndpointPicker(pool *inferencev1.InferencePool) (string, bool) {
	ref := pool.Spec.EndpointPickerRef
	if ref == nil || ref.Name == "" {
		return "", false
	}

	// FailClose unless the pool says otherwise: a picker exists because the
	// endpoints are not interchangeable.
	failOpen := ref.FailureMode == inferencev1.EndpointPickerFailOpen

	group := ""
	if ref.Group != nil {
		group = string(*ref.Group)
	}

	kind := "Service"
	if ref.Kind != "" {
		kind = string(ref.Kind)
	}

	if group != "" || kind != "Service" {
		logger.GetGlobal().Warnf(
			"InferencePool %s/%s: unsupported endpointPickerRef %s/%s; only core Services are supported",
			pool.Namespace, pool.Name, group, kind,
		)

		return "", failOpen
	}

	if ref.Port == nil {
		logger.GetGlobal().Warnf(
			"InferencePool %s/%s: endpointPickerRef names a Service but no port",
			pool.Namespace, pool.Name,
		)

		return "", failOpen
	}

	svc, err := c.services.Services(pool.Namespace).Get(string(ref.Name))
	if err != nil {
		logger.GetGlobal().Warnf(
			"InferencePool %s/%s: cannot resolve endpoint picker %s/%s: %s",
			pool.Namespace, pool.Name, pool.Namespace, ref.Name, err,
		)

		return "", failOpen
	}

	host := svc.Spec.ClusterIP
	if host == "" || host == corev1.ClusterIPNone {
		// A headless picker Service has no address of its own; its DNS name
		// still resolves to the pods behind it.
		host = fmt.Sprintf("%s.%s.svc", svc.Name, svc.Namespace)
	}

	return net.JoinHostPort(host, strconv.Itoa(int(ref.Port.Number))), failOpen
}

// setupInferenceInformers - Watches InferencePools and the Pods they select.
//
// The CRD is optional. When it is absent the informer simply never syncs
// anything, and every InferencePool backendRef is reported unresolvable rather
// than taking the controller down.
func (c *Controller) setupInferenceInformers(
	client inferenceclientset.Interface,
	core informers.SharedInformerFactory,
	opts Options,
) {
	factoryOpts := []inferenceinformers.SharedInformerOption{}
	if opts.WatchNamespace != "" {
		factoryOpts = append(factoryOpts, inferenceinformers.WithNamespace(opts.WatchNamespace))
	}

	factory := inferenceinformers.NewSharedInformerFactoryWithOptions(client, opts.ResyncPeriod, factoryOpts...)

	c.inferenceFactory = factory
	c.inferenceState = &inferenceState{
		pools: factory.Inference().V1().InferencePools().Lister(),
		pods:  core.Core().V1().Pods().Lister(),
	}

	c.watch(
		factory.Inference().V1().InferencePools().Informer(),
		core.Core().V1().Pods().Informer(),
	)
}
