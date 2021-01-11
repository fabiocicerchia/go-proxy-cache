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
	"crypto/tls"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/fabiocicerchia/go-proxy-cache/cache/engine"
	"github.com/fabiocicerchia/go-proxy-cache/config"
	"github.com/fabiocicerchia/go-proxy-cache/server/balancer"
	"github.com/fabiocicerchia/go-proxy-cache/server/handler"
	"github.com/fabiocicerchia/go-proxy-cache/utils"
	circuit_breaker "github.com/fabiocicerchia/go-proxy-cache/utils/circuit-breaker"
	"github.com/stretchr/testify/assert"
)

func TestEndToEndHandleWSRequestAndProxy(t *testing.T) {
	initLogs()

	config.Config = config.Configuration{
		Server: config.Server{
			Upstream: config.Upstream{
				Host:      "testing.local",
				Scheme:    "ws",
				Endpoints: []string{utils.GetEnv("NGINX_HOST_WS", "localhost:40081")},
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

	// --- WEBSOCKET

	req, err := http.NewRequest("GET", "/", nil)
	req.URL.Scheme = config.Config.Server.Upstream.Scheme
	req.URL.Host = config.Config.Server.Upstream.Host
	req.Host = config.Config.Server.Upstream.Host
	req.Header = http.Header{
		"Connection": []string{"upgrade"},
		"Upgrade":    []string{"websocket"},
	}
	assert.Nil(t, err)

	rr := httptest.NewRecorder()
	h := http.HandlerFunc(handler.HandleRequest)

	_, err = engine.GetConn(domainID).PurgeAll()
	assert.Nil(t, err)

	h.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	body := rr.Body.String()

	assert.Equal(t, "", body)
}

func TestEndToEndHandleWSRequestAndProxySecure(t *testing.T) {
	initLogs()

	config.Config = config.Configuration{
		Server: config.Server{
			Upstream: config.Upstream{
				Host:      "testing.local",
				Scheme:    "ws",
				Endpoints: []string{utils.GetEnv("NGINX_HOST_WSS", "localhost:40082")},
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

	// --- WEBSOCKET

	req, err := http.NewRequest("GET", "/", nil)
	req.URL.Scheme = config.Config.Server.Upstream.Scheme
	req.URL.Host = config.Config.Server.Upstream.Host
	req.Host = config.Config.Server.Upstream.Host
	req.TLS = &tls.ConnectionState{} // mock a fake https
	req.Header = http.Header{
		"Connection": []string{"upgrade"},
		"Upgrade":    []string{"websocket"},
	}
	assert.Nil(t, err)

	rr := httptest.NewRecorder()
	h := http.HandlerFunc(handler.HandleRequest)

	_, err = engine.GetConn(domainID).PurgeAll()
	assert.Nil(t, err)

	h.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	body := rr.Body.String()

	assert.Equal(t, "", body)
}
