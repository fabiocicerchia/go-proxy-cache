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
	"github.com/fabiocicerchia/go-proxy-cache/server/handler"
)

func TestHealthcheckWithRedis(t *testing.T) {
	config.Config = config.Configuration{
		Cache: config.Cache{
			Host:     "localhost",
			Port:     "6379",
			Password: "",
			DB:       0,
		},
		CircuitBreaker: config.CircuitBreaker{
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

	config.InitCircuitBreaker(config.Config.CircuitBreaker)

	engine.InitConn("global", config.Config.Cache)

	h.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Body.String(), `HTTP OK`)
	assert.Contains(t, rr.Body.String(), `REDIS OK`)
}

// func TestHealthcheckWithoutRedis(t *testing.T) {
// 	_ = engine.Close()

// 	req, err := http.NewRequest("GET", "/healthcheck", nil)
// 	assert.Nil(t, err)

// 	rr := httptest.NewRecorder()
// 	h := http.HandlerFunc(handler.HandleHealthcheck)

// 	h.ServeHTTP(rr, req)

// 	assert.Equal(t, http.StatusOK, rr.Code)
// 	assert.Contains(t, rr.Body.String(), `HTTP OK`)
// 	assert.NotContains(t, rr.Body.String(), `REDIS OK`)

// 	_ = engine.Connect(config.Config.Cache)
// }
