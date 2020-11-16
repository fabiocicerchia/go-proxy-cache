package handler_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/fabiocicerchia/go-proxy-cache/cache/engine"
	"github.com/fabiocicerchia/go-proxy-cache/config"
	"github.com/fabiocicerchia/go-proxy-cache/server/handler"
)

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
	h := http.HandlerFunc(handler.HandleRequestAndProxy)

	h.ServeHTTP(rr, req)

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
	handler := http.HandlerFunc(handler.HandleRequestAndProxy)

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

	engine.Connect(config.Config.Cache)

	req, err := http.NewRequest("GET", "/en-US/docs/Web/HTTP/Headers/Cache-Control", nil)
	assert.Nil(t, err)

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(handler.HandleRequestAndProxy)

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

	engine.Connect(config.Config.Cache)

	req, err := http.NewRequest("GET", "/en-US/docs/Web/HTTP/Headers/Cache-Control", nil)
	assert.Nil(t, err)

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(handler.HandleRequestAndProxy)

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
