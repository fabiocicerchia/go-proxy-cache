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

func TestHTTPEndToEndCallRedirect(t *testing.T) {
	config.Config = config.Configuration{
		Server: config.Server{
			Forwarding: config.Forward{
				Host:      "www.fabiocicerchia.it",
				Scheme:    "https",
				Endpoints: []string{"161.35.67.75"},
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
				Endpoints: []string{"161.35.67.75"},
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

	balancer.InitRoundRobin(config.Config.Server.Forwarding.Endpoints)

	engine.Connect(config.Config.Cache)
	_, err := engine.PurgeAll()
	assert.Nil(t, err)

	req, err := http.NewRequest("GET", "/en-US/docs/Web/HTTP/Headers/Cache-Control", nil)
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

	balancer.InitRoundRobin(config.Config.Server.Forwarding.Endpoints)

	engine.Connect(config.Config.Cache)

	// --- MISS

	req, err := http.NewRequest("GET", "/en-US/docs/Web/HTTP/Headers/Cache-Control", nil)
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

	assert.Contains(t, body, "<!DOCTYPE html>\n<html lang=\"en\"")
	assert.Contains(t, body, `<title>Cache-Control - HTTP | MDN</title>`)
	assert.Contains(t, body, "</body>\n</html>")

	time.Sleep(1 * time.Second)

	// --- HIT

	req, err = http.NewRequest("GET", "/en-US/docs/Web/HTTP/Headers/Cache-Control", nil)
	req.URL.Scheme = config.Config.Server.Forwarding.Scheme
	req.URL.Host = config.Config.Server.Forwarding.Host
	req.Host = config.Config.Server.Forwarding.Host
	assert.Nil(t, err)

	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	assert.Equal(t, "HIT", rr.HeaderMap["X-Go-Proxy-Cache-Status"][0])

	body = rr.Body.String()

	assert.Contains(t, body, "<!DOCTYPE html>\n<html lang=\"en\"")
	assert.Contains(t, body, `<title>Cache-Control - HTTP | MDN</title>`)
	assert.Contains(t, body, "</body>\n</html>")

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
