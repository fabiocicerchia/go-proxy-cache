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

const jwksURL = "https://issuer.example.com/.well-known/jwks.json"

// Splitting one host across several Ingress objects is ordinary practice, not
// an attack. It used to disable JWT for the whole host whenever the object
// without the annotation sorted first, because the middleware resolved the
// settings from the Host header and only the first object for a host was
// published. The settings now travel on the route itself.
func TestJWTSettingsAreCarriedPerRouteNotPerHost(t *testing.T) {
	unprotected := simpleIngress("default", "aaa-static", "api.example.com", "/assets", networkingv1.PathTypePrefix, "web", 80)

	protected := simpleIngress("default", "zzz-api", "api.example.com", "/api", networkingv1.PathTypePrefix, "web", 80)
	protected.Annotations = map[string]string{AnnotationPrefix + AnnJwtJwksURL: jwksURL}

	c, _, cancel := newTestController(t, testOptions(), []runtime.Object{
		ingressClass(IngressClassName, ControllerName, false),
		unprotected,
		protected,
		service("default", "web", 80, "http"),
		endpointSlice("default", "web", "http", 8080, "10.1.0.1"),
	}, nil)
	defer cancel()

	staticRoutes, problems := c.translateIngress(unprotected, config.Config)
	assert.Empty(t, problems)
	assert.Len(t, staticRoutes, 1)

	apiRoutes, problems := c.translateIngress(protected, config.Config)
	assert.Empty(t, problems)
	assert.Len(t, apiRoutes, 1)

	// Same host, different settings: the protected route keeps its JWKS URL
	// even though the unprotected one sorts first.
	assert.Equal(t, "api.example.com", staticRoutes[0].Host)
	assert.Equal(t, "api.example.com", apiRoutes[0].Host)
	assert.Empty(t, staticRoutes[0].Config.Jwt.JwksUrl)
	assert.Equal(t, jwksURL, apiRoutes[0].Config.Jwt.JwksUrl)
	assert.NotNil(t, apiRoutes[0].Config.Jwt.JwkCache, "an annotated route needs a usable JWKS cache")
}

// routedDomains can only hold one entry per host, which is why it must not be
// what authentication is resolved from.
func TestRoutedDomainsKeepsOneEntryPerHost(t *testing.T) {
	first := &router.Route{Host: "api.example.com", Source: "Ingress default/aaa-static"}
	second := &router.Route{Host: "api.example.com", Source: "Ingress default/zzz-api"}
	second.Config.Jwt.JwksUrl = jwksURL

	domains := routedDomains([]*router.Route{first, second})

	assert.Len(t, domains, 1)
	assert.Empty(t, domains["api.example.com"].Jwt.JwksUrl,
		"the first object still wins here, so this is not a safe place to resolve authentication from")
}

// A hostless route (a default backend) has no domain entry to contribute.
func TestRoutedDomainsSkipsHostlessRoutes(t *testing.T) {
	domains := routedDomains([]*router.Route{{Host: "", Source: "Ingress default/fallback"}})

	assert.Empty(t, domains)
}
