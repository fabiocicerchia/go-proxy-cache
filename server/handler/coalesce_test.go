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
	"strings"
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

func TestCoalescingKeepsPrivateAndCredentialedResponsesSeparate(t *testing.T) {
	for _, testCase := range []struct {
		name           string
		requestHeader  string
		responseHeader string
		responseValue  string
	}{
		{"private", "", "Cache-Control", "private, max-age=60"},
		{"no-store", "", "Cache-Control", "public, no-store"},
		{"unspecified", "", "", ""},
		{"cookie", "Cookie", "Cache-Control", "public, max-age=60"},
		{"authorization", "Authorization", "Cache-Control", "public, max-age=60"},
		{"set-cookie", "", "Set-Cookie", "session=example"},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			var hits atomic.Int32
			upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				hits.Add(1)
				if testCase.responseHeader != "" {
					w.Header().Set(testCase.responseHeader, testCase.responseValue)
				}
				_, _ = io.WriteString(w, "example")
			}))
			defer upstream.Close()
			transport := coalescingRoundTripper{RoundTripper: http.DefaultTransport}
			var requests sync.WaitGroup
			for range 10 {
				requests.Go(func() {
					request, err := http.NewRequest(http.MethodGet, upstream.URL, nil)
					assert.NoError(t, err)
					if testCase.requestHeader != "" {
						request.Header.Set(testCase.requestHeader, "example")
					}
					response, err := transport.RoundTrip(request)
					if !assert.NoError(t, err) {
						return
					}
					_, _ = io.Copy(io.Discard, response.Body)
					_ = response.Body.Close()
				})
			}
			requests.Wait()
			assert.EqualValues(t, 10, hits.Load())
		})
	}
}

func TestCoalescingPreservesLargeStreamingBodies(t *testing.T) {
	body := strings.Repeat("example", 200000)
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "public, max-age=60")
		_, _ = io.WriteString(w, body)
	}))
	defer upstream.Close()
	transport := coalescingRoundTripper{RoundTripper: http.DefaultTransport}
	request, err := http.NewRequest(http.MethodGet, upstream.URL, nil)
	assert.NoError(t, err)
	response, err := transport.RoundTrip(request)
	if !assert.NoError(t, err) {
		return
	}
	defer response.Body.Close()
	actual, err := io.ReadAll(response.Body)
	assert.NoError(t, err)
	assert.Equal(t, body, string(actual))
}

func TestCoalescingRoundTripperCollapsesConcurrentGETs(t *testing.T) {
	var hits int32

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&hits, 1)
		w.Header().Set("Cache-Control", "public, max-age=60")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("hello"))
	}))
	defer upstream.Close()

	rt := coalescingRoundTripper{RoundTripper: http.DefaultTransport}

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

	rt := coalescingRoundTripper{RoundTripper: http.DefaultTransport}

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
