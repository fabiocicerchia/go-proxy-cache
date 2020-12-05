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
	"github.com/fabiocicerchia/go-proxy-cache/server/handler"
	"github.com/fabiocicerchia/go-proxy-cache/utils"
	circuit_breaker "github.com/fabiocicerchia/go-proxy-cache/utils/circuit-breaker"
)

func TestHealthcheckWithoutRedis(t *testing.T) {
	log.SetReportCaller(true)
	log.SetLevel(log.DebugLevel)

	config.Config = config.Configuration{
		Cache: config.Cache{
			Host: utils.GetEnv("REDIS_HOST", "localhost"),
			Port: "6379",
			DB:   0,
		},
		CircuitBreaker: circuit_breaker.CircuitBreaker{
			Threshold:   2,   // after 2nd request, if meet FailureRate goes open.
			FailureRate: 0.5, // 1 out of 2 fails, or more
			Interval:    time.Duration(1),
			Timeout:     time.Duration(1), // clears state immediately
		},
	}

	circuit_breaker.InitCircuitBreaker(config.Config.Server.Upstream.Host, config.Config.CircuitBreaker)

	engine.InitConn(config.Config.Server.Upstream.Host, config.Config.Cache)
	engine.GetConn(config.Config.Server.Upstream.Host).Close()

	req, err := http.NewRequest("GET", "/healthcheck", nil)
	assert.Nil(t, err)

	rr := httptest.NewRecorder()
	h := http.HandlerFunc(handler.HandleHealthcheck)

	h.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Body.String(), `HTTP OK`)
	assert.Contains(t, rr.Body.String(), `REDIS KO`)
	assert.NotContains(t, rr.Body.String(), `REDIS OK`)

	engine.InitConn(config.Config.Server.Upstream.Host, config.Config.Cache)
}

func TestHealthcheckWithRedis(t *testing.T) {
	log.SetReportCaller(true)
	log.SetLevel(log.DebugLevel)

	config.Config = config.Configuration{
		Cache: config.Cache{
			Host: utils.GetEnv("REDIS_HOST", "localhost"),
			Port: "6379",
			DB:   0,
		},
		CircuitBreaker: circuit_breaker.CircuitBreaker{
			Threshold:   2,   // after 2nd request, if meet FailureRate goes open.
			FailureRate: 0.5, // 1 out of 2 fails, or more
			Interval:    time.Duration(1),
			Timeout:     time.Duration(1), // clears state immediately
		},
	}

	req, err := http.NewRequest("GET", "/healthcheck", nil)
	assert.Nil(t, err)

	rr := httptest.NewRecorder()
	h := http.HandlerFunc(handler.HandleHealthcheck)

	circuit_breaker.InitCircuitBreaker(config.Config.Server.Upstream.Host, config.Config.CircuitBreaker)

	engine.InitConn(config.Config.Server.Upstream.Host, config.Config.Cache)

	h.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Body.String(), `HTTP OK`)
	assert.Contains(t, rr.Body.String(), `REDIS OK`)
}
