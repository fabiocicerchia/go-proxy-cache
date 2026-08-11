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
	"crypto/tls"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/fabiocicerchia/go-proxy-cache/config"
	"github.com/fabiocicerchia/go-proxy-cache/server/handler"
)

func TestSetHSTSHeaderEnabledOverHTTPS(t *testing.T) {
	reqMock := http.Request{TLS: &tls.ConnectionState{}}

	r := handler.NewRequestCall(httptest.NewRecorder(), &reqMock)
	r.DomainConfig = config.Configuration{
		Server: config.Server{
			TLS: config.TLS{
				HSTS: config.HSTS{Enabled: true, MaxAge: 63072000, IncludeSubdomains: true, Preload: true},
			},
		},
	}

	r.SetHSTSHeader()

	assert.Equal(t, "max-age=63072000; includeSubDomains; preload", r.Response.Header().Get("Strict-Transport-Security"))
}

func TestSetHSTSHeaderAbsentOverPlainHTTP(t *testing.T) {
	reqMock := http.Request{} // TLS == nil -> plain HTTP

	r := handler.NewRequestCall(httptest.NewRecorder(), &reqMock)
	r.DomainConfig = config.Configuration{
		Server: config.Server{
			TLS: config.TLS{
				HSTS: config.HSTS{Enabled: true},
			},
		},
	}

	r.SetHSTSHeader()

	assert.Empty(t, r.Response.Header().Get("Strict-Transport-Security"))
}

func TestSetHSTSHeaderAbsentWhenDisabled(t *testing.T) {
	reqMock := http.Request{TLS: &tls.ConnectionState{}}

	r := handler.NewRequestCall(httptest.NewRecorder(), &reqMock)
	r.DomainConfig = config.Configuration{
		Server: config.Server{
			TLS: config.TLS{
				HSTS: config.HSTS{Enabled: false},
			},
		},
	}

	r.SetHSTSHeader()

	assert.Empty(t, r.Response.Header().Get("Strict-Transport-Security"))
}
