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
	"net/http"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/fabiocicerchia/go-proxy-cache/config"
	"github.com/fabiocicerchia/go-proxy-cache/server/balancer"
	"github.com/fabiocicerchia/go-proxy-cache/server/handler"
)

func TestFixRequestOneItemInLB(t *testing.T) {
	config.Config = config.Configuration{
		Server: config.Server{
			Forwarding: config.Forward{
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

	balancer.InitRoundRobin(config.Config.Server.Forwarding.Host, config.Config.Server.Forwarding.Endpoints)

	r := handler.RequestCall{Request: reqMock}
	r.FixRequest(u, config.Config.Server.Forwarding)

	assert.Equal(t, "localhost", r.Request.Header.Get("X-Forwarded-Host"))

	assert.Equal(t, "server1:443", r.Request.URL.Host)
	assert.Equal(t, "developer.mozilla.org", r.Request.Host)
}

func TestFixRequestThreeItemsInLB(t *testing.T) {
	config.Config = config.Configuration{
		Server: config.Server{
			Forwarding: config.Forward{
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

	reqMock := &http.Request{
		Proto:      "HTTPS",
		Method:     "POST",
		RemoteAddr: "127.0.0.1",
		URL:        &url.URL{Path: "/path/to/file"},
		Header: http.Header{
			"Host": []string{"localhost"},
		},
	}

	balancer.InitRoundRobin(config.Config.Server.Forwarding.Host, config.Config.Server.Forwarding.Endpoints)

	// --- FIRST ROUND

	r := handler.RequestCall{Request: reqMock}
	r.FixRequest(u, config.Config.Server.Forwarding)

	assert.Equal(t, "localhost", r.Request.Header.Get("X-Forwarded-Host"))

	assert.Equal(t, "server1:443", r.Request.URL.Host)
	assert.Equal(t, "developer.mozilla.org", r.Request.Host)

	// --- SECOND ROUND

	r.FixRequest(u, config.Config.Server.Forwarding)

	assert.Equal(t, "localhost", r.Request.Header.Get("X-Forwarded-Host"))

	assert.Equal(t, "server2:443", r.Request.URL.Host)
	assert.Equal(t, "developer.mozilla.org", r.Request.Host)
}
