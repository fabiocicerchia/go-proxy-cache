//go:build all || unit
// +build all unit

package balancer_test

//                                                                         __
// .-----.-----.______.-----.----.-----.--.--.--.--.______.----.---.-.----|  |--.-----.
// |  _  |  _  |______|  _  |   _|  _  |_   _|  |  |______|  __|  _  |  __|     |  -__|
// |___  |_____|      |   __|__| |_____|__.__|___  |      |____|___._|____|__|__|_____|
// |_____|            |__|                   |_____|
//
// Copyright (c) 2023 Fabio Cicerchia. https://fabiocicerchia.it. MIT License
// Repo: https://github.com/fabiocicerchia/go-proxy-cache/

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/fabiocicerchia/go-proxy-cache/config"
	"github.com/fabiocicerchia/go-proxy-cache/server/balancer"
)

// A bare "host:port" endpoint is not a parseable URL, so url.Parse hands back a
// nil URL. Endpoints derived from pod addresses always take that form, and
// dereferencing the nil result killed the process from the healthcheck
// goroutine.
func TestDoHealthCheckAcceptsHostPortEndpoints(t *testing.T) {
	setUp()

	for _, endpoint := range []string{"10.1.2.3:8080", "[fd00::1]:8080", "10.1.2.3"} {
		item := balancer.Item{Endpoint: endpoint}
		conf := config.HealthCheck{Scheme: "http", Port: "80", StatusCodes: []string{"200"}}

		assert.NotPanics(t, func() { balancer.DoHealthCheck(&item, "example.com", conf) }, endpoint)
	}

	tearDown()
}

func TestDoHealthCheckReachesHostPortEndpoint(t *testing.T) {
	setUp()

	var probed string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		probed = r.Host
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	// httptest hands back "http://127.0.0.1:PORT"; strip the scheme so the
	// endpoint is the bare host:port form the ingress controller produces.
	hostPort := strings.TrimPrefix(srv.URL, "http://")

	item := balancer.Item{Endpoint: hostPort}
	conf := config.HealthCheck{Scheme: "http", Port: "80", StatusCodes: []string{strconv.Itoa(http.StatusOK)}}

	balancer.DoHealthCheck(&item, "example.com", conf)

	assert.Equal(t, hostPort, probed)
	assert.True(t, item.Healthy)

	tearDown()
}

// An endpoint that already carries a scheme keeps it, rather than being
// rewritten to the healthcheck default.
func TestDoHealthCheckKeepsExplicitScheme(t *testing.T) {
	setUp()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	item := balancer.Item{Endpoint: srv.URL}
	conf := config.HealthCheck{Scheme: "https", Port: "443", StatusCodes: []string{strconv.Itoa(http.StatusOK)}}

	balancer.DoHealthCheck(&item, "example.com", conf)

	assert.True(t, item.Healthy)

	tearDown()
}
