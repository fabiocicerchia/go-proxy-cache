//                                                                         __
// .-----.-----.______.-----.----.-----.--.--.--.--.______.----.---.-.----|  |--.-----.
// |  _  |  _  |______|  _  |   _|  _  |_   _|  |  |______|  __|  _  |  __|     |  -__|
// |___  |_____|      |   __|__| |_____|__.__|___  |      |____|___._|____|__|__|_____|
// |_____|            |__|                   |_____|
//
// Copyright (c) 2020 Fabio Cicerchia. https://fabiocicerchia.it. MIT License
// Repo: https://github.com/fabiocicerchia/go-proxy-cache
// +build functional

package handler_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/fabiocicerchia/go-proxy-cache/cache/engine"
	"github.com/fabiocicerchia/go-proxy-cache/config"
	"github.com/fabiocicerchia/go-proxy-cache/server/balancer"
	"github.com/fabiocicerchia/go-proxy-cache/server/handler"
)

func setCommonConfig() {
	config.Config = config.Configuration{
		Server: config.Server{
			Forwarding: config.Forward{
				Host:      "www.fabiocicerchia.it",
				Scheme:    "https",
				Endpoints: []string{"161.35.67.75"},
			},
		},
		Cache: config.Cache{
			Host:     "localhost",
			Port:     "6379",
			Password: "",
			DB:       0,
		},
		CircuitBreaker: config.CircuitBreaker{
			Threshold:   2,                // after 2nd request, if meet FailureRate goes open.
			FailureRate: 0.5,              // 1 out of 2 fails, or more
			Interval:    time.Duration(1), // clears counts immediately
			Timeout:     time.Duration(1), // clears state immediately
		},
	}
}

func TestHTTPEndToEndCallRedirect(t *testing.T) {
	setCommonConfig()
	balancer.InitRoundRobin(config.Config.Server.Forwarding.Endpoints)
	config.InitCircuitBreaker(config.Config.CircuitBreaker)
	engine.InitConn("global", config.Config.Cache)

	req, err := http.NewRequest("GET", "/", nil)
	req.URL.Scheme = config.Config.Server.Forwarding.Scheme
	req.URL.Host = config.Config.Server.Forwarding.Host
	req.Host = config.Config.Server.Forwarding.Host
	assert.Nil(t, err)

	rr := httptest.NewRecorder()
	h := http.HandlerFunc(handler.HandleRequest)

	h.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusMovedPermanently, rr.Code)
	assert.Contains(t, rr.Body.String(), `<title>301 Moved Permanently</title>`)

	tearDownHTTPFunctional()
}

func TestHTTPEndToEndCallWithoutCache(t *testing.T) {
	setCommonConfig()
	config.Config.Domains = make(config.Domains)
	conf := config.Config
	config.Config.Server.Forwarding = config.Forward{
		Host:      "www.w3.org",
		Scheme:    "https",
		Endpoints: []string{"www.w3.org"},
	}
	config.Config.Domains["www.w3.org"] = conf

	balancer.InitRoundRobin(config.Config.Server.Forwarding.Endpoints)
	config.InitCircuitBreaker(config.Config.CircuitBreaker)

	req, err := http.NewRequest("GET", "/", nil)
	req.URL.Scheme = config.Config.Server.Forwarding.Scheme
	req.URL.Host = config.Config.Server.Forwarding.Host
	req.Host = config.Config.Server.Forwarding.Host
	assert.Nil(t, err)

	rr := httptest.NewRecorder()
	h := http.HandlerFunc(handler.HandleRequest)

	h.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	assert.Equal(t, "MISS", rr.HeaderMap["X-Go-Proxy-Cache-Status"][0])

	body := rr.Body.String()

	assert.Contains(t, string(body), "<!DOCTYPE html PUBLIC")
	assert.Contains(t, string(body), `<title>World Wide Web Consortium (W3C)</title>`)
	assert.Contains(t, string(body), "</body>\n</html>\n")

	tearDownHTTPFunctional()
}

func TestHTTPEndToEndCallWithCacheMiss(t *testing.T) {
	setCommonConfig()
	config.Config.Server.Forwarding = config.Forward{
		Host:      "www.w3.org",
		Scheme:    "https",
		Endpoints: []string{"www.w3.org"},
	}

	balancer.InitRoundRobin(config.Config.Server.Forwarding.Endpoints)
	config.InitCircuitBreaker(config.Config.CircuitBreaker)
	engine.InitConn("global", config.Config.Cache)

	_, err := engine.GetConn("global").PurgeAll()
	assert.Nil(t, err)

	req, err := http.NewRequest("GET", "/", nil)
	req.URL.Scheme = config.Config.Server.Forwarding.Scheme
	req.URL.Host = config.Config.Server.Forwarding.Host
	req.Host = config.Config.Server.Forwarding.Host
	assert.Nil(t, err)

	rr := httptest.NewRecorder()
	h := http.HandlerFunc(handler.HandleRequest)

	h.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	assert.Equal(t, "MISS", rr.HeaderMap["X-Go-Proxy-Cache-Status"][0])

	body := rr.Body.String()

	assert.Contains(t, string(body), "<!DOCTYPE html PUBLIC")
	assert.Contains(t, string(body), `<title>World Wide Web Consortium (W3C)</title>`)
	assert.Contains(t, string(body), "</body>\n</html>\n")

	tearDownHTTPFunctional()
}

func TestHTTPEndToEndCallWithCacheHit(t *testing.T) {
	setCommonConfig()
	config.Config = config.Configuration{
		Server: config.Server{
			Forwarding: config.Forward{
				Host:      "www.w3.org",
				Scheme:    "https",
				Endpoints: []string{"www.w3.org"},
			},
		},
		Cache: config.Cache{
			Host:            "localhost",
			Port:            "6379",
			Password:        "",
			DB:              0,
			AllowedStatuses: []int{200, 301, 302},
			AllowedMethods:  []string{"HEAD", "GET"},
		},
		CircuitBreaker: config.CircuitBreaker{
			Threshold:   2,                // after 2nd request, if meet FailureRate goes open.
			FailureRate: 0.5,              // 1 out of 2 fails, or more
			Interval:    time.Duration(1), // clears counts immediately
			Timeout:     time.Duration(1), // clears state immediately
		},
	}

	balancer.InitRoundRobin(config.Config.Server.Forwarding.Endpoints)
	config.InitCircuitBreaker(config.Config.CircuitBreaker)
	engine.InitConn("global", config.Config.Cache)

	// --- MISS

	req, err := http.NewRequest("GET", "/", nil)
	req.URL.Scheme = config.Config.Server.Forwarding.Scheme
	req.URL.Host = config.Config.Server.Forwarding.Host
	req.Host = config.Config.Server.Forwarding.Host
	assert.Nil(t, err)

	rr := httptest.NewRecorder()
	h := http.HandlerFunc(handler.HandleRequest)

	h.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	assert.Equal(t, "MISS", rr.HeaderMap["X-Go-Proxy-Cache-Status"][0])

	body := rr.Body.String()

	assert.Contains(t, string(body), "<!DOCTYPE html PUBLIC")
	assert.Contains(t, string(body), `<title>World Wide Web Consortium (W3C)</title>`)
	assert.Contains(t, string(body), "</body>\n</html>\n")

	time.Sleep(1 * time.Second)

	// --- HIT

	req, err = http.NewRequest("GET", "/", nil)
	req.URL.Scheme = config.Config.Server.Forwarding.Scheme
	req.URL.Host = config.Config.Server.Forwarding.Host
	req.Host = config.Config.Server.Forwarding.Host
	assert.Nil(t, err)

	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	assert.Equal(t, "HIT", rr.HeaderMap["X-Go-Proxy-Cache-Status"][0])

	body = rr.Body.String()

	assert.Contains(t, string(body), "<!DOCTYPE html PUBLIC")
	assert.Contains(t, string(body), `<title>World Wide Web Consortium (W3C)</title>`)
	assert.Contains(t, string(body), "</body>\n</html>\n")

	tearDownHTTPFunctional()
}

func TestHTTPEndToEndCallWithHTTPSRedirect(t *testing.T) {
	config.Config = config.Configuration{
		Server: config.Server{
			Forwarding: config.Forward{
				Host:               "fabiocicerchia.it",
				Scheme:             "http",
				Endpoints:          []string{"161.35.67.75"},
				HTTP2HTTPS:         true,
				RedirectStatusCode: http.StatusFound,
			},
		},
	}

	balancer.InitRoundRobin(config.Config.Server.Forwarding.Endpoints)

	req, err := http.NewRequest("GET", "/", nil)
	req.URL.Scheme = config.Config.Server.Forwarding.Scheme
	req.URL.Host = config.Config.Server.Forwarding.Host
	req.Host = config.Config.Server.Forwarding.Host
	assert.Nil(t, err)

	rr := httptest.NewRecorder()
	h := http.HandlerFunc(handler.HandleRequest)

	h.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusFound, rr.Code)

	// TODO: does it make sense?
	assert.Equal(t, "https://fabiocicerchia.it/", rr.HeaderMap["Location"][0])

	tearDownHTTPFunctional()
}

func tearDownHTTPFunctional() {
	config.Config = config.Configuration{}
}
