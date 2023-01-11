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
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/fabiocicerchia/go-proxy-cache/config"
	"github.com/fabiocicerchia/go-proxy-cache/server/balancer"
	"github.com/fabiocicerchia/go-proxy-cache/server/handler"
)

func TestGetUpstreamURLWithWildcard(t *testing.T) {
	cfg := config.Configuration{
		Server: config.Server{
			Upstream: config.Upstream{
				Host:      "developer.mozilla.org",
				Scheme:    "*", // emulate config.copyOverWithUpstream:179
				Endpoints: []string{"server1"},
			},
		},
	}

	reqMock := http.Request{
		Method:     "POST",
		RemoteAddr: "127.0.0.1",
		URL:        &url.URL{Path: "/path/to/file"},
		Header: http.Header{
			"Host": []string{"localhost"},
		},
	}

	domainID := cfg.Server.Upstream.GetDomainID()
	balancer.InitRoundRobin(domainID, cfg.Server.Upstream, false)

	r := handler.RequestCall{Request: reqMock, DomainConfig: cfg}
	proxyURL, err := r.GetUpstreamURL()
	assert.NoError(t, err)

	assert.Equal(t, "server1:80", proxyURL.Host)
	assert.Equal(t, "http", proxyURL.Scheme)
}

// Avoid the error parse "123abc.com:8080": first path segment in URL cannot contain colon
func TestGetUpstreamURLWithUnformattedEndpoint(t *testing.T) {
	cfg := config.Configuration{
		Server: config.Server{
			Upstream: config.Upstream{
				Host:      "example.com",
				Scheme:    "http",
				Endpoints: []string{"123abc.com:8080"},
			},
		},
	}

	reqMock := http.Request{
		Method:     "POST",
		RemoteAddr: "127.0.0.1",
		URL:        &url.URL{Path: "/path/to/file"},
		Header: http.Header{
			"Host": []string{"example.com"},
		},
	}

	domainID := cfg.Server.Upstream.GetDomainID()
	balancer.InitRoundRobin(domainID, cfg.Server.Upstream, false)

	r := handler.RequestCall{Request: reqMock, DomainConfig: cfg}
	proxyURL, err := r.GetUpstreamURL()
	assert.NoError(t, err)

	assert.Equal(t, "http", proxyURL.Scheme)
	assert.Equal(t, "", proxyURL.Opaque)
	assert.Nil(t, proxyURL.User)
	assert.Equal(t, "123abc.com:8080", proxyURL.Host)
	assert.Equal(t, "", proxyURL.Path)
	assert.Equal(t, "", proxyURL.RawPath)
	assert.False(t, proxyURL.ForceQuery)
	assert.Equal(t, "", proxyURL.RawQuery)
	assert.Equal(t, "", proxyURL.Fragment)
	assert.Equal(t, "", proxyURL.RawFragment)
}
