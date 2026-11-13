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
	"net"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Every request builds its own RequestCall, so this is what serving N requests
// looks like to the origin: it should see one connection, not N.
func TestUpstreamConnectionsAreReusedAcrossRequests(t *testing.T) {
	var conns int32

	upstream := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("ok"))
	}))
	upstream.Config.ConnState = func(_ net.Conn, s http.ConnState) {
		if s == http.StateNew {
			atomic.AddInt32(&conns, 1)
		}
	}
	upstream.Start()
	defer upstream.Close()

	for i := 0; i < 5; i++ {
		req, _ := http.NewRequest(http.MethodGet, upstream.URL, nil)
		resp, err := RequestCall{}.patchProxyTransport().RoundTrip(req)
		assert.Nil(t, err)
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
	}

	assert.Equal(t, int32(1), atomic.LoadInt32(&conns))
}

func TestUpstreamTransportIsPerInsecureBridge(t *testing.T) {
	secure := RequestCall{}
	insecure := RequestCall{}
	insecure.DomainConfig.Server.Upstream.InsecureBridge = true

	assert.Same(t, secure.patchProxyTransport(), secure.patchProxyTransport())
	assert.NotSame(t, secure.patchProxyTransport(), insecure.patchProxyTransport())
	assert.True(t, insecure.patchProxyTransport().TLSClientConfig.InsecureSkipVerify)
	assert.False(t, secure.patchProxyTransport().TLSClientConfig.InsecureSkipVerify)
}
