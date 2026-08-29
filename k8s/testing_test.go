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
	"time"

	corev1 "k8s.io/api/core/v1"
	discoveryv1 "k8s.io/api/discovery/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
	gatewayfake "sigs.k8s.io/gateway-api/pkg/client/clientset/versioned/fake"

	"github.com/fabiocicerchia/go-proxy-cache/config"
	"github.com/fabiocicerchia/go-proxy-cache/server/router"
	srvtls "github.com/fabiocicerchia/go-proxy-cache/server/tls"
)

// noopRegistry - A BalancerRegistry that records what it was asked for,
// keeping the tests free of real Redis and health-check goroutines.
type noopRegistry struct {
	last map[string]config.Upstream
}

func (r *noopRegistry) Reconcile(wanted map[string]config.Upstream) {
	r.last = wanted
}

// newTestController - A controller backed by fake clientsets, with its
// informer caches synced against the given objects.
func newTestController(t *testing.T, opts Options, coreObjects []runtime.Object, gatewayObjects []runtime.Object) (*Controller, *noopRegistry, context.CancelFunc) {
	t.Helper()

	opts.ResyncPeriod = time.Hour
	opts.DisableStatusUpdates = true

	core := fake.NewSimpleClientset(coreObjects...)
	gateway := gatewayfake.NewClientset()

	registry := &noopRegistry{}
	c := newWithClients(opts, core, gateway, srvtls.NewStore(), registry)

	ctx, cancel := context.WithCancel(context.Background())

	c.factory.Start(ctx.Done())
	c.factory.WaitForCacheSync(ctx.Done())

	if c.gatewayFactory != nil {
		c.gatewayFactory.Start(ctx.Done())
		c.gatewayFactory.WaitForCacheSync(ctx.Done())

		seedGatewayInformers(t, c, gatewayObjects)
	}

	return c, registry, cancel
}

// seedGatewayInformers - Puts Gateway API objects straight into the informer
// indexers.
//
// The generated fake clientset in sigs.k8s.io/gateway-api v1.5.1 silently
// drops Gateway objects -- they are accepted by the tracker but never
// retrievable -- so seeding the caches the informers actually read from is the
// only reliable way to drive the translation logic under test.
func seedGatewayInformers(t *testing.T, c *Controller, objects []runtime.Object) {
	t.Helper()

	for _, obj := range objects {
		var err error

		switch typed := obj.(type) {
		case *gatewayv1.GatewayClass:
			err = c.gatewayFactory.Gateway().V1().GatewayClasses().Informer().GetIndexer().Add(typed)
		case *gatewayv1.Gateway:
			err = c.gatewayFactory.Gateway().V1().Gateways().Informer().GetIndexer().Add(typed)
		case *gatewayv1.HTTPRoute:
			err = c.gatewayFactory.Gateway().V1().HTTPRoutes().Informer().GetIndexer().Add(typed)
		default:
			t.Fatalf("unsupported Gateway API object %T", obj)
		}

		if err != nil {
			t.Fatalf("cannot seed informer with %T: %s", obj, err)
		}
	}
}

func testOptions() Options {
	opts := Options{
		IngressClass:         IngressClassName,
		ControllerName:       ControllerName,
		DisableStatusUpdates: true,
	}
	opts.Normalise()

	return opts
}

func ingressClass(name string, controller string, isDefault bool) *networkingv1.IngressClass {
	class := &networkingv1.IngressClass{
		ObjectMeta: metav1.ObjectMeta{Name: name},
		Spec:       networkingv1.IngressClassSpec{Controller: controller},
	}

	if isDefault {
		class.Annotations = map[string]string{DefaultIngressClassAnnotation: "true"}
	}

	return class
}

func service(namespace string, name string, port int32, portName string) *corev1.Service {
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{Namespace: namespace, Name: name},
		Spec: corev1.ServiceSpec{
			Type:      corev1.ServiceTypeClusterIP,
			ClusterIP: "10.96.0.1",
			Ports:     []corev1.ServicePort{{Name: portName, Port: port}},
		},
	}
}

func endpointSlice(namespace string, serviceName string, portName string, targetPort int32, addresses ...string) *discoveryv1.EndpointSlice {
	ready := true

	endpoints := make([]discoveryv1.Endpoint, 0, len(addresses))
	for _, addr := range addresses {
		endpoints = append(endpoints, discoveryv1.Endpoint{
			Addresses:  []string{addr},
			Conditions: discoveryv1.EndpointConditions{Ready: &ready},
		})
	}

	name := portName

	return &discoveryv1.EndpointSlice{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      serviceName + "-abc",
			Labels:    map[string]string{discoveryv1.LabelServiceName: serviceName},
		},
		AddressType: discoveryv1.AddressTypeIPv4,
		Endpoints:   endpoints,
		Ports:       []discoveryv1.EndpointPort{{Name: &name, Port: &targetPort}},
	}
}

func simpleIngress(namespace string, name string, host string, path string, pathType networkingv1.PathType, svc string, port int32) *networkingv1.Ingress {
	className := IngressClassName

	return &networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{Namespace: namespace, Name: name},
		Spec: networkingv1.IngressSpec{
			IngressClassName: &className,
			Rules: []networkingv1.IngressRule{{
				Host: host,
				IngressRuleValue: networkingv1.IngressRuleValue{
					HTTP: &networkingv1.HTTPIngressRuleValue{
						Paths: []networkingv1.HTTPIngressPath{{
							Path:     path,
							PathType: &pathType,
							Backend: networkingv1.IngressBackend{
								Service: &networkingv1.IngressServiceBackend{
									Name: svc,
									Port: networkingv1.ServiceBackendPort{Number: port},
								},
							},
						}},
					},
				},
			}},
		},
	}
}

func routeByID(routes []*router.Route, id string) *router.Route {
	for _, r := range routes {
		if r.ID == id {
			return r
		}
	}

	return nil
}

func gatewayClass(name string, controller string) *gatewayv1.GatewayClass {
	return &gatewayv1.GatewayClass{
		ObjectMeta: metav1.ObjectMeta{Name: name},
		Spec:       gatewayv1.GatewayClassSpec{ControllerName: gatewayv1.GatewayController(controller)},
	}
}
