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

	"github.com/fabiocicerchia/go-proxy-cache/cache/engine"
	"github.com/fabiocicerchia/go-proxy-cache/config"
	"github.com/fabiocicerchia/go-proxy-cache/server/balancer"
	"github.com/fabiocicerchia/go-proxy-cache/server/handler"
	"github.com/fabiocicerchia/go-proxy-cache/utils"
	circuit_breaker "github.com/fabiocicerchia/go-proxy-cache/utils/circuit-breaker"
	"github.com/stretchr/testify/assert"
)

func TestEndToEndCallPurgeDoNothing(t *testing.T) {
	log.SetReportCaller(true)
	log.SetLevel(log.DebugLevel)

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

	domainID := config.Config.Server.Upstream.Host + utils.StringSeparatorOne + config.Config.Server.Upstream.Scheme
	circuit_breaker.InitCircuitBreaker(domainID, config.Config.CircuitBreaker)
	engine.InitConn(domainID, config.Config.Cache)

	// --- PURGE

	req, err := http.NewRequest("PURGE", "/", nil)
	req.URL.Scheme = config.Config.Server.Upstream.Scheme
	req.URL.Host = config.Config.Server.Upstream.Host
	req.Host = config.Config.Server.Upstream.Host
	assert.Nil(t, err)

	rr := httptest.NewRecorder()
	h := http.HandlerFunc(handler.HandleRequest)

	_, err = engine.GetConn(domainID).PurgeAll()
	assert.Nil(t, err)

	h.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusNotFound, rr.Code)

	body := rr.Body.String()

	assert.Equal(t, body, "KO")

	time.Sleep(1 * time.Second)
}

func TestEndToEndCallPurge(t *testing.T) {
	log.SetReportCaller(true)
	log.SetLevel(log.DebugLevel)

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

	domainID := config.Config.Server.Upstream.Host + utils.StringSeparatorOne + config.Config.Server.Upstream.Scheme
	balancer.InitRoundRobin(domainID, config.Config.Server.Upstream.Endpoints)
	circuit_breaker.InitCircuitBreaker(domainID, config.Config.CircuitBreaker)
	engine.InitConn(domainID, config.Config.Cache)

	// --- MISS

	req, err := http.NewRequest("GET", "/", nil)
	req.URL.Scheme = config.Config.Server.Upstream.Scheme
	req.URL.Host = config.Config.Server.Upstream.Host
	req.Host = config.Config.Server.Upstream.Host
	assert.Nil(t, err)

	rr := httptest.NewRecorder()
	h := http.HandlerFunc(handler.HandleRequest)

	_, err = engine.GetConn(domainID).PurgeAll()
	assert.Nil(t, err)

	h.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	body := rr.Body.String()

	assert.Equal(t, "MISS", rr.HeaderMap["X-Go-Proxy-Cache-Status"][0])

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
	h = http.HandlerFunc(handler.HandleRequest)
	h.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	assert.Equal(t, "HIT", rr.HeaderMap["X-Go-Proxy-Cache-Status"][0])

	body = rr.Body.String()

	assert.Contains(t, body, "<!DOCTYPE html PUBLIC")
	assert.Contains(t, body, `<title>World Wide Web Consortium (W3C)</title>`)
	assert.Contains(t, body, "</body>\n</html>\n")

	// --- PURGE

	req, err = http.NewRequest("PURGE", "/", nil)
	req.URL.Scheme = config.Config.Server.Upstream.Scheme
	req.URL.Host = config.Config.Server.Upstream.Host
	req.Host = config.Config.Server.Upstream.Host
	assert.Nil(t, err)

	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	body = rr.Body.String()

	assert.Equal(t, body, "OK")

	time.Sleep(1 * time.Second)

	// --- MISS

	req, err = http.NewRequest("GET", "/", nil)
	req.URL.Scheme = config.Config.Server.Upstream.Scheme
	req.URL.Host = config.Config.Server.Upstream.Host
	req.Host = config.Config.Server.Upstream.Host
	assert.Nil(t, err)

	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	assert.Equal(t, "MISS", rr.HeaderMap["X-Go-Proxy-Cache-Status"][0])

	body = rr.Body.String()

	assert.Contains(t, body, "<!DOCTYPE html PUBLIC")
	assert.Contains(t, body, `<title>World Wide Web Consortium (W3C)</title>`)
	assert.Contains(t, body, "</body>\n</html>\n")
}
