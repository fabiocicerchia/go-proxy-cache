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
	"context"
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/fabiocicerchia/go-proxy-cache/config"
	"github.com/lestrrat-go/jwx/jwt"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestAllowedScope(t *testing.T) {
	co := &config.Jwt{AllowedScopes: []string{"admin"}}
	res := haveAllowedScope([]string{""}, co.AllowedScopes)
	assert.Equal(t, res, false, "No scope provided, should be false")

	res = haveAllowedScope([]string{"admin"}, co.AllowedScopes)
	assert.Equal(t, res, true, "Admin is provided and allowed, should be true")

	res = haveAllowedScope([]string{"admin"}, []string{})
	assert.Equal(t, res, false, "No allowed scopes, should be false")

	res = haveAllowedScope([]string{}, []string{})
	assert.Equal(t, res, false, "Empty scopes and empty allowed scopes, should be false")
}

func TestGetScopesWithScopeClaim(t *testing.T) {
	jwkKeySingle, _, _, _ := GenerateTestKeysAndKeySets()
	strExpiredToken, _ := GenerateTestJWT(jwkKeySingle, "scope", true)

	token, _ := jwt.ParseString(strExpiredToken, jwt.WithTypedClaim("scope", json.RawMessage{}))

	res := getScopes(token)

	assert.ElementsMatch(t, res, []string{"scope1", "scope2", "scope3"}, "Scopes provided doesn't match")
}

func TestGetScopesWithScpClaim(t *testing.T) {
	jwkKeySingle, _, _, _ := GenerateTestKeysAndKeySets()
	scpClaimToken, _ := GenerateTestJWT(jwkKeySingle, "scp", true)
	token, _ := jwt.ParseString(scpClaimToken, jwt.WithTypedClaim("scp", json.RawMessage{}))

	res := getScopes(token)

	assert.ElementsMatch(t, res, []string{"scope1", "scope2", "scope3"}, "Scopes provided doesn't match")
}

func TestValidateJWTWithoutAnyToken(t *testing.T) {
	_, _, jsonJWKKeySetSingle, jsonJWKKeySetMultiple := GenerateTestKeysAndKeySets()
	ts := CreateTestServer(t, jsonJWKKeySetSingle, jsonJWKKeySetMultiple)
	defer ts.Close()
	config.Config.Jwt.JwksUrl = ts.URL + "/.well-known-single/jwks.json"
	co = nil
	InitJWT(&config.Jwt{
		IncludedPaths: config.Config.Jwt.IncludedPaths,
		AllowedScopes: config.Config.Jwt.AllowedScopes,
		JwksUrl:       config.Config.Jwt.JwksUrl,
		Context:       context.Background(),
		Logger:        log.New(),
	})
	req := httptest.NewRequest("GET", "http://example.com/foo", nil)
	w := httptest.NewRecorder()

	keySet, _ := fetchKeySet(w)
	err := ValidateJWT(w, req, keySet)

	assert.NotNil(t, err)
	assert.Equal(t, w.Code, 401, "No token provided status code should be 401")
	assert.Containsf(t, w.Body.String(), "failed to find a valid token in any location of the request", "No token provided status code should be 401")
}

func TestValidateJWTWithoutAnyKeySet(t *testing.T) {
	_, jwkKeyMultiple, jsonJWKKeySetSingle, jsonJWKKeySetMultiple := GenerateTestKeysAndKeySets()
	scpExpiredToken, _ := GenerateTestJWT(jwkKeyMultiple, "scp", true)
	ts := CreateTestServer(t, jsonJWKKeySetSingle, jsonJWKKeySetMultiple)
	defer ts.Close()
	config.Config.Jwt.JwksUrl = ts.URL + "/.bad-known/jwks.json"
	co = nil
	InitJWT(&config.Jwt{
		IncludedPaths: config.Config.Jwt.IncludedPaths,
		AllowedScopes: config.Config.Jwt.AllowedScopes,
		JwksUrl:       config.Config.Jwt.JwksUrl,
		Context:       context.Background(),
		Logger:        log.New(),
	})
	req := httptest.NewRequest("GET", "http://example.com/foo", nil)
	w := httptest.NewRecorder()
	req.Header.Add("Authorization", "Bearer "+scpExpiredToken)

	_, err := fetchKeySet(w)

	assert.NotNil(t, err)
	assert.Equal(t, w.Code, 401, "failed to fetch resource pointed by")
	assert.Containsf(t, w.Body.String(), "failed to fetch resource pointed by", "failed to fetch resource pointed by")
}

func TestValidateJWTWithAnExpiredToken(t *testing.T) {
	_, jwkKeyMultiple, jsonJWKKeySetSingle, jsonJWKKeySetMultiple := GenerateTestKeysAndKeySets()
	scpExpiredToken, _ := GenerateTestJWT(jwkKeyMultiple, "scp", true)
	ts := CreateTestServer(t, jsonJWKKeySetSingle, jsonJWKKeySetMultiple)
	defer ts.Close()
	config.Config.Jwt.JwksUrl = ts.URL + "/.well-known-multiple/jwks.json"
	co = nil
	InitJWT(&config.Jwt{
		IncludedPaths: config.Config.Jwt.IncludedPaths,
		AllowedScopes: config.Config.Jwt.AllowedScopes,
		JwksUrl:       config.Config.Jwt.JwksUrl,
		Context:       context.Background(),
		Logger:        log.New(),
	})
	req := httptest.NewRequest("GET", "http://example.com/foo", nil)
	w := httptest.NewRecorder()
	req.Header.Add("Authorization", "Bearer "+scpExpiredToken)

	keySet, _ := fetchKeySet(w)
	err := ValidateJWT(w, req, keySet)

	assert.NotNil(t, err)
	assert.Equal(t, w.Code, 401, "exp not satisfied")
	assert.Containsf(t, w.Body.String(), "exp not satisfied", "Token expired: exp not satisfied")
}

func TestValidateJWTWithoutAnyScopeInTheConfig(t *testing.T) {
	jwkKeySingle, _, jsonJWKKeySetSingle, jsonJWKKeySetMultiple := GenerateTestKeysAndKeySets()
	scopeGoodToken, _ := GenerateTestJWT(jwkKeySingle, "scope", false)
	ts := CreateTestServer(t, jsonJWKKeySetSingle, jsonJWKKeySetMultiple)
	defer ts.Close()
	config.Config.Jwt.AllowedScopes = nil
	config.Config.Jwt.JwksUrl = ts.URL + "/.well-known-single/jwks.json"
	co = nil
	InitJWT(&config.Jwt{
		IncludedPaths: config.Config.Jwt.IncludedPaths,
		AllowedScopes: config.Config.Jwt.AllowedScopes,
		JwksUrl:       config.Config.Jwt.JwksUrl,
		Context:       context.Background(),
		Logger:        log.New(),
	})
	req := httptest.NewRequest("GET", "http://example.com/foo", nil)
	w := httptest.NewRecorder()
	req.Header.Add("Authorization", "Bearer "+scopeGoodToken)

	keySet, _ := fetchKeySet(w)
	err := ValidateJWT(w, req, keySet)

	assert.NotNil(t, err)
	assert.Equal(t, w.Code, 401, "Invalid Scope")
	assert.Containsf(t, w.Body.String(), "Invalid Scope", "Invalid Scope")
}

func TestValidateJWTWithScopeConfigAndScopeClaimToken(t *testing.T) {
	jwkKeySingle, _, jsonJWKKeySetSingle, jsonJWKKeySetMultiple := GenerateTestKeysAndKeySets()
	scopeGoodToken, _ := GenerateTestJWT(jwkKeySingle, "scope", false)
	ts := CreateTestServer(t, jsonJWKKeySetSingle, jsonJWKKeySetMultiple)
	defer ts.Close()
	config.Config.Jwt.JwksUrl = ts.URL + "/.well-known-single/jwks.json"
	config.Config.Jwt.AllowedScopes = []string{"scope1"}
	co = nil
	InitJWT(&config.Jwt{
		IncludedPaths: config.Config.Jwt.IncludedPaths,
		AllowedScopes: config.Config.Jwt.AllowedScopes,
		JwksUrl:       config.Config.Jwt.JwksUrl,
		Context:       context.Background(),
		Logger:        log.New(),
	})
	req := httptest.NewRequest("GET", "http://example.com/foo", nil)
	w := httptest.NewRecorder()
	req.Header.Add("Authorization", "Bearer "+scopeGoodToken)

	keySet, _ := fetchKeySet(w)
	err := ValidateJWT(w, req, keySet)

	assert.Nil(t, err)
	assert.Equal(t, w.Code, 200, "Status OK")
	assert.Containsf(t, w.Body.String(), "", "Status OK")
}

func TestValidateJWTWithScopeConfigAndScpClaimToken(t *testing.T) {
	jwkKeySingle, _, jsonJWKKeySetSingle, jsonJWKKeySetMultiple := GenerateTestKeysAndKeySets()
	scpGoodToken, _ := GenerateTestJWT(jwkKeySingle, "scp", false)
	ts := CreateTestServer(t, jsonJWKKeySetSingle, jsonJWKKeySetMultiple)
	defer ts.Close()
	config.Config.Jwt.JwksUrl = ts.URL + "/.well-known-single/jwks.json"
	config.Config.Jwt.AllowedScopes = []string{"scope1"}
	co = nil
	InitJWT(&config.Jwt{
		IncludedPaths: config.Config.Jwt.IncludedPaths,
		AllowedScopes: config.Config.Jwt.AllowedScopes,
		JwksUrl:       config.Config.Jwt.JwksUrl,
		Context:       context.Background(),
		Logger:        log.New(),
	})
	req := httptest.NewRequest("GET", "http://example.com/foo", nil)
	w := httptest.NewRecorder()
	req.Header.Add("Authorization", "Bearer "+scpGoodToken)

	keySet, _ := fetchKeySet(w)
	err := ValidateJWT(w, req, keySet)

	assert.Nil(t, err)
	assert.Equal(t, w.Code, 200, "Status OK")
	assert.Containsf(t, w.Body.String(), "", "Status OK")
}

func TestValidateJWTWithMultipleKeysInKeySet(t *testing.T) {
	_, jwkKeyMultiple, jsonJWKKeySetSingle, jsonJWKKeySetMultiple := GenerateTestKeysAndKeySets()
	scopeGoodTokenMultiple, _ := GenerateTestJWT(jwkKeyMultiple, "scope", false)
	ts := CreateTestServer(t, jsonJWKKeySetSingle, jsonJWKKeySetMultiple)
	defer ts.Close()
	config.Config.Jwt.JwksUrl = ts.URL + "/.well-known-multiple/jwks.json"
	config.Config.Jwt.AllowedScopes = []string{"scope1"}
	co = nil
	InitJWT(&config.Jwt{
		IncludedPaths: config.Config.Jwt.IncludedPaths,
		AllowedScopes: config.Config.Jwt.AllowedScopes,
		JwksUrl:       config.Config.Jwt.JwksUrl,
		Context:       context.Background(),
		Logger:        log.New(),
	})
	req := httptest.NewRequest("GET", "http://example.com/foo", nil)
	w := httptest.NewRecorder()
	req.Header.Add("Authorization", "Bearer "+scopeGoodTokenMultiple)

	keySet, _ := fetchKeySet(w)
	err := ValidateJWT(w, req, keySet)

	assert.Nil(t, err)
	assert.Equal(t, w.Code, 200, "Status OK")
	assert.Containsf(t, w.Body.String(), "", "Status OK")
}
