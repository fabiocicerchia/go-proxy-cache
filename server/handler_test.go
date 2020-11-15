package server_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	redis "github.com/fabiocicerchia/go-proxy-cache/cache"
	"github.com/fabiocicerchia/go-proxy-cache/config"
	"github.com/fabiocicerchia/go-proxy-cache/server"
)

// --- UNIT --------------------------------------------------------------------

func TestGetLBRoundRobinUndefined(t *testing.T) {
	setUpHandler()

	var endpoints []string
	endpoint := server.GetLBRoundRobin(endpoints, "8.8.8.8")

	assert.Equal(t, "8.8.8.8", endpoint)

	tearDownHandler()
}

func TestGetLBRoundRobinDefined(t *testing.T) {
	setUpHandler()

	var endpoints = []string{"1.2.3.4"}
	endpoint := server.GetLBRoundRobin(endpoints, "8.8.8.8")

	assert.Equal(t, "1.2.3.4", endpoint)

	tearDownHandler()
}

func setUpHandler() {
	config.Config = config.Configuration{}
}

func tearDownHandler() {
	config.Config = config.Configuration{}
}

// --- FUNCTIONAL --------------------------------------------------------------
const RedisLocalHost = "localhost"

func TestEndToEndCallRedirect(t *testing.T) {
	config.Config = config.Configuration{
		Server: config.Server{
			Forwarding: config.Forward{
				Host:      "www.fabiocicerchia.it",
				Scheme:    "https",
				Endpoints: []string{"www.fabiocicerchia.it"},
			},
		},
	}

	req, err := http.NewRequest("GET", "/", nil)
	assert.Nil(t, err)

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(server.HandleRequestAndRedirect)

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusMovedPermanently, rr.Code)
	assert.Contains(t, rr.Body.String(), `<title>301 Moved Permanently</title>`)

	tearDownFunctional()
}

func TestEndToEndCallWithoutCache(t *testing.T) {
	config.Config = config.Configuration{
		Server: config.Server{
			Forwarding: config.Forward{
				Host:      "fabiocicerchia.it",
				Scheme:    "https",
				Endpoints: []string{"fabiocicerchia.it"},
			},
		},
	}

	req, err := http.NewRequest("GET", "/", nil)
	assert.Nil(t, err)

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(server.HandleRequestAndRedirect)

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	body := rr.Body.String()

	assert.Contains(t, body, "<!DOCTYPE html>\n<html lang=\"en\"")
	assert.Contains(t, body, `<title>Fabio Cicerchia`)
	assert.Contains(t, body, "</body>\n</html>")

	tearDownFunctional()
}

func TestEndToEndCallWithCacheMiss(t *testing.T) {
	config.Config = config.Configuration{
		Server: config.Server{
			Forwarding: config.Forward{
				Host:      "developer.mozilla.org",
				Scheme:    "https",
				Endpoints: []string{"developer.mozilla.org"},
			},
		},
		Cache: config.Cache{
			Host:     RedisLocalHost,
			Port:     "6379",
			Password: "",
			DB:       0,
		},
	}

	redis.Connect(config.Config.Cache)

	req, err := http.NewRequest("GET", "/en-US/docs/Web/HTTP/Headers/Cache-Control", nil)
	assert.Nil(t, err)

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(server.HandleRequestAndRedirect)

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	body := rr.Body.String()

	assert.Contains(t, body, "<!DOCTYPE html>\n<html lang=\"en\"")
	assert.Contains(t, body, `<title>Cache-Control - HTTP | MDN</title>`)
	assert.Contains(t, body, "</body>\n</html>")

	tearDownFunctional()
}

func TestEndToEndCallWithCacheHit(t *testing.T) {
	config.Config = config.Configuration{
		Server: config.Server{
			Forwarding: config.Forward{
				Host:      "developer.mozilla.org",
				Scheme:    "https",
				Endpoints: []string{"developer.mozilla.org"},
			},
		},
		Cache: config.Cache{
			Host:     RedisLocalHost,
			Port:     "6379",
			Password: "",
			DB:       0,
		},
	}

	redis.Connect(config.Config.Cache)

	req, err := http.NewRequest("GET", "/en-US/docs/Web/HTTP/Headers/Cache-Control", nil)
	assert.Nil(t, err)

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(server.HandleRequestAndRedirect)

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	body := rr.Body.String()

	assert.Contains(t, body, "<!DOCTYPE html>\n<html lang=\"en\"")
	assert.Contains(t, body, `<title>Cache-Control - HTTP | MDN</title>`)
	assert.Contains(t, body, "</body>\n</html>")

	tearDownFunctional()
}

func tearDownFunctional() {
	config.Config = config.Configuration{}
}
