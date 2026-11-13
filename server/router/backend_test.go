//go:build all || unit
// +build all unit

package router_test

//                                                                         __
// .-----.-----.______.-----.----.-----.--.--.--.--.______.----.---.-.----|  |--.-----.
// |  _  |  _  |______|  _  |   _|  _  |_   _|  |  |______|  __|  _  |  __|     |  -__|
// |___  |_____|      |   __|__| |_____|__.__|___  |      |____|___._|____|__|__|_____|
// |_____|            |__|                   |_____|
//
// Copyright (c) 2023 Fabio Cicerchia. https://fabiocicerchia.it. MIT License
// Repo: https://github.com/fabiocicerchia/go-proxy-cache

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/fabiocicerchia/go-proxy-cache/server/router"
)

func TestSelectBackendSingle(t *testing.T) {
	r := &router.Route{Backends: []router.Backend{
		{Endpoints: []string{"10.0.0.1:80"}, Weight: 1},
	}}

	idx, ok := r.SelectBackend()

	assert.True(t, ok)
	assert.Equal(t, 0, idx)
}

func TestSelectBackendNoBackends(t *testing.T) {
	r := &router.Route{}

	_, ok := r.SelectBackend()

	assert.False(t, ok)
}

func TestSelectBackendSkipsEmptyBackends(t *testing.T) {
	r := &router.Route{Backends: []router.Backend{
		{Endpoints: nil, Weight: 100},
		{Endpoints: []string{"10.0.0.2:80"}, Weight: 1},
	}}

	// A backend whose pods all went away must never be selected, however
	// heavily it is weighted, or it would 502 most of the traffic.
	for i := 0; i < 200; i++ {
		idx, ok := r.SelectBackend()

		assert.True(t, ok)
		assert.Equal(t, 1, idx)
	}
}

func TestSelectBackendAllEmpty(t *testing.T) {
	r := &router.Route{Backends: []router.Backend{
		{Endpoints: nil, Weight: 1},
		{Endpoints: nil, Weight: 1},
	}}

	_, ok := r.SelectBackend()

	assert.False(t, ok)
}

func TestSelectBackendZeroWeightsFallBack(t *testing.T) {
	r := &router.Route{Backends: []router.Backend{
		{Endpoints: []string{"10.0.0.1:80"}, Weight: 0},
		{Endpoints: []string{"10.0.0.2:80"}, Weight: 0},
	}}

	idx, ok := r.SelectBackend()

	assert.True(t, ok)
	assert.Equal(t, 0, idx)
}

func TestSelectBackendHonoursWeights(t *testing.T) {
	r := &router.Route{Backends: []router.Backend{
		{Endpoints: []string{"10.0.0.1:80"}, Weight: 90},
		{Endpoints: []string{"10.0.0.2:80"}, Weight: 10},
	}}

	counts := map[int]int{}
	for i := 0; i < 10000; i++ {
		idx, ok := r.SelectBackend()
		assert.True(t, ok)
		counts[idx]++
	}

	// Loose bounds: this asserts the weights are respected at all, not that
	// the PRNG is well distributed.
	assert.Greater(t, counts[0], counts[1]*3)
	assert.Greater(t, counts[1], 0)
}

func TestStorePublishIsRaceFree(t *testing.T) {
	defer router.Reset()

	router.Enable()
	assert.True(t, router.Enabled())

	var wg sync.WaitGroup

	for i := 0; i < 8; i++ {
		wg.Add(1)

		go func(n int) {
			defer wg.Done()

			for j := 0; j < 200; j++ {
				router.Publish(router.Build([]*router.Route{
					{ID: "r", Host: "a.example.com", Path: "/", PathType: router.PathPrefix},
				}))
				_ = router.Current().Len()
			}
		}(i)
	}

	wg.Wait()
}

func TestResetLeavesRoutedMode(t *testing.T) {
	router.Enable()
	router.Publish(router.Build([]*router.Route{{ID: "r", Host: "x", Path: "/"}}))

	router.Reset()

	assert.False(t, router.Enabled())
	assert.Nil(t, router.Current())
}

// The health check is inherited from the global configuration, whose scheme
// defaults to https. Probing a plain-HTTP backend over TLS marks every node
// unhealthy, which leaves the balancer with nothing to pick.
func TestUpstreamProbesTheBackendScheme(t *testing.T) {
	r := &router.Route{Backends: []router.Backend{
		{Endpoints: []string{"10.0.0.1:8080"}, Scheme: "http"},
		{Endpoints: []string{"10.0.0.2:8443"}, Scheme: "https"},
	}}
	r.Config.Server.Upstream.HealthCheck.Scheme = "https"

	plain := r.Upstream(0)
	assert.Equal(t, "http", plain.Scheme)
	assert.Equal(t, "http", plain.HealthCheck.Scheme)
	assert.Equal(t, []string{"10.0.0.1:8080"}, plain.Endpoints)

	secure := r.Upstream(1)
	assert.Equal(t, "https", secure.Scheme)
	assert.Equal(t, "https", secure.HealthCheck.Scheme)
}

// A WebSocket backend is reached through an HTTP handshake, and an HTTP client
// refuses a ws:// URL outright, so probing one marks every node unhealthy.
func TestUpstreamProbesWebSocketBackendsOverHTTP(t *testing.T) {
	for backend, probe := range map[string]string{
		"http":  "http",
		"https": "https",
		"ws":    "http",
		"wss":   "https",
	} {
		r := &router.Route{Backends: []router.Backend{
			{Endpoints: []string{"10.0.0.1:8080"}, Scheme: backend},
		}}

		upstream := r.Upstream(0)

		assert.Equal(t, backend, upstream.Scheme, backend)
		assert.Equal(t, probe, upstream.HealthCheck.Scheme, backend)
	}
}
