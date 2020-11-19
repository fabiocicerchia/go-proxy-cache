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

func TestHTTPEndToEndCallRedirect(t *testing.T) {
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
	h := http.HandlerFunc(handler.HandleRequest)

	h.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusMovedPermanently, rr.Code)
	assert.Contains(t, rr.Body.String(), `<title>301 Moved Permanently</title>`)

	tearDownHTTPFunctional()
}

func TestHTTPEndToEndCallWithoutCache(t *testing.T) {
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
	handler := http.HandlerFunc(handler.HandleRequest)

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	assert.Equal(t, "MISS", rr.HeaderMap["X-Go-Proxy-Cache-Status"][0])

	body := rr.Body.String()

	assert.Contains(t, body, "<!DOCTYPE html>\n<html lang=\"en\"")
	assert.Contains(t, body, `<title>Fabio Cicerchia`)
	assert.Contains(t, body, "</body>\n</html>")

	tearDownHTTPFunctional()
}

func TestHTTPEndToEndCallWithCacheMiss(t *testing.T) {
	config.Config = config.Configuration{
		Server: config.Server{
			Forwarding: config.Forward{
				Host:      "developer.mozilla.org",
				Scheme:    "https",
				Endpoints: []string{"developer.mozilla.org"},
			},
		},
		Cache: config.Cache{
			Host:     "localhost",
			Port:     "6379",
			Password: "",
			DB:       0,
		},
	}

	engine.Connect(config.Config.Cache)
	_, err := engine.PurgeAll()
	assert.Nil(t, err)

	req, err := http.NewRequest("GET", "/en-US/docs/Web/HTTP/Headers/Cache-Control", nil)
	assert.Nil(t, err)

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(handler.HandleRequest)

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	assert.Equal(t, "MISS", rr.HeaderMap["X-Go-Proxy-Cache-Status"][0])

	body := rr.Body.String()

	assert.Contains(t, body, "<!DOCTYPE html>\n<html lang=\"en\"")
	assert.Contains(t, body, `<title>Cache-Control - HTTP | MDN</title>`)
	assert.Contains(t, body, "</body>\n</html>")

	tearDownHTTPFunctional()
}

func TestHTTPEndToEndCallWithCacheHit(t *testing.T) {
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
		},
	}

	engine.Connect(config.Config.Cache)

	// --- MISS

	req, err := http.NewRequest("GET", "/en-US/docs/Web/HTTP/Headers/Cache-Control", nil)
	assert.Nil(t, err)

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(handler.HandleRequest)

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	assert.Equal(t, "MISS", rr.HeaderMap["X-Go-Proxy-Cache-Status"][0])

	body := rr.Body.String()

	assert.Contains(t, body, "<!DOCTYPE html>\n<html lang=\"en\"")
	assert.Contains(t, body, `<title>Cache-Control - HTTP | MDN</title>`)
	assert.Contains(t, body, "</body>\n</html>")

	time.Sleep(1 * time.Second)

	// --- HIT

	req, err = http.NewRequest("GET", "/en-US/docs/Web/HTTP/Headers/Cache-Control", nil)
	assert.Nil(t, err)

	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	assert.Equal(t, "HIT", rr.HeaderMap["X-Go-Proxy-Cache-Status"][0])

	body = rr.Body.String()

	assert.Contains(t, body, "<!DOCTYPE html>\n<html lang=\"en\"")
	assert.Contains(t, body, `<title>Cache-Control - HTTP | MDN</title>`)
	assert.Contains(t, body, "</body>\n</html>")

	tearDownHTTPFunctional()
}

func tearDownHTTPFunctional() {
	config.Config = config.Configuration{}
}
