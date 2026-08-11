//go:build all || unit
// +build all unit

package handler

//                                                                         __
// .-----.-----.______.-----.----.-----.--.--.--.--.______.----.---.-.----|  |--.-----.
// |  _  |  _  |______|  _  |   _|  _  |_   _|  |  |______|  __|  _  |  __|     |  -__|
// |___  |_____|      |   __|__| |_____|__.__|___  |      |____|___._|____|__|__|_____|
// |_____|            |__|                   |_____|
//
// Copyright (c) 2023 Fabio Cicerchia. https://fabiocicerchia.it. MIT License
// Repo: https://github.com/fabiocicerchia/go-proxy-cache

import (
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCollapsedForwardingFeatureFlag(t *testing.T) {
	rc := RequestCall{}

	assert.IsType(t, &http.Transport{}, rc.proxyTransport())

	rc.DomainConfig.Server.Upstream.CollapsedForwarding = true

	assert.IsType(t, coalescingRoundTripper{}, rc.proxyTransport())
}

func TestCoalescingRoundTripperCollapsesConcurrentGETs(t *testing.T) {
	var hits int32

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&hits, 1)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("hello"))
	}))
	defer upstream.Close()

	rt := coalescingRoundTripper{http.DefaultTransport}

	const n = 20
	var wg sync.WaitGroup
	wg.Add(n)
	for i := 0; i < n; i++ {
		go func() {
			defer wg.Done()

			req, err := http.NewRequest(http.MethodGet, upstream.URL, nil)
			assert.NoError(t, err)

			resp, err := rt.RoundTrip(req)
			assert.NoError(t, err)
			defer resp.Body.Close()

			body, err := io.ReadAll(resp.Body)
			assert.NoError(t, err)
			assert.Equal(t, "hello", string(body))
			assert.Equal(t, http.StatusOK, resp.StatusCode)
		}()
	}
	wg.Wait()

	assert.Less(t, int(atomic.LoadInt32(&hits)), n, "concurrent identical GETs should be collapsed into fewer upstream hits")
}

func TestCoalescingRoundTripperDoesNotCollapsePOST(t *testing.T) {
	var hits int32

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&hits, 1)
		w.WriteHeader(http.StatusOK)
	}))
	defer upstream.Close()

	rt := coalescingRoundTripper{http.DefaultTransport}

	const n = 5
	var wg sync.WaitGroup
	wg.Add(n)
	for i := 0; i < n; i++ {
		go func() {
			defer wg.Done()
			req, err := http.NewRequest(http.MethodPost, upstream.URL, nil)
			assert.NoError(t, err)
			resp, err := rt.RoundTrip(req)
			assert.NoError(t, err)
			_ = resp.Body.Close()
		}()
	}
	wg.Wait()

	assert.EqualValues(t, n, hits, "POST requests must never be coalesced")
}
