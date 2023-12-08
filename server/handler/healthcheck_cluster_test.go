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
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"

	"github.com/fabiocicerchia/go-proxy-cache/cache/engine"
	"github.com/fabiocicerchia/go-proxy-cache/config"
	"github.com/fabiocicerchia/go-proxy-cache/logger"
	"github.com/fabiocicerchia/go-proxy-cache/server/handler"
	"github.com/fabiocicerchia/go-proxy-cache/utils"
	circuit_breaker "github.com/fabiocicerchia/go-proxy-cache/utils/circuit-breaker"
)

func TestClusterHealthcheckWithoutRedis(t *testing.T) {
	initLogs()

	config.Config = config.Configuration{
		Cache: config.Cache{
			Hosts: strings.Split(utils.GetEnv("REDIS_HOSTS", "172.20.0.36:6379,172.20.0.37:6379,172.20.0.38:6379"), ","),
			DB:    0,
		},
		CircuitBreaker: circuit_breaker.CircuitBreaker{
			Threshold:   2,   // after 2nd request, if meet FailureRate goes open.
			FailureRate: 0.5, // 1 out of 2 fails, or more
			Interval:    time.Duration(1),
			Timeout:     time.Duration(1), // clears state immediately
		},
		Server: config.Server{
			Upstream: config.Upstream{
				Host:      "testing.local",
				Scheme:    "https",
				Endpoints: []string{utils.GetEnv("NGINX_HOST_80", "localhost:40080")},
			},
		},
	}

	domainID := config.Config.Server.Upstream.GetDomainID()
	circuit_breaker.InitCircuitBreaker(domainID, config.Config.CircuitBreaker, logger.GetGlobal())
	engine.InitConn(domainID, config.Config.Cache, log.StandardLogger())
	engine.GetConn(domainID).Close()

	req, err := http.NewRequest("GET", "/healthcheck", nil)
	req.Host = "testing.local"
	assert.Nil(t, err)

	rr := httptest.NewRecorder()
	h := http.HandlerFunc(handler.HandleHealthcheck(config.Config))

	h.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Body.String(), `HTTP OK`)
	assert.Contains(t, rr.Body.String(), `REDIS KO`)
	assert.NotContains(t, rr.Body.String(), `REDIS OK`)

	engine.InitConn(domainID, config.Config.Cache, log.StandardLogger())
}

func TestClusterHealthcheckWithRedis(t *testing.T) {
	initLogs()

	config.Config = config.Configuration{
		Cache: config.Cache{
			Hosts: strings.Split(utils.GetEnv("REDIS_HOSTS", "172.20.0.36:6379,172.20.0.37:6379,172.20.0.38:6379"), ","),
			DB:    0,
		},
		CircuitBreaker: circuit_breaker.CircuitBreaker{
			Threshold:   2,   // after 2nd request, if meet FailureRate goes open.
			FailureRate: 0.5, // 1 out of 2 fails, or more
			Interval:    time.Duration(1),
			Timeout:     time.Duration(1), // clears state immediately
		},
		Server: config.Server{
			Upstream: config.Upstream{
				Host:      "testing.local",
				Scheme:    "http",
				Endpoints: []string{utils.GetEnv("NGINX_HOST_80", "localhost:40080")},
			},
		},
	}

	domainID := config.Config.Server.Upstream.GetDomainID()
	circuit_breaker.InitCircuitBreaker(domainID, config.Config.CircuitBreaker, logger.GetGlobal())
	engine.InitConn(domainID, config.Config.Cache, log.StandardLogger())

	req, err := http.NewRequest("GET", "/healthcheck", nil)
	req.Host = "testing.local"
	assert.Nil(t, err)

	rr := httptest.NewRecorder()
	h := http.HandlerFunc(handler.HandleHealthcheck(config.Config))

	h.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Body.String(), `HTTP OK`)
	assert.Contains(t, rr.Body.String(), `REDIS OK`)
}
