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
	"github.com/fabiocicerchia/go-proxy-cache/utils"
)

func TestFixRequestOneItemInLB(t *testing.T) {
	config.Config = config.Configuration{
		Server: config.Server{
			Upstream: config.Upstream{
				Host:      "developer.mozilla.org",
				Scheme:    "https",
				Endpoints: []string{"server1"},
			},
		},
	}

	u := url.URL{
		Scheme: "https",
		Host:   "localhost",
	}

	reqMock := &http.Request{
		Proto:      "HTTPS",
		Method:     "POST",
		RemoteAddr: "127.0.0.1",
		URL:        &url.URL{Path: "/path/to/file"},
		Header: http.Header{
			"Host": []string{"localhost"},
		},
	}

	domainID := config.Config.Server.Upstream.Host + utils.StringSeparatorOne + config.Config.Server.Upstream.Scheme
	balancer.InitRoundRobin(domainID, config.Config.Server.Upstream.Endpoints)

	r := handler.RequestCall{Request: reqMock}
	r.FixRequest(u, config.Config.Server.Upstream)

	assert.Equal(t, "localhost", r.Request.Header.Get("X-Forwarded-Host"))
	assert.Equal(t, "http", r.Request.Header.Get("X-Forwarded-Proto"))
	assert.Equal(t, "127.0.0.1", r.Request.Header.Get("X-Forwarded-For"))

	assert.Equal(t, "server1:443", r.Request.URL.Host)
	assert.Equal(t, "developer.mozilla.org", r.Request.Host)
}

func TestFixRequestOneItemWithPortInLB(t *testing.T) {
	config.Config = config.Configuration{
		Server: config.Server{
			Upstream: config.Upstream{
				Host:      "developer.mozilla.org",
				Scheme:    "https",
				Endpoints: []string{"server1:8080"},
			},
		},
	}

	u := url.URL{
		Scheme: "https",
		Host:   "localhost",
	}

	reqMock := &http.Request{
		Proto:      "HTTPS",
		Method:     "POST",
		RemoteAddr: "127.0.0.1",
		URL:        &url.URL{Path: "/path/to/file"},
		Header: http.Header{
			"Host": []string{"localhost"},
		},
	}

	domainID := config.Config.Server.Upstream.Host + utils.StringSeparatorOne + config.Config.Server.Upstream.Scheme
	balancer.InitRoundRobin(domainID, config.Config.Server.Upstream.Endpoints)

	r := handler.RequestCall{Request: reqMock}
	r.FixRequest(u, config.Config.Server.Upstream)

	assert.Equal(t, "localhost", r.Request.Header.Get("X-Forwarded-Host"))
	assert.Equal(t, "http", r.Request.Header.Get("X-Forwarded-Proto"))
	assert.Equal(t, "127.0.0.1", r.Request.Header.Get("X-Forwarded-For"))

	assert.Equal(t, "server1:8080", r.Request.URL.Host)
	assert.Equal(t, "developer.mozilla.org", r.Request.Host)
}

func TestFixRequestThreeItemsInLB(t *testing.T) {
	config.Config = config.Configuration{
		Server: config.Server{
			Upstream: config.Upstream{
				Host:      "developer.mozilla.org",
				Scheme:    "https",
				Endpoints: []string{"server1", "server2", "server3"},
			},
		},
	}

	u := url.URL{
		Scheme: "https",
		Host:   "localhost",
	}

	domainID := config.Config.Server.Upstream.Host + utils.StringSeparatorOne + config.Config.Server.Upstream.Scheme
	balancer.InitRoundRobin(domainID, config.Config.Server.Upstream.Endpoints)

	// --- FIRST ROUND

	reqMock := &http.Request{
		Proto:      "HTTPS",
		Method:     "POST",
		RemoteAddr: "127.0.0.1",
		URL:        &url.URL{Path: "/path/to/file"},
		Header: http.Header{
			"Host": []string{"localhost"},
		},
	}

	r := handler.RequestCall{Request: reqMock}
	r.FixRequest(u, config.Config.Server.Upstream)

	assert.Equal(t, "localhost", r.Request.Header.Get("X-Forwarded-Host"))
	assert.Equal(t, "http", r.Request.Header.Get("X-Forwarded-Proto"))
	assert.Equal(t, "127.0.0.1", r.Request.Header.Get("X-Forwarded-For"))

	assert.Equal(t, "server1:443", r.Request.URL.Host)
	assert.Equal(t, "developer.mozilla.org", r.Request.Host)

	// --- SECOND ROUND

	reqMock = &http.Request{
		Proto:      "HTTPS",
		Method:     "POST",
		RemoteAddr: "127.0.0.1",
		URL:        &url.URL{Path: "/path/to/file"},
		Header: http.Header{
			"Host": []string{"localhost"},
		},
	}

	r = handler.RequestCall{Request: reqMock}
	r.FixRequest(u, config.Config.Server.Upstream)

	assert.Equal(t, "localhost", r.Request.Header.Get("X-Forwarded-Host"))
	assert.Equal(t, "http", r.Request.Header.Get("X-Forwarded-Proto"))
	assert.Equal(t, "127.0.0.1", r.Request.Header.Get("X-Forwarded-For"))

	assert.Equal(t, "server2:443", r.Request.URL.Host)
	assert.Equal(t, "developer.mozilla.org", r.Request.Host)
}

func TestXForwardedFor(t *testing.T) {
	config.Config = config.Configuration{
		Server: config.Server{
			Upstream: config.Upstream{
				Host:      "developer.mozilla.org",
				Scheme:    "https",
				Endpoints: []string{"server1"},
			},
		},
	}

	u := url.URL{
		Scheme: "https",
		Host:   "localhost",
	}

	reqMock := &http.Request{
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

	domainID := config.Config.Server.Upstream.Host + utils.StringSeparatorOne + config.Config.Server.Upstream.Scheme
	balancer.InitRoundRobin(domainID, config.Config.Server.Upstream.Endpoints)

	r := handler.RequestCall{Request: reqMock}
	r.FixRequest(u, config.Config.Server.Upstream)

	assert.Equal(t, "https", r.Request.Header.Get("X-Forwarded-Proto"))
	assert.Equal(t, "192.168.1.1, 127.0.0.1", r.Request.Header.Get("X-Forwarded-For"))
}
