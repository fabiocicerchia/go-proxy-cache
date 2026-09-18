//go:build all || unit
// +build all unit

package config_test

//                                                                         __
// .-----.-----.______.-----.----.-----.--.--.--.--.______.----.---.-.----|  |--.-----.
// |  _  |  _  |______|  _  |   _|  _  |_   _|  |  |______|  __|  _  |  __|     |  -__|
// |___  |_____|      |   __|__| |_____|__.__|___  |      |____|___._|____|__|__|_____|
// |_____|            |__|                   |_____|
//
// Copyright (c) 2023 Fabio Cicerchia. https://fabiocicerchia.it. MIT License
// Repo: https://github.com/fabiocicerchia/go-proxy-cache

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/fabiocicerchia/go-proxy-cache/config"
)

// Each jwk.Cache runs a refresh goroutine for as long as its context lives.
// Route translation calls InitJWT on every pass, so building a fresh cache per
// call leaked a goroutine set per call.
func TestInitJWTReusesTheCachePerURL(t *testing.T) {
	first := config.Jwt{JwksUrl: "https://example.com/.well-known/jwks.json", JwksRefreshInterval: 15}
	second := config.Jwt{JwksUrl: "https://example.com/.well-known/jwks.json", JwksRefreshInterval: 15}

	config.InitJWT(&first)
	config.InitJWT(&second)

	assert.NotNil(t, first.JwkCache)
	assert.Same(t, first.JwkCache, second.JwkCache)
}

func TestInitJWTSeparatesDistinctSettings(t *testing.T) {
	url := config.Jwt{JwksUrl: "https://one.example.com/jwks.json", JwksRefreshInterval: 15}
	other := config.Jwt{JwksUrl: "https://two.example.com/jwks.json", JwksRefreshInterval: 15}
	interval := config.Jwt{JwksUrl: "https://one.example.com/jwks.json", JwksRefreshInterval: 30}

	config.InitJWT(&url)
	config.InitJWT(&other)
	config.InitJWT(&interval)

	assert.NotSame(t, url.JwkCache, other.JwkCache)
	assert.NotSame(t, url.JwkCache, interval.JwkCache)
}

// A config with no JWKS URL still gets a usable cache, as it did before, so
// callers that dereference it unconditionally keep working.
func TestInitJWTWithoutURL(t *testing.T) {
	empty := config.Jwt{JwksRefreshInterval: 15}

	config.InitJWT(&empty)

	assert.NotNil(t, empty.JwkCache)
	assert.NotNil(t, empty.Context)
}

// The per-request metrics switch is tri-state: unset lets the caller pick a
// default that suits where its routes come from, while an explicit value is
// honoured either way.
func TestMetricsPerRequestSeriesIsTriState(t *testing.T) {
	assert.Nil(t, config.Configuration{}.Metrics.PerRequestSeries, "unset means the caller decides")

	on := true
	off := false

	assert.True(t, *config.Configuration{Metrics: config.Metrics{PerRequestSeries: &on}}.Metrics.PerRequestSeries)
	assert.False(t, *config.Configuration{Metrics: config.Metrics{PerRequestSeries: &off}}.Metrics.PerRequestSeries)
}
