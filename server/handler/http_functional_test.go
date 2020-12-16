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
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"

	"github.com/fabiocicerchia/go-proxy-cache/cache/engine"
	"github.com/fabiocicerchia/go-proxy-cache/config"
	"github.com/fabiocicerchia/go-proxy-cache/server/balancer"
	"github.com/fabiocicerchia/go-proxy-cache/server/handler"
	"github.com/fabiocicerchia/go-proxy-cache/utils"
	circuit_breaker "github.com/fabiocicerchia/go-proxy-cache/utils/circuit-breaker"
)

func setCommonConfig() {
	log.SetReportCaller(true)
	log.SetLevel(log.DebugLevel)

	config.Config = config.Configuration{
		Server: config.Server{
			Upstream: config.Upstream{
				Host:      "www.testing.local",
				Scheme:    "https",
				Endpoints: []string{"127.0.0.1"},
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
	setCommonConfig()
	config.Config.Server.Upstream.Scheme = "http"
	balancer.InitRoundRobin(config.Config.Server.Upstream.Host, config.Config.Server.Upstream.Endpoints)
	circuit_breaker.InitCircuitBreaker(config.Config.Server.Upstream.Host, config.Config.CircuitBreaker)
	engine.InitConn(config.Config.Server.Upstream.Host, config.Config.Cache)

	req, err := http.NewRequest("GET", "/", nil)
	req.URL.Scheme = config.Config.Server.Upstream.Scheme
	req.URL.Host = config.Config.Server.Upstream.Host
	req.Host = config.Config.Server.Upstream.Host
	assert.Nil(t, err)

	rr := httptest.NewRecorder()
	h := http.HandlerFunc(handler.HandleRequest)

	h.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusMovedPermanently, rr.Code)
	assert.Equal(t, "https://testing.local/", rr.HeaderMap["Location"][0])
	assert.Contains(t, rr.Body.String(), `Moved Permanently`)

	tearDownHTTPFunctional()
}

func TestHTTPEndToEndCallWithoutCache(t *testing.T) {
	setCommonConfig()
	config.Config.Server.Upstream.Scheme = "http"
	config.Config.Domains = make(config.Domains)
	conf := config.Config
	config.Config.Server.Upstream = config.Upstream{
		Host:      "www.w3.org",
		Scheme:    "https",
		Endpoints: []string{"www.w3.org"},
	}
	config.Config.Domains["www.w3.org"] = conf

	balancer.InitRoundRobin(config.Config.Server.Upstream.Host, config.Config.Server.Upstream.Endpoints)
	circuit_breaker.InitCircuitBreaker(config.Config.Server.Upstream.Host, config.Config.CircuitBreaker)
	engine.InitConn(config.Config.Server.Upstream.Host, config.Config.Cache)

	engine.GetConn(config.Config.Server.Upstream.Host).Close()

	req, err := http.NewRequest("GET", "/", nil)
	req.URL.Scheme = config.Config.Server.Upstream.Scheme
	req.URL.Host = config.Config.Server.Upstream.Host
	req.Host = config.Config.Server.Upstream.Host
	assert.Nil(t, err)

	rr := httptest.NewRecorder()
	h := http.HandlerFunc(handler.HandleRequest)

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
	setCommonConfig()
	config.Config.Server.Upstream = config.Upstream{
		Host:      "www.w3.org",
		Scheme:    "http",
		Endpoints: []string{"www.w3.org"},
	}

	balancer.InitRoundRobin(config.Config.Server.Upstream.Host, config.Config.Server.Upstream.Endpoints)
	circuit_breaker.InitCircuitBreaker(config.Config.Server.Upstream.Host, config.Config.CircuitBreaker)
	engine.InitConn(config.Config.Server.Upstream.Host, config.Config.Cache)

	_, err := engine.GetConn(config.Config.Server.Upstream.Host).PurgeAll()
	assert.Nil(t, err)

	req, err := http.NewRequest("GET", "/", nil)
	req.URL.Scheme = config.Config.Server.Upstream.Scheme
	req.URL.Host = config.Config.Server.Upstream.Host
	req.Host = config.Config.Server.Upstream.Host
	assert.Nil(t, err)

	rr := httptest.NewRecorder()
	h := http.HandlerFunc(handler.HandleRequest)

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
	setCommonConfig()
	config.Config = config.Configuration{
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
			DB:              0,
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

	balancer.InitRoundRobin(config.Config.Server.Upstream.Host, config.Config.Server.Upstream.Endpoints)
	circuit_breaker.InitCircuitBreaker(config.Config.Server.Upstream.Host, config.Config.CircuitBreaker)
	engine.InitConn(config.Config.Server.Upstream.Host, config.Config.Cache)

	_, _ = engine.GetConn(config.Config.Server.Upstream.Host).PurgeAll()

	time.Sleep(1 * time.Second)

	// --- MISS

	req, err := http.NewRequest("GET", "/", nil)
	req.URL.Scheme = config.Config.Server.Upstream.Scheme
	req.URL.Host = config.Config.Server.Upstream.Host
	req.Host = config.Config.Server.Upstream.Host
	assert.Nil(t, err)

	rr := httptest.NewRecorder()
	h := http.HandlerFunc(handler.HandleRequest)

	h.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	assert.Equal(t, "MISS", rr.HeaderMap["X-Go-Proxy-Cache-Status"][0])

	body := rr.Body.String()

	assert.Contains(t, body, "<!DOCTYPE html PUBLIC")
	assert.Contains(t, body, `<title>World Wide Web Consortium (W3C)</title>`)
	assert.Contains(t, body, "</body>\n</html>\n")

	// --- HIT

	req, err = http.NewRequest("GET", "/", nil)
	req.URL.Scheme = config.Config.Server.Upstream.Scheme
	req.URL.Host = config.Config.Server.Upstream.Host
	req.Host = config.Config.Server.Upstream.Host
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

func TestHTTPEndToEndCallWithHTTPSRedirect(t *testing.T) {
	config.Config = config.Configuration{
		Server: config.Server{
			Upstream: config.Upstream{
				Host:               "testing.local",
				Scheme:             "http",
				Endpoints:          []string{"127.0.0.1"},
				HTTP2HTTPS:         true,
				RedirectStatusCode: http.StatusFound,
			},
		},
	}

	balancer.InitRoundRobin(config.Config.Server.Upstream.Host, config.Config.Server.Upstream.Endpoints)

	req, err := http.NewRequest("GET", "/", nil)
	req.URL.Scheme = config.Config.Server.Upstream.Scheme
	req.URL.Host = config.Config.Server.Upstream.Host
	req.Host = config.Config.Server.Upstream.Host
	assert.Nil(t, err)

	rr := httptest.NewRecorder()
	h := http.HandlerFunc(handler.HandleRequest)

	h.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusFound, rr.Code)

	assert.Equal(t, "https://testing.local/", rr.HeaderMap["Location"][0])

	tearDownHTTPFunctional()
}

func TestHTTPEndToEndCallWithMissingDomain(t *testing.T) {
	setCommonConfig()
	config.Config.Domains = make(config.Domains)
	conf := config.Config
	config.Config.Server.Upstream = config.Upstream{
		Host:      "www.w3.org",
		Scheme:    "http",
		Endpoints: []string{"www.w3.org"},
	}
	config.Config.Domains["www.w3.org"] = conf

	balancer.InitRoundRobin(config.Config.Server.Upstream.Host, config.Config.Server.Upstream.Endpoints)
	circuit_breaker.InitCircuitBreaker(config.Config.Server.Upstream.Host, config.Config.CircuitBreaker)
	engine.InitConn(config.Config.Server.Upstream.Host, config.Config.Cache)

	engine.GetConn(config.Config.Server.Upstream.Host).Close()

	req, err := http.NewRequest("GET", "/", nil)
	req.URL.Scheme = "https"
	req.URL.Host = "www.google.com"
	req.Host = "www.google.com"
	assert.Nil(t, err)

	rr := httptest.NewRecorder()
	h := http.HandlerFunc(handler.HandleRequest)

	h.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusNotImplemented, rr.Code)

	tearDownHTTPFunctional()
}

// --- HTTPS

func TestHTTPSEndToEndCallRedirect(t *testing.T) {
	setCommonConfig()
	// This is because there's no client sending their certificate, so the handshake will be broken with a
	// `remote error: tls: bad certificate`.
	// More details on: https://www.prakharsrivastav.com/posts/from-http-to-https-using-go/
	config.Config.Server.Upstream.InsecureBridge = true

	balancer.InitRoundRobin(config.Config.Server.Upstream.Host, config.Config.Server.Upstream.Endpoints)
	circuit_breaker.InitCircuitBreaker(config.Config.Server.Upstream.Host, config.Config.CircuitBreaker)
	engine.InitConn(config.Config.Server.Upstream.Host, config.Config.Cache)

	req, err := http.NewRequest("GET", "/", nil)
	req.URL.Scheme = config.Config.Server.Upstream.Scheme
	req.URL.Host = config.Config.Server.Upstream.Host
	req.Host = config.Config.Server.Upstream.Host
	assert.Nil(t, err)

	rr := httptest.NewRecorder()
	h := http.HandlerFunc(handler.HandleRequest)

	h.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusMovedPermanently, rr.Code)
	assert.Equal(t, "https://testing.local/", rr.HeaderMap["Location"][0])
	assert.Contains(t, rr.Body.String(), `<title>301 Moved Permanently</title>`)

	tearDownHTTPFunctional()
}

func TestHTTPSEndToEndCallWithoutCache(t *testing.T) {
	setCommonConfig()
	config.Config.Domains = make(config.Domains)
	conf := config.Config
	config.Config.Server.Upstream = config.Upstream{
		Host:      "www.w3.org",
		Scheme:    "https",
		Endpoints: []string{"www.w3.org"},
	}
	config.Config.Domains["www.w3.org"] = conf

	balancer.InitRoundRobin(config.Config.Server.Upstream.Host, config.Config.Server.Upstream.Endpoints)
	circuit_breaker.InitCircuitBreaker(config.Config.Server.Upstream.Host, config.Config.CircuitBreaker)
	engine.InitConn(config.Config.Server.Upstream.Host, config.Config.Cache)

	engine.GetConn(config.Config.Server.Upstream.Host).Close()

	req, err := http.NewRequest("GET", "/", nil)
	req.URL.Scheme = config.Config.Server.Upstream.Scheme
	req.URL.Host = config.Config.Server.Upstream.Host
	req.Host = config.Config.Server.Upstream.Host
	assert.Nil(t, err)

	rr := httptest.NewRecorder()
	h := http.HandlerFunc(handler.HandleRequest)

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
	setCommonConfig()
	config.Config.Server.Upstream = config.Upstream{
		Host:      "www.w3.org",
		Scheme:    "https",
		Endpoints: []string{"www.w3.org"},
	}

	balancer.InitRoundRobin(config.Config.Server.Upstream.Host, config.Config.Server.Upstream.Endpoints)
	circuit_breaker.InitCircuitBreaker(config.Config.Server.Upstream.Host, config.Config.CircuitBreaker)
	engine.InitConn(config.Config.Server.Upstream.Host, config.Config.Cache)

	_, err := engine.GetConn(config.Config.Server.Upstream.Host).PurgeAll()
	assert.Nil(t, err)

	req, err := http.NewRequest("GET", "/", nil)
	req.URL.Scheme = config.Config.Server.Upstream.Scheme
	req.URL.Host = config.Config.Server.Upstream.Host
	req.Host = config.Config.Server.Upstream.Host
	assert.Nil(t, err)

	rr := httptest.NewRecorder()
	h := http.HandlerFunc(handler.HandleRequest)

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
	setCommonConfig()
	config.Config = config.Configuration{
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
			DB:              0,
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

	balancer.InitRoundRobin(config.Config.Server.Upstream.Host, config.Config.Server.Upstream.Endpoints)
	circuit_breaker.InitCircuitBreaker(config.Config.Server.Upstream.Host, config.Config.CircuitBreaker)
	engine.InitConn(config.Config.Server.Upstream.Host, config.Config.Cache)

	_, _ = engine.GetConn(config.Config.Server.Upstream.Host).PurgeAll()

	// --- MISS

	req, err := http.NewRequest("GET", "/", nil)
	req.URL.Scheme = config.Config.Server.Upstream.Scheme
	req.URL.Host = config.Config.Server.Upstream.Host
	req.Host = config.Config.Server.Upstream.Host
	assert.Nil(t, err)

	rr := httptest.NewRecorder()
	h := http.HandlerFunc(handler.HandleRequest)

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
	req.URL.Scheme = config.Config.Server.Upstream.Scheme
	req.URL.Host = config.Config.Server.Upstream.Host
	req.Host = config.Config.Server.Upstream.Host
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
	setCommonConfig()
	config.Config.Domains = make(config.Domains)
	conf := config.Config
	config.Config.Server.Upstream = config.Upstream{
		Host:      "www.w3.org",
		Scheme:    "https",
		Endpoints: []string{"www.w3.org"},
	}
	config.Config.Domains["www.w3.org"] = conf

	balancer.InitRoundRobin(config.Config.Server.Upstream.Host, config.Config.Server.Upstream.Endpoints)
	circuit_breaker.InitCircuitBreaker(config.Config.Server.Upstream.Host, config.Config.CircuitBreaker)
	engine.InitConn(config.Config.Server.Upstream.Host, config.Config.Cache)

	engine.GetConn(config.Config.Server.Upstream.Host).Close()

	req, err := http.NewRequest("GET", "/", nil)
	req.URL.Scheme = "https"
	req.URL.Host = "www.google.com"
	req.Host = "www.google.com"
	assert.Nil(t, err)

	rr := httptest.NewRecorder()
	h := http.HandlerFunc(handler.HandleRequest)

	h.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusNotImplemented, rr.Code)

	tearDownHTTPFunctional()
}

func tearDownHTTPFunctional() {
	config.Config = config.Configuration{}
}
