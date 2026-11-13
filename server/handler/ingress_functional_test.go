//go:build all || functional
// +build all functional

package handler_test

//                                                                         __
// .-----.-----.______.-----.----.-----.--.--.--.--.______.----.---.-.----|  |--.-----.
// |  _  |  _  |______|  _  |   _|  _  |_   _|  |  |______|  __|  _  |  __|     |  -__|
// |___  |_____|      |   __|__| |_____|__.__|___  |      |____|___._|____|__|__|_____|
// |_____|            |__|                   |_____|
//
// Copyright (c) 2023 Fabio Cicerchia. https://fabiocicerchia.it. MIT License
// Repo: https://github.com/fabiocicerchia/go-proxy-cache

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"

	"github.com/fabiocicerchia/go-proxy-cache/cache/engine"
	"github.com/fabiocicerchia/go-proxy-cache/config"
	"github.com/fabiocicerchia/go-proxy-cache/server/balancer"
	"github.com/fabiocicerchia/go-proxy-cache/server/handler"
	"github.com/fabiocicerchia/go-proxy-cache/server/response"
	"github.com/fabiocicerchia/go-proxy-cache/server/router"
	"github.com/fabiocicerchia/go-proxy-cache/utils"
	circuitbreaker "github.com/fabiocicerchia/go-proxy-cache/utils/circuit-breaker"
)

// routedTestConfig - A configuration equivalent to what the ingress controller
// publishes: a global config, no domains, caching against the local Redis.
func routedTestConfig() config.Configuration {
	cfg := config.Config
	cfg.Domains = nil
	cfg.Server.Port.HTTP = "80"
	cfg.Server.Port.HTTPS = "443"
	cfg.Server.Upstream.Scheme = "http"
	cfg.Cache.Hosts = []string{utils.GetEnv("REDIS_HOSTS", "localhost:6379")}
	cfg.Cache.AllowedStatuses = []int{200, 301, 302}
	cfg.Cache.AllowedMethods = []string{"HEAD", "GET"}

	return cfg
}

// serveRouted - Runs one request through the full handler with a routing table
// in place, exactly as the ingress controller would.
func serveRouted(t *testing.T, routes []*router.Route, req *http.Request) *httptest.ResponseRecorder {
	t.Helper()

	router.Enable()
	router.Publish(router.Build(routes))

	res := httptest.NewRecorder()
	handler.HandleRequest(res, req)

	return res
}

func setupRoutedEngine(t *testing.T, cfg config.Configuration) {
	t.Helper()

	domainID := cfg.Server.Upstream.GetDomainID()
	engine.InitConn(domainID, cfg.Cache, log.StandardLogger())
	engine.GetConn(domainID).Close()
	engine.InitConn(domainID, cfg.Cache, log.StandardLogger())
}

func backendEndpoint(server *httptest.Server) string {
	return strings.TrimPrefix(server.URL, "http://")
}

func TestRoutedRequestReachesTheMatchedBackend(t *testing.T) {
	defer router.Reset()

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		_, _ = w.Write([]byte("path=" + r.URL.Path))
	}))
	defer upstream.Close()

	cfg := routedTestConfig()
	setupRoutedEngine(t, cfg)

	route := &router.Route{
		ID:           "test/api",
		Host:         "demo.local",
		Path:         "/api",
		PathType:     router.PathPrefix,
		Config:       cfg,
		PreserveHost: true,
		Backends: []router.Backend{{
			Endpoints: []string{backendEndpoint(upstream)},
			Scheme:    "http",
			Weight:    1,
		}},
	}

	balancer.Init(route.BalancerID(0), route.Upstream(0))

	req := httptest.NewRequest(http.MethodGet, "/api/items", nil)
	req.Host = "demo.local"
	req.TLS = nil

	res := serveRouted(t, []*router.Route{route}, req)

	assert.Equal(t, http.StatusOK, res.Code)
	assert.Equal(t, "path=/api/items", res.Body.String())
}

func TestRoutedRequestPreservesClientHost(t *testing.T) {
	defer router.Reset()

	seen := make(chan string, 1)
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen <- r.Host
		_, _ = w.Write([]byte("ok"))
	}))
	defer upstream.Close()

	cfg := routedTestConfig()
	setupRoutedEngine(t, cfg)

	route := &router.Route{
		ID:           "test/host",
		Host:         "demo.local",
		Path:         "/",
		PathType:     router.PathPrefix,
		Config:       cfg,
		PreserveHost: true,
		Backends: []router.Backend{{
			Endpoints: []string{backendEndpoint(upstream)},
			Scheme:    "http",
			Weight:    1,
		}},
	}

	balancer.Init(route.BalancerID(0), route.Upstream(0))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Host = "demo.local"
	req.TLS = nil

	serveRouted(t, []*router.Route{route}, req)

	// Ingress semantics: the upstream sees the client's Host, not the
	// backend's address (which is what the static configuration would send).
	assert.Equal(t, "demo.local", <-seen)
}

func TestRoutedRequestAppliesRewriteAndHeaderFilters(t *testing.T) {
	defer router.Reset()

	type observed struct {
		path   string
		tenant string
	}

	seen := make(chan observed, 1)
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen <- observed{path: r.URL.Path, tenant: r.Header.Get("X-Tenant")}
		_, _ = w.Write([]byte("ok"))
	}))
	defer upstream.Close()

	cfg := routedTestConfig()
	setupRoutedEngine(t, cfg)

	replacement := "/"
	route := &router.Route{
		ID:           "test/rewrite",
		Host:         "demo.local",
		Path:         "/api",
		PathType:     router.PathPrefix,
		Config:       cfg,
		PreserveHost: true,
		Backends: []router.Backend{{
			Endpoints: []string{backendEndpoint(upstream)},
			Scheme:    "http",
			Weight:    1,
		}},
		Filters: []router.Filter{
			{
				Type:    router.FilterURLRewrite,
				Rewrite: &router.Rewrite{ReplacePrefixMatch: &replacement},
			},
			{
				Type:           router.FilterRequestHeaderModifier,
				RequestHeaders: &router.HeaderOp{Set: map[string]string{"X-Tenant": "acme"}},
			},
		},
	}

	balancer.Init(route.BalancerID(0), route.Upstream(0))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/items", nil)
	req.Host = "demo.local"
	req.TLS = nil

	serveRouted(t, []*router.Route{route}, req)

	got := <-seen
	assert.Equal(t, "/v1/items", got.path, "the matched prefix must be replaced")
	assert.Equal(t, "acme", got.tenant)
}

func TestRoutedRequestRedirectFilterDoesNotProxy(t *testing.T) {
	defer router.Reset()

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("the upstream must not be reached for a redirect route")
	}))
	defer upstream.Close()

	cfg := routedTestConfig()
	setupRoutedEngine(t, cfg)

	route := &router.Route{
		ID:           "test/redirect",
		Host:         "demo.local",
		Path:         "/old",
		PathType:     router.PathPrefix,
		Config:       cfg,
		PreserveHost: true,
		Backends: []router.Backend{{
			Endpoints: []string{backendEndpoint(upstream)},
			Scheme:    "http",
			Weight:    1,
		}},
		Filters: []router.Filter{{
			Type: router.FilterRequestRedirect,
			Redirect: &router.Redirect{
				Hostname:   "new.local",
				StatusCode: http.StatusMovedPermanently,
			},
		}},
	}

	balancer.Init(route.BalancerID(0), route.Upstream(0))

	req := httptest.NewRequest(http.MethodGet, "/old/page?a=1", nil)
	req.Host = "demo.local"
	req.TLS = nil

	res := serveRouted(t, []*router.Route{route}, req)

	assert.Equal(t, http.StatusMovedPermanently, res.Code)
	assert.Equal(t, "http://new.local/old/page?a=1", res.Header().Get("Location"))
}

func TestRoutedRequestIsCached(t *testing.T) {
	defer router.Reset()

	hits := 0
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++

		w.Header().Set("Cache-Control", "max-age=60")
		_, _ = w.Write([]byte("cacheable"))
	}))
	defer upstream.Close()

	cfg := routedTestConfig()
	setupRoutedEngine(t, cfg)

	route := &router.Route{
		ID:           "test/cache",
		Host:         "cache-demo.local",
		Path:         "/",
		PathType:     router.PathPrefix,
		Config:       cfg,
		PreserveHost: true,
		Backends: []router.Backend{{
			Endpoints: []string{backendEndpoint(upstream)},
			Scheme:    "http",
			Weight:    1,
		}},
	}

	circuitbreaker.InitCircuitBreaker(cfg.Server.Upstream.GetDomainID(), cfg.CircuitBreaker, log.StandardLogger())
	balancer.Init(route.BalancerID(0), route.Upstream(0))

	newRequest := func() *http.Request {
		req := httptest.NewRequest(http.MethodGet, "/cacheable-resource", nil)
		req.Host = "cache-demo.local"
		req.TLS = nil

		return req
	}

	// Clear anything a previous run left behind, so the first request is a
	// genuine miss.
	purge := httptest.NewRequest("PURGE", "/cacheable-resource", nil)
	purge.Host = "cache-demo.local"
	purge.TLS = nil
	serveRouted(t, []*router.Route{route}, purge)

	first := serveRouted(t, []*router.Route{route}, newRequest())
	assert.Equal(t, http.StatusOK, first.Code)
	assert.Equal(t, "cacheable", first.Body.String())
	assert.Equal(t, response.CacheStatusHeaderMiss, first.Header().Get(response.CacheStatusHeader))

	second := serveRouted(t, []*router.Route{route}, newRequest())
	assert.Equal(t, http.StatusOK, second.Code)
	assert.Equal(t, "cacheable", second.Body.String())
	assert.Equal(t, response.CacheStatusHeaderHit, second.Header().Get(response.CacheStatusHeader))

	// The whole point: the second request never reached the backend.
	assert.Equal(t, 1, hits, "the cached response must not hit the upstream again")
}

func TestUnmatchedRoutedRequestIs404(t *testing.T) {
	defer router.Reset()

	cfg := routedTestConfig()
	setupRoutedEngine(t, cfg)

	route := &router.Route{
		ID:       "test/api",
		Host:     "demo.local",
		Path:     "/api",
		PathType: router.PathPrefix,
		Config:   cfg,
		Backends: []router.Backend{{Endpoints: []string{"127.0.0.1:1"}, Scheme: "http", Weight: 1}},
	}

	req := httptest.NewRequest(http.MethodGet, "/nope", nil)
	req.Host = "demo.local"
	req.TLS = nil

	res := serveRouted(t, []*router.Route{route}, req)

	// An unmatched request is a 404, the ingress convention, rather than the
	// 501 the static configuration returns for an unknown virtual host.
	assert.Equal(t, http.StatusNotFound, res.Code)
}
