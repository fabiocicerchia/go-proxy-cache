// +build all unit

package handler_test

//                                                                         __
// .-----.-----.______.-----.----.-----.--.--.--.--.______.----.---.-.----|  |--.-----.
// |  _  |  _  |______|  _  |   _|  _  |_   _|  |  |______|  __|  _  |  __|     |  -__|
// |___  |_____|      |   __|__| |_____|__.__|___  |      |____|___._|____|__|__|_____|
// |_____|            |__|                   |_____|
//
// Copyright (c) 2020 Fabio Cicerchia. https://fabiocicerchia.it. MIT License
// Repo: https://github.com/fabiocicerchia/go-proxy-cache

import (
	"crypto/tls"
	"net/http"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/fabiocicerchia/go-proxy-cache/config"
	"github.com/fabiocicerchia/go-proxy-cache/server/balancer"
	"github.com/fabiocicerchia/go-proxy-cache/server/handler"
)

func TestProxyCallOneItemInLB(t *testing.T) {
	cfg := config.Configuration{
		Server: config.Server{
			Upstream: config.Upstream{
				Host:      "developer.mozilla.org",
				Scheme:    "https",
				Endpoints: []string{"server1"},
			},
		},
	}

	reqMock := http.Request{
		Proto:      "HTTPS",
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
	proxyURL, _ := r.GetUpstreamURL()
	r.ProxyDirector(&r.Request)

	assert.Equal(t, "localhost", r.Request.Header.Get("X-Forwarded-Host"))
	assert.Equal(t, "http", r.Request.Header.Get("X-Forwarded-Proto"))
	assert.Equal(t, "127.0.0.1", r.Request.Header.Get("X-Forwarded-For"))

	assert.Equal(t, "server1:443", proxyURL.Host)
	assert.Equal(t, "developer.mozilla.org", r.Request.Host)
}

func TestProxyCallOneItemWithPortInLB(t *testing.T) {
	cfg := config.Configuration{
		Server: config.Server{
			Upstream: config.Upstream{
				Host:      "developer.mozilla.org",
				Scheme:    "https",
				Endpoints: []string{"server1:8080"},
			},
		},
	}

	reqMock := http.Request{
		Proto:      "HTTPS",
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
	proxyURL, _ := r.GetUpstreamURL()
	r.ProxyDirector(&r.Request)

	assert.Equal(t, "localhost", r.Request.Header.Get("X-Forwarded-Host"))
	assert.Equal(t, "http", r.Request.Header.Get("X-Forwarded-Proto"))
	assert.Equal(t, "127.0.0.1", r.Request.Header.Get("X-Forwarded-For"))

	assert.Equal(t, "server1:8080", proxyURL.Host)
	assert.Equal(t, "developer.mozilla.org", r.Request.Host)
}

func TestProxyCallThreeItemsInLB(t *testing.T) {
	cfg := config.Configuration{
		Server: config.Server{
			Upstream: config.Upstream{
				Host:      "developer.mozilla.org",
				Scheme:    "https",
				Endpoints: []string{"server1", "server2", "server3"},
			},
		},
	}

	domainID := cfg.Server.Upstream.GetDomainID()
	balancer.InitRoundRobin(domainID, cfg.Server.Upstream, false)

	// --- FIRST ROUND

	reqMock := http.Request{
		Proto:      "HTTPS",
		Method:     "POST",
		RemoteAddr: "127.0.0.1",
		URL:        &url.URL{Path: "/path/to/file"},
		Header: http.Header{
			"Host": []string{"localhost"},
		},
	}

	r := handler.RequestCall{Request: reqMock, DomainConfig: cfg}
	proxyURL, _ := r.GetUpstreamURL()
	r.ProxyDirector(&r.Request)

	assert.Equal(t, "localhost", r.Request.Header.Get("X-Forwarded-Host"))
	assert.Equal(t, "http", r.Request.Header.Get("X-Forwarded-Proto"))
	assert.Equal(t, "127.0.0.1", r.Request.Header.Get("X-Forwarded-For"))

	assert.Equal(t, "server1:443", proxyURL.Host)
	assert.Equal(t, "developer.mozilla.org", r.Request.Host)

	// --- SECOND ROUND

	reqMock = http.Request{
		Proto:      "HTTPS",
		Method:     "POST",
		RemoteAddr: "127.0.0.1",
		URL:        &url.URL{Path: "/path/to/file"},
		Header: http.Header{
			"Host": []string{"localhost"},
		},
	}

	r = handler.RequestCall{Request: reqMock, DomainConfig: cfg}
	proxyURL, _ = r.GetUpstreamURL()
	r.ProxyDirector(&r.Request)

	assert.Equal(t, "localhost", r.Request.Header.Get("X-Forwarded-Host"))
	assert.Equal(t, "http", r.Request.Header.Get("X-Forwarded-Proto"))
	assert.Equal(t, "127.0.0.1", r.Request.Header.Get("X-Forwarded-For"))

	assert.Equal(t, "server2:443", proxyURL.Host)
	assert.Equal(t, "developer.mozilla.org", r.Request.Host)
}

func TestXForwardedFor(t *testing.T) {
	cfg := config.Configuration{
		Server: config.Server{
			Upstream: config.Upstream{
				Host:      "developer.mozilla.org",
				Scheme:    "https",
				Endpoints: []string{"server1"},
			},
		},
	}

	reqMock := http.Request{
		Proto:      "HTTPS",
		Method:     "POST",
		RemoteAddr: "127.0.0.1",
		URL:        &url.URL{Path: "/path/to/file"},
		Header: http.Header{
			"Host":            []string{"localhost"},
			"X-Forwarded-For": []string{"192.168.1.1"},
		},
		TLS: &tls.ConnectionState{}, // mock a fake https
	}

	domainID := cfg.Server.Upstream.GetDomainID()
	balancer.InitRoundRobin(domainID, cfg.Server.Upstream, false)

	r := handler.RequestCall{Request: reqMock, DomainConfig: cfg}
	_, _ = r.GetUpstreamURL()
	r.ProxyDirector(&r.Request)

	assert.Equal(t, "https", r.Request.Header.Get("X-Forwarded-Proto"))
	assert.Equal(t, "192.168.1.1, 127.0.0.1", r.Request.Header.Get("X-Forwarded-For"))
}
