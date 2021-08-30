// +build all functional

package handler_test

//                                                                         __
// .-----.-----.______.-----.----.-----.--.--.--.--.______.----.---.-.----|  |--.-----.
// |  _  |  _  |______|  _  |   _|  _  |_   _|  |  |______|  __|  _  |  __|     |  -__|
// |___  |_____|      |   __|__| |_____|__.__|___  |      |____|___._|____|__|__|_____|
// |_____|            |__|                   |_____|
//
// Copyright (c) 2020 Fabio Cicerchia. https://fabiocicerchia.it. MIT License
// Repo: https://github.com/fabiocicerchia/go-proxy-cache

import (
	"fmt"
	"crypto/tls"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/fabiocicerchia/go-proxy-cache/cache/engine"
	"github.com/fabiocicerchia/go-proxy-cache/config"
	"github.com/fabiocicerchia/go-proxy-cache/server/balancer"
	"github.com/fabiocicerchia/go-proxy-cache/server/handler"
	"github.com/fabiocicerchia/go-proxy-cache/utils"
	circuit_breaker "github.com/fabiocicerchia/go-proxy-cache/utils/circuit-breaker"
)

func getCommonConfig() config.Configuration {
	initLogs()

	return config.Configuration{
		Server: config.Server{
			Upstream: config.Upstream{
				Host:      "www.testing.local",
				Scheme:    "https",
				Endpoints: []string{utils.GetEnv("NGINX_HOST_80", "localhost:40080")},
			},
		},
		Cache: config.Cache{
			Host: utils.GetEnv("REDIS_HOST", "localhost"),
			Port: "6379",
			DB:   0,
		},
		CircuitBreaker: circuit_breaker.CircuitBreaker{
			Threshold:   2,                // after 2nd request, if meet FailureRate goes open.
			FailureRate: 0.5,              // 1 out of 2 fails, or more
			Interval:    time.Duration(1), // clears counts immediately
			Timeout:     time.Duration(1), // clears state immediately
		},
	}
}

// --- HTTP

func TestHTTPEndToEndCallRedirect(t *testing.T) {
	cfg := getCommonConfig()
	cfg.Cache.DB = 1
	cfg.Server.Upstream.Host = "testing.local"
	cfg.Server.Upstream.Scheme = "http"
	cfg.Server.Upstream.HTTP2HTTPS = true
	cfg.Server.Upstream.RedirectStatusCode = 301
	domainID := cfg.Server.Upstream.GetDomainID()
	balancer.InitRoundRobin(domainID, cfg.Server.Upstream.Endpoints)
	circuit_breaker.InitCircuitBreaker(domainID, cfg.CircuitBreaker)
	engine.InitConn(domainID, cfg.Cache)

	req, err := http.NewRequest("GET", "/", nil)
	req.URL.Scheme = cfg.Server.Upstream.Scheme
	req.URL.Host = cfg.Server.Upstream.Host
	req.Host = cfg.Server.Upstream.Host
	assert.Nil(t, err)

	rr := httptest.NewRecorder()
	h := http.HandlerFunc(handler.HandleRequest(cfg))

	h.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusMovedPermanently, rr.Code)
	assert.Equal(t, "https://testing.local/", rr.HeaderMap["Location"][0])
	assert.Contains(t, rr.Body.String(), `Moved Permanently`)

	tearDownHTTPFunctional()
}

func TestHTTPEndToEndCallWithoutCache(t *testing.T) {
	cfg := getCommonConfig()
	cfg.Cache.DB = 2
	cfg.Domains = make(config.Domains)
	conf := cfg
	cfg.Server.Upstream = config.Upstream{
		Host:      "www.w3.org",
		Scheme:    "https",
		Endpoints: []string{"www.w3.org"},
	}
	cfg.Domains["www.w3.org"] = conf

	domainID := cfg.Server.Upstream.GetDomainID()
	balancer.InitRoundRobin(domainID, cfg.Server.Upstream.Endpoints)
	circuit_breaker.InitCircuitBreaker(domainID, cfg.CircuitBreaker)
	engine.InitConn(domainID, cfg.Cache)

	engine.GetConn(domainID).Close()

	req, err := http.NewRequest("GET", "/", nil)
	req.URL.Scheme = cfg.Server.Upstream.Scheme
	req.URL.Host = cfg.Server.Upstream.Host
	req.Host = cfg.Server.Upstream.Host
	assert.Nil(t, err)

	rr := httptest.NewRecorder()
	h := http.HandlerFunc(handler.HandleRequest(cfg))

	h.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	assert.Equal(t, "MISS", rr.HeaderMap["X-Go-Proxy-Cache-Status"][0])

	body := rr.Body.String()

	assert.Contains(t, body, "<!DOCTYPE html PUBLIC")
	assert.Contains(t, body, `<title>World Wide Web Consortium (W3C)</title>`)
	assert.Contains(t, body, "</body>\n</html>\n")

	tearDownHTTPFunctional()
}

func TestHTTPEndToEndCallWithCacheMiss(t *testing.T) {
	cfg := getCommonConfig()
	cfg.Cache.DB = 3
	cfg.Server.Upstream = config.Upstream{
		Host:      "www.w3.org",
		Scheme:    "http",
		Endpoints: []string{"www.w3.org"},
	}

	domainID := cfg.Server.Upstream.GetDomainID()
	balancer.InitRoundRobin(domainID, cfg.Server.Upstream.Endpoints)
	circuit_breaker.InitCircuitBreaker(domainID, cfg.CircuitBreaker)
	engine.InitConn(domainID, cfg.Cache)

	_, err := engine.GetConn(domainID).PurgeAll()
	assert.Nil(t, err)

	req, err := http.NewRequest("GET", "/", nil)
	req.URL.Scheme = cfg.Server.Upstream.Scheme
	req.URL.Host = cfg.Server.Upstream.Host
	req.Host = cfg.Server.Upstream.Host
	assert.Nil(t, err)

	rr := httptest.NewRecorder()
	h := http.HandlerFunc(handler.HandleRequest(cfg))

	h.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	assert.Equal(t, "MISS", rr.HeaderMap["X-Go-Proxy-Cache-Status"][0])

	body := rr.Body.String()

	assert.Contains(t, body, "<!DOCTYPE html PUBLIC")
	assert.Contains(t, body, `<title>World Wide Web Consortium (W3C)</title>`)
	assert.Contains(t, body, "</body>\n</html>\n")

	tearDownHTTPFunctional()
}

func TestHTTPEndToEndCallWithCacheHit(t *testing.T) {
	cfg := config.Configuration{
		Server: config.Server{
			Upstream: config.Upstream{
				Host:      "www.w3.org",
				Scheme:    "http",
				Endpoints: []string{"www.w3.org"},
			},
		},
		Cache: config.Cache{
			Host:            utils.GetEnv("REDIS_HOST", "localhost"),
			Port:            "6379",
			DB:              4,
			AllowedStatuses: []int{200, 301, 302},
			AllowedMethods:  []string{"HEAD", "GET"},
		},
		CircuitBreaker: circuit_breaker.CircuitBreaker{
			Threshold:   2,                // after 2nd request, if meet FailureRate goes open.
			FailureRate: 0.5,              // 1 out of 2 fails, or more
			Interval:    time.Duration(1), // clears counts immediately
			Timeout:     time.Duration(1), // clears state immediately
		},
	}

	domainID := cfg.Server.Upstream.GetDomainID()
	balancer.InitRoundRobin(domainID, cfg.Server.Upstream.Endpoints)
	circuit_breaker.InitCircuitBreaker(domainID, cfg.CircuitBreaker)
	engine.InitConn(domainID, cfg.Cache)

	_, _ = engine.GetConn(domainID).PurgeAll()

	time.Sleep(1 * time.Second)

	// --- MISS

	req, err := http.NewRequest("GET", "/", nil)
	req.URL.Scheme = cfg.Server.Upstream.Scheme
	req.URL.Host = cfg.Server.Upstream.Host
	req.Host = cfg.Server.Upstream.Host
	assert.Nil(t, err)

	rr := httptest.NewRecorder()
	h := http.HandlerFunc(handler.HandleRequest(cfg))

	h.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	assert.Equal(t, "MISS", rr.HeaderMap["X-Go-Proxy-Cache-Status"][0])

	body := rr.Body.String()

	assert.Contains(t, body, "<!DOCTYPE html PUBLIC")
	assert.Contains(t, body, `<title>World Wide Web Consortium (W3C)</title>`)
	assert.Contains(t, body, "</body>\n</html>\n")

	// --- HIT

	req, err = http.NewRequest("GET", "/", nil)
	req.URL.Scheme = cfg.Server.Upstream.Scheme
	req.URL.Host = cfg.Server.Upstream.Host
	req.Host = cfg.Server.Upstream.Host
	assert.Nil(t, err)

	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	assert.Equal(t, "HIT", rr.HeaderMap["X-Go-Proxy-Cache-Status"][0])

	body = rr.Body.String()

	assert.Contains(t, body, "<!DOCTYPE html PUBLIC")
	assert.Contains(t, body, `<title>World Wide Web Consortium (W3C)</title>`)
	assert.Contains(t, body, "</body>\n</html>\n")

	tearDownHTTPFunctional()
}

func TestHTTPEndToEndCallWithCacheStale(t *testing.T) {
	cfg := config.Configuration{
		Server: config.Server{
			Upstream: config.Upstream{
				Host:      "www.w3.org",
				Scheme:    "https",
				Endpoints: []string{"www.w3.org"},
			},
		},
		Cache: config.Cache{
			Host:            utils.GetEnv("REDIS_HOST", "localhost"),
			Port:            "6379",
			DB:              5,
			AllowedStatuses: []int{200, 301, 302},
			AllowedMethods:  []string{"HEAD", "GET"},
		},
		CircuitBreaker: circuit_breaker.CircuitBreaker{
			Threshold:   2,                // after 2nd request, if meet FailureRate goes open.
			FailureRate: 0.5,              // 1 out of 2 fails, or more
			Interval:    time.Duration(1), // clears counts immediately
			Timeout:     time.Duration(1), // clears state immediately
		},
	}

	domainID := cfg.Server.Upstream.GetDomainID()
	balancer.InitRoundRobin(domainID, cfg.Server.Upstream.Endpoints)
	circuit_breaker.InitCircuitBreaker(domainID, cfg.CircuitBreaker)
	engine.InitConn(domainID, cfg.Cache)

	_, _ = engine.GetConn(domainID).PurgeAll()

	time.Sleep(1 * time.Second)

	// --- MISS

	req, err := http.NewRequest("GET", "/standards/", nil)
	req.URL.Scheme = cfg.Server.Upstream.Scheme
	req.URL.Host = cfg.Server.Upstream.Host
	req.Host = cfg.Server.Upstream.Host
	fmt.Println(req.URL)
	assert.Nil(t, err)

	rr := httptest.NewRecorder()
	h := http.HandlerFunc(handler.HandleRequest(cfg))

	h.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	assert.Equal(t, "MISS", rr.HeaderMap["X-Go-Proxy-Cache-Status"][0])

	body := rr.Body.String()

	assert.Contains(t, body, "<!DOCTYPE html PUBLIC")
	assert.Contains(t, body, `<title>Standards - W3C</title>`)
	assert.Contains(t, body, "</div></body></html>\n")

	// --- HIT

	req, err = http.NewRequest("GET", "/standards/", nil)
	req.URL.Scheme = cfg.Server.Upstream.Scheme
	req.URL.Host = cfg.Server.Upstream.Host
	req.Host = cfg.Server.Upstream.Host
	assert.Nil(t, err)

	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	assert.Equal(t, "HIT", rr.HeaderMap["X-Go-Proxy-Cache-Status"][0])

	body = rr.Body.String()

	assert.Contains(t, body, "<!DOCTYPE html PUBLIC")
	assert.Contains(t, body, `<title>Standards - W3C</title>`)
	assert.Contains(t, body, "</div></body></html>\n")

	time.Sleep(10 * time.Second)

	// Manual Timeout All Fresh Keys
	_, _ = engine.GetConn(domainID).DelWildcard("DATA@@GET@@http://www.w3.org/standards/@@*/fresh")
	time.Sleep(10 * time.Second)

	// --- STALE

	req, err = http.NewRequest("GET", "/standards/", nil)
	req.URL.Scheme = cfg.Server.Upstream.Scheme
	req.URL.Host = cfg.Server.Upstream.Host
	req.Host = cfg.Server.Upstream.Host
	assert.Nil(t, err)

	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	assert.Equal(t, "STALE", rr.HeaderMap["X-Go-Proxy-Cache-Status"][0])

	body = rr.Body.String()

	assert.Contains(t, body, "<!DOCTYPE html PUBLIC")
	assert.Contains(t, body, `<title>Standards - W3C</title>`)
	assert.Contains(t, body, "</div></body></html>\n")

	tearDownHTTPFunctional()
}

func TestHTTPEndToEndCallWithHTTPSRedirect(t *testing.T) {
	cfg := config.Configuration{
		Server: config.Server{
			Upstream: config.Upstream{
				Host:               "testing.local",
				Scheme:             "http",
				Endpoints:          []string{utils.GetEnv("NGINX_HOST_80", "localhost:40080")},
				HTTP2HTTPS:         true,
				RedirectStatusCode: http.StatusFound,
			},
		},
	}
	cfg.Cache.DB = 6

	domainID := cfg.Server.Upstream.GetDomainID()
	balancer.InitRoundRobin(domainID, cfg.Server.Upstream.Endpoints)

	req, err := http.NewRequest("GET", "/", nil)
	req.URL.Scheme = cfg.Server.Upstream.Scheme
	req.URL.Host = cfg.Server.Upstream.Host
	req.Host = cfg.Server.Upstream.Host
	assert.Nil(t, err)

	rr := httptest.NewRecorder()
	h := http.HandlerFunc(handler.HandleRequest(cfg))

	h.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusFound, rr.Code)

	assert.Equal(t, "https://testing.local/", rr.HeaderMap["Location"][0])

	tearDownHTTPFunctional()
}

func TestHTTPEndToEndCallWithMissingDomain(t *testing.T) {
	cfg := getCommonConfig()
	cfg.Cache.DB = 7
	cfg.Domains = make(config.Domains)
	conf := cfg
	cfg.Server.Upstream = config.Upstream{
		Host:      "www.w3.org",
		Scheme:    "http",
		Endpoints: []string{"www.w3.org"},
	}
	cfg.Domains["www.w3.org"] = conf

	domainID := cfg.Server.Upstream.GetDomainID()
	balancer.InitRoundRobin(domainID, cfg.Server.Upstream.Endpoints)
	circuit_breaker.InitCircuitBreaker(domainID, cfg.CircuitBreaker)
	engine.InitConn(domainID, cfg.Cache)

	engine.GetConn(domainID).Close()

	req, err := http.NewRequest("GET", "/", nil)
	req.URL.Scheme = "https"
	req.URL.Host = "www.google.com"
	req.Host = "www.google.com"
	assert.Nil(t, err)

	rr := httptest.NewRecorder()
	h := http.HandlerFunc(handler.HandleRequest(cfg))

	h.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusNotImplemented, rr.Code)

	tearDownHTTPFunctional()
}

// --- HTTPS

func TestHTTPSEndToEndCallRedirect(t *testing.T) {
	cfg := getCommonConfig()
	cfg.Cache.DB = 8
	cfg.Server.Upstream.Host = "testing.local"
	cfg.Server.Upstream.Scheme = "http"
	cfg.Server.Upstream.HTTP2HTTPS = true
	cfg.Server.Upstream.RedirectStatusCode = 301
	cfg.Server.Upstream.Endpoints = []string{utils.GetEnv("NGINX_HOST_443", "localhost:40443")}
	// This is because there's no client sending their certificate, so the handshake will be broken with a
	// `remote error: tls: bad certificate`.
	// More details on: https://www.prakharsrivastav.com/posts/from-http-to-https-using-go/
	cfg.Server.Upstream.InsecureBridge = true

	domainID := cfg.Server.Upstream.GetDomainID()
	balancer.InitRoundRobin(domainID, cfg.Server.Upstream.Endpoints)
	circuit_breaker.InitCircuitBreaker(domainID, cfg.CircuitBreaker)
	engine.InitConn(domainID, cfg.Cache)

	req, err := http.NewRequest("GET", "/", nil)
	req.URL.Scheme = cfg.Server.Upstream.Scheme
	req.URL.Host = cfg.Server.Upstream.Host
	req.Host = cfg.Server.Upstream.Host
	assert.Nil(t, err)

	rr := httptest.NewRecorder()
	h := http.HandlerFunc(handler.HandleRequest(cfg))

	h.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusMovedPermanently, rr.Code)
	assert.Equal(t, "https://testing.local/", rr.HeaderMap["Location"][0])
	assert.Contains(t, rr.Body.String(), `Moved Permanently`)

	tearDownHTTPFunctional()
}

func TestHTTPSEndToEndCallWithoutCache(t *testing.T) {
	cfg := getCommonConfig()
	cfg.Cache.DB = 9
	cfg.Domains = make(config.Domains)
	conf := cfg
	cfg.Server.Upstream = config.Upstream{
		Host:      "www.w3.org",
		Scheme:    "https",
		Endpoints: []string{"www.w3.org"},
	}
	cfg.Domains["www.w3.org"] = conf

	domainID := cfg.Server.Upstream.GetDomainID()
	balancer.InitRoundRobin(domainID, cfg.Server.Upstream.Endpoints)
	circuit_breaker.InitCircuitBreaker(domainID, cfg.CircuitBreaker)
	engine.InitConn(domainID, cfg.Cache)

	engine.GetConn(domainID).Close()

	req, err := http.NewRequest("GET", "/", nil)
	req.URL.Scheme = cfg.Server.Upstream.Scheme
	req.URL.Host = cfg.Server.Upstream.Host
	req.Host = cfg.Server.Upstream.Host
	assert.Nil(t, err)

	rr := httptest.NewRecorder()
	h := http.HandlerFunc(handler.HandleRequest(cfg))

	h.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	assert.Equal(t, "MISS", rr.HeaderMap["X-Go-Proxy-Cache-Status"][0])

	body := rr.Body.String()

	assert.Contains(t, body, "<!DOCTYPE html PUBLIC")
	assert.Contains(t, body, `<title>World Wide Web Consortium (W3C)</title>`)
	assert.Contains(t, body, "</body>\n</html>\n")

	tearDownHTTPFunctional()
}

func TestHTTPSEndToEndCallWithCacheMiss(t *testing.T) {
	cfg := getCommonConfig()
	cfg.Cache.DB = 10
	cfg.Server.Upstream = config.Upstream{
		Host:      "www.w3.org",
		Scheme:    "https",
		Endpoints: []string{"www.w3.org"},
	}

	domainID := cfg.Server.Upstream.GetDomainID()
	balancer.InitRoundRobin(domainID, cfg.Server.Upstream.Endpoints)
	circuit_breaker.InitCircuitBreaker(domainID, cfg.CircuitBreaker)
	engine.InitConn(domainID, cfg.Cache)

	_, err := engine.GetConn(domainID).PurgeAll()
	assert.Nil(t, err)

	req, err := http.NewRequest("GET", "/", nil)
	req.URL.Scheme = cfg.Server.Upstream.Scheme
	req.URL.Host = cfg.Server.Upstream.Host
	req.Host = cfg.Server.Upstream.Host
	assert.Nil(t, err)

	rr := httptest.NewRecorder()
	h := http.HandlerFunc(handler.HandleRequest(cfg))

	h.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	assert.Equal(t, "MISS", rr.HeaderMap["X-Go-Proxy-Cache-Status"][0])

	body := rr.Body.String()

	assert.Contains(t, body, "<!DOCTYPE html PUBLIC")
	assert.Contains(t, body, `<title>World Wide Web Consortium (W3C)</title>`)
	assert.Contains(t, body, "</body>\n</html>\n")

	tearDownHTTPFunctional()
}

func TestHTTPSEndToEndCallWithCacheHit(t *testing.T) {
	cfg := config.Configuration{
		Server: config.Server{
			Upstream: config.Upstream{
				Host:      "www.w3.org",
				Scheme:    "https",
				Endpoints: []string{"www.w3.org"},
			},
		},
		Cache: config.Cache{
			Host:            utils.GetEnv("REDIS_HOST", "localhost"),
			Port:            "6379",
			DB:              11,
			AllowedStatuses: []int{200, 301, 302},
			AllowedMethods:  []string{"HEAD", "GET"},
		},
		CircuitBreaker: circuit_breaker.CircuitBreaker{
			Threshold:   2,                // after 2nd request, if meet FailureRate goes open.
			FailureRate: 0.5,              // 1 out of 2 fails, or more
			Interval:    time.Duration(1), // clears counts immediately
			Timeout:     time.Duration(1), // clears state immediately
		},
	}

	domainID := cfg.Server.Upstream.GetDomainID()
	balancer.InitRoundRobin(domainID, cfg.Server.Upstream.Endpoints)
	circuit_breaker.InitCircuitBreaker(domainID, cfg.CircuitBreaker)
	engine.InitConn(domainID, cfg.Cache)

	_, _ = engine.GetConn(domainID).PurgeAll()

	// --- MISS

	req, err := http.NewRequest("GET", "/", nil)
	req.URL.Scheme = cfg.Server.Upstream.Scheme
	req.URL.Host = cfg.Server.Upstream.Host
	req.Host = cfg.Server.Upstream.Host
	req.TLS = &tls.ConnectionState{} // mock a fake https
	assert.Nil(t, err)

	rr := httptest.NewRecorder()
	h := http.HandlerFunc(handler.HandleRequest(cfg))

	h.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	assert.Equal(t, "MISS", rr.HeaderMap["X-Go-Proxy-Cache-Status"][0])

	body := rr.Body.String()

	assert.Contains(t, body, "<!DOCTYPE html PUBLIC")
	assert.Contains(t, body, `<title>World Wide Web Consortium (W3C)</title>`)
	assert.Contains(t, body, "</body>\n</html>\n")

	time.Sleep(1 * time.Second)

	// --- HIT

	req, err = http.NewRequest("GET", "/", nil)
	req.URL.Scheme = cfg.Server.Upstream.Scheme
	req.URL.Host = cfg.Server.Upstream.Host
	req.Host = cfg.Server.Upstream.Host
	req.TLS = &tls.ConnectionState{} // mock a fake https
	assert.Nil(t, err)

	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	assert.Equal(t, "HIT", rr.HeaderMap["X-Go-Proxy-Cache-Status"][0])

	body = rr.Body.String()

	assert.Contains(t, body, "<!DOCTYPE html PUBLIC")
	assert.Contains(t, body, `<title>World Wide Web Consortium (W3C)</title>`)
	assert.Contains(t, body, "</body>\n</html>\n")

	tearDownHTTPFunctional()
}

func TestHTTPSEndToEndCallWithMissingDomain(t *testing.T) {
	cfg := getCommonConfig()
	cfg.Cache.DB = 12
	cfg.Domains = make(config.Domains)
	conf := cfg
	cfg.Server.Upstream = config.Upstream{
		Host:      "www.w3.org",
		Scheme:    "https",
		Endpoints: []string{"www.w3.org"},
	}
	cfg.Domains["www.w3.org"] = conf

	domainID := cfg.Server.Upstream.GetDomainID()
	balancer.InitRoundRobin(domainID, cfg.Server.Upstream.Endpoints)
	circuit_breaker.InitCircuitBreaker(domainID, cfg.CircuitBreaker)
	engine.InitConn(domainID, cfg.Cache)

	engine.GetConn(domainID).Close()

	req, err := http.NewRequest("GET", "/", nil)
	req.URL.Scheme = "https"
	req.URL.Host = "www.google.com"
	req.Host = "www.google.com"
	assert.Nil(t, err)

	rr := httptest.NewRecorder()
	h := http.HandlerFunc(handler.HandleRequest(cfg))

	h.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusNotImplemented, rr.Code)

	tearDownHTTPFunctional()
}

func tearDownHTTPFunctional() {
	config.Config = config.Configuration{}
}
