//go:build all || unit
// +build all unit

package jwt

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
)

// Validate takes the settings rather than resolving them from the Host header,
// so a caller that knows which route matched can hand over that route's
// configuration.
func TestValidateIsANoopWithoutAJwksURL(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "http://example.com/foo", nil)
	w := httptest.NewRecorder()

	err := Validate(w, req, &config.Jwt{})

	assert.Nil(t, err)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestValidateIsANoopForAnExcludedPath(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "http://example.com/health", nil)
	w := httptest.NewRecorder()

	cfg := config.Jwt{JwksUrl: "https://issuer.example.com/jwks.json", ExcludedPaths: []string{"^/health$"}}

	err := Validate(w, req, &cfg)

	assert.Nil(t, err)
	assert.Equal(t, http.StatusOK, w.Code)
}

// A configured JWKS URL with no usable cache is a misconfiguration, and must
// reject rather than wave the request through.
func TestValidateFailsClosedWithoutACache(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "http://example.com/api", nil)
	w := httptest.NewRecorder()

	cfg := config.Jwt{JwksUrl: "https://issuer.example.com/jwks.json"}
	config.InitJWT(&cfg)
	cfg.JwkCache = nil

	err := Validate(w, req, &cfg)

	assert.NotNil(t, err)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestValidateRejectsAMissingToken(t *testing.T) {
	_, _, jsonJWKKeySetSingle, jsonJWKKeySetMultiple := GenerateTestKeysAndKeySets()
	ts := CreateTestServer(t, jsonJWKKeySetSingle, jsonJWKKeySetMultiple, 0)
	defer ts.Close()

	cfg := config.Jwt{JwksUrl: ts.URL + "/.well-known-single/jwks.json", JwksRefreshInterval: 15}
	config.InitJWT(&cfg)

	req := httptest.NewRequest(http.MethodGet, "http://example.com/api", nil)
	w := httptest.NewRecorder()

	err := Validate(w, req, &cfg)

	assert.NotNil(t, err)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}
