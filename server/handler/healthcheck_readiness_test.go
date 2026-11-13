//go:build all || unit
// +build all unit

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
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/fabiocicerchia/go-proxy-cache/config"
	"github.com/fabiocicerchia/go-proxy-cache/server/handler"
	"github.com/fabiocicerchia/go-proxy-cache/server/router"
)

// Listening is not the same as being able to serve. In routed mode the
// listeners come up before the first routing table is published, so without
// this a restarted replica joins the Service while it still 404s everything.
//
// No Redis here, so the healthcheck reports unhealthy either way; what these
// assert is whether the *routes* are what is holding it back.
func TestHealthcheckIsNotReadyBeforeTheFirstRoutingTable(t *testing.T) {
	defer router.Reset()

	router.Enable()

	res := httptest.NewRecorder()
	handler.HandleHealthcheck(config.Config)(res, httptest.NewRequest(http.MethodGet, "/healthcheck", nil))

	assert.Equal(t, http.StatusServiceUnavailable, res.Code)
	assert.Contains(t, res.Body.String(), "ROUTES PENDING")
}

func TestHealthcheckStopsBlockingOnceRoutesArePublished(t *testing.T) {
	defer router.Reset()

	router.Enable()
	router.Publish(router.Build([]*router.Route{{Host: "demo.local", Path: "/"}}))

	res := httptest.NewRecorder()
	handler.HandleHealthcheck(config.Config)(res, httptest.NewRequest(http.MethodGet, "/healthcheck", nil))

	assert.NotEqual(t, http.StatusServiceUnavailable, res.Code)
	assert.NotContains(t, res.Body.String(), "ROUTES PENDING")
}

// The static configuration never publishes a table, so this must not gate it.
func TestHealthcheckIgnoresRoutesOnTheStaticPath(t *testing.T) {
	defer router.Reset()

	res := httptest.NewRecorder()
	handler.HandleHealthcheck(config.Config)(res, httptest.NewRequest(http.MethodGet, "/healthcheck", nil))

	assert.NotEqual(t, http.StatusServiceUnavailable, res.Code)
	assert.NotContains(t, res.Body.String(), "ROUTES PENDING")
}
