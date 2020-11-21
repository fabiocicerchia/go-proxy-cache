// +build functional

package handler_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/fabiocicerchia/go-proxy-cache/cache/engine"
	"github.com/fabiocicerchia/go-proxy-cache/config"
	"github.com/fabiocicerchia/go-proxy-cache/server/balancer"
	"github.com/fabiocicerchia/go-proxy-cache/server/handler"
	"github.com/stretchr/testify/assert"
)

func TestEndToEndCallPurgeDoNothing(t *testing.T) {
	config.Config = config.Configuration{
		Server: config.Server{
			Forwarding: config.Forward{
				Host:      "developer.mozilla.org",
				Scheme:    "https",
				Endpoints: []string{"developer.mozilla.org"},
			},
		},
		Cache: config.Cache{
			Host:            "localhost",
			Port:            "6379",
			Password:        "",
			DB:              0,
			AllowedStatuses: []string{"200", "301", "302"},
			AllowedMethods:  []string{"HEAD", "GET"},
			CircuitBreaker: config.CircuitBreaker{
				Threshold:   2,   // after 2nd request, if meet FailureRate goes open.
				FailureRate: 0.5, // 1 out of 2 fails, or more
				Interval:    time.Duration(1),
				Timeout:     time.Duration(1), // clears state immediately
			},
		},
	}

	engine.InitCircuitBreaker(
		config.Config.Cache.CircuitBreaker.Threshold,
		config.Config.Cache.CircuitBreaker.FailureRate,
		config.Config.Cache.CircuitBreaker.Interval,
		config.Config.Cache.CircuitBreaker.Timeout,
	)
	engine.Connect(config.Config.Cache)

	// --- PURGE

	req, err := http.NewRequest("PURGE", "/", nil)
	assert.Nil(t, err)

	rr := httptest.NewRecorder()
	h := http.HandlerFunc(handler.HandleRequest)

	_, err = engine.PurgeAll()
	assert.Nil(t, err)

	h.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusNotFound, rr.Code)

	body := rr.Body.String()

	assert.Equal(t, body, "KO")

	time.Sleep(1 * time.Second)
}

func TestEndToEndCallPurge(t *testing.T) {
	config.Config = config.Configuration{
		Server: config.Server{
			Forwarding: config.Forward{
				Host:      "developer.mozilla.org",
				Scheme:    "https",
				Endpoints: []string{"developer.mozilla.org"},
			},
		},
		Cache: config.Cache{
			Host:            "localhost",
			Port:            "6379",
			Password:        "",
			DB:              0,
			AllowedStatuses: []string{"200", "301", "302"},
			AllowedMethods:  []string{"HEAD", "GET"},
			CircuitBreaker: config.CircuitBreaker{
				Threshold:   2,   // after 2nd request, if meet FailureRate goes open.
				FailureRate: 0.5, // 1 out of 2 fails, or more
				Interval:    time.Duration(1),
				Timeout:     time.Duration(1), // clears state immediately
			},
		},
	}

	balancer.InitRoundRobin(config.Config.Server.Forwarding.Endpoints)

	engine.InitCircuitBreaker(
		config.Config.Cache.CircuitBreaker.Threshold,
		config.Config.Cache.CircuitBreaker.FailureRate,
		config.Config.Cache.CircuitBreaker.Interval,
		config.Config.Cache.CircuitBreaker.Timeout,
	)
	engine.Connect(config.Config.Cache)

	// --- MISS

	req, err := http.NewRequest("GET", "/en-US/docs/Web/HTTP/Headers/Cache-Control", nil)
	req.URL.Scheme = config.Config.Server.Forwarding.Scheme
	req.URL.Host = config.Config.Server.Forwarding.Host
	req.Host = config.Config.Server.Forwarding.Host
	assert.Nil(t, err)

	rr := httptest.NewRecorder()
	h := http.HandlerFunc(handler.HandleRequest)

	_, err = engine.PurgeAll()
	assert.Nil(t, err)

	h.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	body := rr.Body.String()

	assert.Equal(t, "MISS", rr.HeaderMap["X-Go-Proxy-Cache-Status"][0])

	assert.Contains(t, body, "<!DOCTYPE html>\n<html lang=\"en\"")
	assert.Contains(t, body, `<title>Cache-Control - HTTP | MDN</title>`)
	assert.Contains(t, body, "</body>\n</html>")

	// --- HIT

	req, err = http.NewRequest("GET", "/en-US/docs/Web/HTTP/Headers/Cache-Control", nil)
	req.URL.Scheme = config.Config.Server.Forwarding.Scheme
	req.URL.Host = config.Config.Server.Forwarding.Host
	req.Host = config.Config.Server.Forwarding.Host
	assert.Nil(t, err)

	rr = httptest.NewRecorder()
	h = http.HandlerFunc(handler.HandleRequest)
	h.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	assert.Equal(t, "HIT", rr.HeaderMap["X-Go-Proxy-Cache-Status"][0])

	body = rr.Body.String()

	assert.Contains(t, body, "<!DOCTYPE html>\n<html lang=\"en\"")
	assert.Contains(t, body, `<title>Cache-Control - HTTP | MDN</title>`)
	assert.Contains(t, body, "</body>\n</html>")

	// // --- PURGE

	req, err = http.NewRequest("PURGE", "/en-US/docs/Web/HTTP/Headers/Cache-Control", nil)
	req.URL.Scheme = config.Config.Server.Forwarding.Scheme
	req.URL.Host = config.Config.Server.Forwarding.Host
	req.Host = config.Config.Server.Forwarding.Host
	assert.Nil(t, err)

	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	body = rr.Body.String()

	assert.Equal(t, body, "OK")

	time.Sleep(1 * time.Second)

	// --- MISS

	req, err = http.NewRequest("GET", "/en-US/docs/Web/HTTP/Headers/Cache-Control", nil)
	req.URL.Scheme = config.Config.Server.Forwarding.Scheme
	req.URL.Host = config.Config.Server.Forwarding.Host
	req.Host = config.Config.Server.Forwarding.Host
	assert.Nil(t, err)

	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	assert.Equal(t, "MISS", rr.HeaderMap["X-Go-Proxy-Cache-Status"][0])

	body = rr.Body.String()

	assert.Contains(t, body, "<!DOCTYPE html>\n<html lang=\"en\"")
	assert.Contains(t, body, `<title>Cache-Control - HTTP | MDN</title>`)
	assert.Contains(t, body, "</body>\n</html>")
}
