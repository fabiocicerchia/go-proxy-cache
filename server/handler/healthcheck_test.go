// +build functional

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

func TestHealthcheckWithRedis(t *testing.T) {
	config.Config = config.Configuration{
		Cache: config.Cache{
			Host:     "localhost",
			Port:     "6379",
			Password: "",
			DB:       0,
		},
	}

	req, err := http.NewRequest("GET", "/healthcheck", nil)
	assert.Nil(t, err)

	rr := httptest.NewRecorder()
	h := http.HandlerFunc(handler.HandleHealthcheck)

	engine.Connect(config.Config.Cache)

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
