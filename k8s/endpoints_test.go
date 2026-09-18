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
	corev1 "k8s.io/api/core/v1"
	discoveryv1 "k8s.io/api/discovery/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func TestResolveByPortName(t *testing.T) {
	c, _, cancel := newTestController(t, testOptions(), []runtime.Object{
		service("default", "web", 80, "http"),
		endpointSlice("default", "web", "http", 8080, "10.1.0.1"),
	}, nil)
	defer cancel()

	got, err := c.backends.resolve("default", "web", intOrName{Name: "http"})

	assert.NoError(t, err)
	assert.Equal(t, []string{"10.1.0.1:8080"}, got)
}

func TestResolveUnknownPort(t *testing.T) {
	c, _, cancel := newTestController(t, testOptions(), []runtime.Object{
		service("default", "web", 80, "http"),
		endpointSlice("default", "web", "http", 8080, "10.1.0.1"),
	}, nil)
	defer cancel()

	_, err := c.backends.resolve("default", "web", intOrName{Number: 9999})

	assert.Error(t, err)
}

func TestResolveUnknownService(t *testing.T) {
	c, _, cancel := newTestController(t, testOptions(), nil, nil)
	defer cancel()

	_, err := c.backends.resolve("default", "nope", intOrName{Number: 80})

	assert.Error(t, err)
}

func TestResolveExternalName(t *testing.T) {
	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "external"},
		Spec: corev1.ServiceSpec{
			Type:         corev1.ServiceTypeExternalName,
			ExternalName: "api.example.com",
			Ports:        []corev1.ServicePort{{Port: 443}},
		},
	}

	c, _, cancel := newTestController(t, testOptions(), []runtime.Object{svc}, nil)
	defer cancel()

	got, err := c.backends.resolve("default", "external", intOrName{Number: 443})

	assert.NoError(t, err)
	assert.Equal(t, []string{"api.example.com:443"}, got)
}

func TestResolveIgnoresFQDNSlices(t *testing.T) {
	slice := endpointSlice("default", "web", "http", 8080, "10.1.0.1")
	slice.AddressType = discoveryv1.AddressTypeFQDN

	c, _, cancel := newTestController(t, testOptions(), []runtime.Object{
		service("default", "web", 80, "http"),
		slice,
	}, nil)
	defer cancel()

	got, err := c.backends.resolve("default", "web", intOrName{Number: 80})

	// No usable slice: falls back to the Service DNS name rather than failing.
	assert.NoError(t, err)
	assert.Equal(t, []string{"web.default.svc:80"}, got)
}

func TestResolveHeadlessServiceWithNoEndpointsFails(t *testing.T) {
	svc := service("default", "web", 80, "http")
	svc.Spec.ClusterIP = corev1.ClusterIPNone

	c, _, cancel := newTestController(t, testOptions(), []runtime.Object{svc}, nil)
	defer cancel()

	_, err := c.backends.resolve("default", "web", intOrName{Number: 80})

	assert.Error(t, err, "a headless service has no DNS fallback worth proxying to")
}

func TestResolveSortsEndpointsForStability(t *testing.T) {
	c, _, cancel := newTestController(t, testOptions(), []runtime.Object{
		service("default", "web", 80, "http"),
		endpointSlice("default", "web", "http", 8080, "10.1.0.9", "10.1.0.2", "10.1.0.5"),
	}, nil)
	defer cancel()

	got, err := c.backends.resolve("default", "web", intOrName{Number: 80})

	assert.NoError(t, err)
	assert.Equal(t, []string{"10.1.0.2:8080", "10.1.0.5:8080", "10.1.0.9:8080"}, got)
}
