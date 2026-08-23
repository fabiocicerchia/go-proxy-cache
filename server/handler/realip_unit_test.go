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
	"github.com/fabiocicerchia/go-proxy-cache/server/response"
)

func newRequestCall(remoteAddr string, trustedProxies []string, headers map[string]string, tls bool) handler.RequestCall {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = remoteAddr

	for k, v := range headers {
		req.Header.Set(k, v)
	}

	if !tls {
		req.TLS = nil
	}

	domainConfig := config.Configuration{}
	domainConfig.Server.TrustedProxies = trustedProxies

	rc := handler.RequestCall{
		ReqID:        "test",
		Request:      *req,
		DomainConfig: domainConfig,
		Response:     response.NewLoggedResponseWriter(httptest.NewRecorder(), "test"),
	}

	return rc
}

func TestClientIPWithoutTrustedProxies(t *testing.T) {
	rc := newRequestCall("203.0.113.9:1234", nil, map[string]string{
		"X-Forwarded-For": "1.2.3.4",
	}, false)

	// With nothing trusted, a forged X-Forwarded-For must be ignored entirely.
	assert.Equal(t, "203.0.113.9", rc.ClientIP())
}

func TestClientIPBehindTrustedProxy(t *testing.T) {
	rc := newRequestCall("10.0.0.5:1234", []string{"10.0.0.0/8"}, map[string]string{
		"X-Forwarded-For": "203.0.113.9",
	}, false)

	assert.Equal(t, "203.0.113.9", rc.ClientIP())
}

func TestClientIPSkipsTrustedHopsFromTheRight(t *testing.T) {
	rc := newRequestCall("10.0.0.5:1234", []string{"10.0.0.0/8"}, map[string]string{
		// The client claims to be 1.1.1.1 but the first address we control is
		// 203.0.113.9, so anything to its left is attacker controlled.
		"X-Forwarded-For": "1.1.1.1, 203.0.113.9, 10.0.0.7",
	}, false)

	assert.Equal(t, "203.0.113.9", rc.ClientIP())
}

func TestClientIPFallsBackOnMalformedChain(t *testing.T) {
	rc := newRequestCall("10.0.0.5:1234", []string{"10.0.0.0/8"}, map[string]string{
		"X-Forwarded-For": "not-an-ip",
	}, false)

	assert.Equal(t, "10.0.0.5", rc.ClientIP())
}

func TestClientIPAllHopsTrusted(t *testing.T) {
	rc := newRequestCall("10.0.0.5:1234", []string{"10.0.0.0/8"}, map[string]string{
		"X-Forwarded-For": "10.0.0.7, 10.0.0.8",
	}, false)

	assert.Equal(t, "10.0.0.5", rc.ClientIP())
}

func TestSchemeIgnoresForwardedProtoFromUntrustedPeer(t *testing.T) {
	rc := newRequestCall("203.0.113.9:1234", nil, map[string]string{
		"X-Forwarded-Proto": "https",
	}, false)

	assert.Equal(t, "http", rc.GetScheme())
}

func TestSchemeHonoursForwardedProtoFromTrustedProxy(t *testing.T) {
	rc := newRequestCall("10.0.0.5:1234", []string{"10.0.0.0/8"}, map[string]string{
		"X-Forwarded-Proto": "https",
	}, false)

	// The load balancer terminated TLS: believing the connection would
	// suppress HSTS and, with http_to_https on, loop the redirect.
	assert.Equal(t, "https", rc.GetScheme())
}

func TestSchemeUsesFirstHopOfForwardedProtoChain(t *testing.T) {
	rc := newRequestCall("10.0.0.5:1234", []string{"10.0.0.0/8"}, map[string]string{
		"X-Forwarded-Proto": "https, http",
	}, false)

	assert.Equal(t, "https", rc.GetScheme())
}

func TestSchemeFallsBackToConnectionWhenHeaderAbsent(t *testing.T) {
	rc := newRequestCall("10.0.0.5:1234", []string{"10.0.0.0/8"}, nil, false)

	assert.Equal(t, "http", rc.GetScheme())
}
