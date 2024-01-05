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
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/fabiocicerchia/go-proxy-cache/config"
	"github.com/lestrrat-go/jwx/v2/jwt"
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
	strExpiredToken, _ := GenerateTestJWT(jwkKeySingle, "scope", false)

	token, err := jwt.ParseString(strExpiredToken, jwt.WithTypedClaim("scope", json.RawMessage{}), jwt.WithVerify(false))
	assert.Nil(t, err)

	res := getScopes(token)

	assert.ElementsMatch(t, res, []string{"scope1", "scope2", "scope3"}, "Scopes provided doesn't match")
}

func TestGetScopesWithScpClaim(t *testing.T) {
	jwkKeySingle, _, _, _ := GenerateTestKeysAndKeySets()
	scpClaimToken, _ := GenerateTestJWT(jwkKeySingle, "scp", false)
	token, err := jwt.ParseString(scpClaimToken, jwt.WithTypedClaim("scp", json.RawMessage{}), jwt.WithVerify(false))
	assert.Nil(t, err)

	res := getScopes(token)

	assert.ElementsMatch(t, res, []string{"scope1", "scope2", "scope3"}, "Scopes provided doesn't match")
}

var jwtConfig = config.Jwt{
	ExcludedPaths:       config.Config.Jwt.ExcludedPaths,
	AllowedScopes:       config.Config.Jwt.AllowedScopes,
	JwksUrl:             config.Config.Jwt.JwksUrl,
	JwksRefreshInterval: config.Config.Jwt.JwksRefreshInterval,
	Context:             context.Background(),
	Logger:              log.New(),
}

func TestValidateJWTWithoutAnyToken(t *testing.T) {
	req := httptest.NewRequest("GET", "http://example.com/foo", nil)
	config.InitJWT(&jwtConfig)
	w := httptest.NewRecorder()
	privateKey, _ := rsa.GenerateKey(rand.Reader, 2048)
	publicKey := &privateKey.PublicKey
	_, keySet, _ := generateTestJWKMultipleKeys(privateKey, publicKey, "key-id-multiple", 1)
	co = &jwtConfig

	err := ValidateJWT(w, req, keySet)

	assert.NotNil(t, err)
	assert.Equal(t, w.Code, 401, "No token provided status code should be 401")
	assert.Containsf(t, w.Body.String(), "failed to find a valid token in any location of the request", "No token provided status code should be 401")
}

func TestValidateJWTWithoutAnyKeySet(t *testing.T) {
	_, jwkKeyMultiple, jsonJWKKeySetSingle, jsonJWKKeySetMultiple := GenerateTestKeysAndKeySets()
	scpExpiredToken, _ := GenerateTestJWT(jwkKeyMultiple, "scp", true)
	ts := CreateTestServer(t, jsonJWKKeySetSingle, jsonJWKKeySetMultiple, 0)
	defer ts.Close()
	jwtConfig.JwksUrl = ts.URL + "/.bad-known/jwks.json"
	req := httptest.NewRequest("GET", "http://example.com/foo", nil)
	config.InitJWT(&jwtConfig)
	w := httptest.NewRecorder()
	req.Header.Add("Authorization", "Bearer "+scpExpiredToken)
	co = &jwtConfig

	_, err := getKeySet(w)

	assert.NotNil(t, err)
	assert.Equal(t, w.Code, 401, "failed to unmarshal JWK set: EOF")
	assert.Containsf(t, w.Body.String(), "failed to unmarshal JWK set: EOF", "failed to unmarshal JWK set: EOF")
}

func TestValidateJWTWithAnExpiredToken(t *testing.T) {
	privateKey, _ := rsa.GenerateKey(rand.Reader, 2048)
	publicKey := &privateKey.PublicKey
	key, keySet, _ := generateTestJWKMultipleKeys(privateKey, publicKey, "key-id-multiple", 1)
	scpExpiredToken, _ := GenerateTestJWT(key, "scp", true)
	req := httptest.NewRequest("GET", "http://example.com/foo", nil)
	config.InitJWT(&jwtConfig)
	w := httptest.NewRecorder()
	req.Header.Add("Authorization", "Bearer "+scpExpiredToken)
	co = &jwtConfig

	err := ValidateJWT(w, req, keySet)

	assert.NotNil(t, err)
	assert.Equal(t, w.Code, 401, "exp not satisfied")
	assert.Containsf(t, w.Body.String(), "exp", "exp not satisfied")
	assert.Containsf(t, w.Body.String(), "not satisfied", "exp not satisfied")
}

func TestValidateJWTWithoutAnyScopeInTheConfig(t *testing.T) {
	privateKey, _ := rsa.GenerateKey(rand.Reader, 2048)
	publicKey := &privateKey.PublicKey
	key, keySet, _ := generateTestJWKSingleKey(privateKey, publicKey, "key-id-single")
	scopeGoodToken, _ := GenerateTestJWT(key, "scope", false)
	req := httptest.NewRequest("GET", "http://example.com/foo", nil)
	config.InitJWT(&jwtConfig)
	w := httptest.NewRecorder()
	req.Header.Add("Authorization", "Bearer "+scopeGoodToken)
	co = &jwtConfig

	err := ValidateJWT(w, req, keySet)

	assert.NotNil(t, err)
	assert.Equal(t, w.Code, 401, "Invalid Scope")
	assert.Containsf(t, w.Body.String(), "Invalid Scope", "Invalid Scope")
}

func TestValidateJWTWithScopeConfigAndScopeClaimToken(t *testing.T) {
	jwkKeySingle, _, jsonJWKKeySetSingle, jsonJWKKeySetMultiple := GenerateTestKeysAndKeySets()
	scopeGoodToken, _ := GenerateTestJWT(jwkKeySingle, "scope", false)
	ts := CreateTestServer(t, jsonJWKKeySetSingle, jsonJWKKeySetMultiple, 0)
	defer ts.Close()
	jwtConfig.JwksUrl = ts.URL + "/.well-known-single/jwks.json"
	jwtConfig.JwksRefreshInterval = 15
	jwtConfig.AllowedScopes = []string{"scope1"}
	req := httptest.NewRequest("GET", "http://example.com/foo", nil)
	config.InitJWT(&jwtConfig)
	w := httptest.NewRecorder()
	req.Header.Add("Authorization", "Bearer "+scopeGoodToken)
	co = &jwtConfig

	keySet, err := getKeySet(w)
	assert.Nil(t, err)
	err = ValidateJWT(w, req, keySet)

	assert.Nil(t, err)
	assert.Equal(t, w.Code, 200, "Status OK")
	assert.Containsf(t, w.Body.String(), "", "Status OK")
}

func TestValidateJWTWithScopeConfigAndScpClaimToken(t *testing.T) {
	jwkKeySingle, _, jsonJWKKeySetSingle, jsonJWKKeySetMultiple := GenerateTestKeysAndKeySets()
	scpGoodToken, _ := GenerateTestJWT(jwkKeySingle, "scp", false)
	ts := CreateTestServer(t, jsonJWKKeySetSingle, jsonJWKKeySetMultiple, 0)
	defer ts.Close()
	jwtConfig.JwksUrl = ts.URL + "/.well-known-single/jwks.json"
	jwtConfig.JwksRefreshInterval = 15
	jwtConfig.AllowedScopes = []string{"scope1"}
	req := httptest.NewRequest("GET", "http://example.com/foo", nil)
	config.InitJWT(&jwtConfig)
	w := httptest.NewRecorder()
	req.Header.Add("Authorization", "Bearer "+scpGoodToken)
	co = &jwtConfig

	keySet, err := getKeySet(w)
	assert.Nil(t, err)
	err = ValidateJWT(w, req, keySet)

	assert.Nil(t, err)
	assert.Equal(t, w.Code, 200, "Status OK")
	assert.Containsf(t, w.Body.String(), "", "Status OK")
}

func TestValidateJWTWithMultipleKeysInKeySet(t *testing.T) {
	_, jwkKeyMultiple, jsonJWKKeySetSingle, jsonJWKKeySetMultiple := GenerateTestKeysAndKeySets()
	scopeGoodTokenMultiple, _ := GenerateTestJWT(jwkKeyMultiple, "scope", false)
	ts := CreateTestServer(t, jsonJWKKeySetSingle, jsonJWKKeySetMultiple, 0)
	defer ts.Close()
	jwtConfig.JwksUrl = ts.URL + "/.well-known-multiple/jwks.json"
	jwtConfig.JwksRefreshInterval = 15
	jwtConfig.AllowedScopes = []string{"scope1"}
	req := httptest.NewRequest("GET", "http://example.com/foo", nil)
	config.InitJWT(&jwtConfig)
	w := httptest.NewRecorder()
	req.Header.Add("Authorization", "Bearer "+scopeGoodTokenMultiple)
	co = &jwtConfig

	keySet, err := getKeySet(w)
	assert.Nil(t, err)
	err = ValidateJWT(w, req, keySet)

	assert.Nil(t, err)
	assert.Equal(t, w.Code, 200, "Status OK")
	assert.Containsf(t, w.Body.String(), "", "Status OK")
}

func TestJWKSUrlRefreshInterval(t *testing.T) {
	config.InitConfigFromFileOrEnv("../../test/full-setup/config.yml")
	domain := config.Config.Domains["example_com"].Jwt

	config.InitJWT(&domain)

	assert.Equal(t, config.Config.Domains["example_com"].Jwt.JwksRefreshInterval, 60)
}

func TestJWKSUrlFromEnv(t *testing.T) {
	t.Setenv("JWT_JWKS_URL_example_com", "http://testJwksUrlEnv.com")

	config.InitConfigFromFileOrEnv("../../test/full-setup/config.yml")

	assert.Contains(t, config.Config.Domains["example_com"].Jwt.JwksUrl, "http://testJwksUrlEnv.com")
}

func TestJWKSUrlFromEnvAndYaml(t *testing.T) {
	t.Setenv("JWT_JWKS_URL_example_com", "http://testJwksUrlEnv.com")

	config.InitConfigFromFileOrEnv("../../test/full-setup/config.yml")

	assert.Contains(t, config.Config.Domains["example_com"].Jwt.JwksUrl, "http://testJwksUrlEnv.com")
}

func TestJWKSUrlFromYaml(t *testing.T) {
	config.InitConfigFromFileOrEnv("../../test/full-setup/config.yml")

	assert.Contains(t, config.Config.Domains["example_com"].Jwt.JwksUrl, "http://testJwksUrlYaml.com")
}

func TestRefreshKeySet(t *testing.T) {
	t.Skip("To run this test, you should set refreshIntervalDuration (from config.go) to time.Second")
	// To run this test, you should set refreshIntervalDuration (from config.go) to time.Second
	_, _, jsonJWKKeySetSingle, jsonJWKKeySetMultiple := GenerateTestKeysAndKeySets()
	ts := CreateTestServer(t, jsonJWKKeySetSingle, jsonJWKKeySetMultiple, 8081)
	config.Config.Domains = make(config.Domains)
	domainConf := config.Config
	domainConf.Jwt.AllowedScopes = []string{"scope1", "scope2"}
	domainConf.Jwt.JwksUrl = ts.URL + "/.well-known-multiple/jwks.json"
	domainConf.Jwt.JwksRefreshInterval = 1
	domainConf.Jwt.Context = context.Background()
	domainConf.Jwt.Logger = log.New()
	config.InitJWT(&domainConf.Jwt)
	config.Config.Domains["example_com"] = domainConf
	w := httptest.NewRecorder()
	co = &domainConf.Jwt

	keySet1, err := getKeySet(w)
	assert.Nil(t, err)

	ts.Close()
	_, _, jsonJWKKeySetSingle2, jsonJWKKeySetMultiple2 := GenerateTestKeysAndKeySets()
	ts = CreateTestServer(t, jsonJWKKeySetSingle2, jsonJWKKeySetMultiple2, 8081)
	time.Sleep(time.Duration(2) * time.Second)

	keySet2, err := getKeySet(w)
	assert.Nil(t, err)

	assert.NotEqualValues(t, keySet1, keySet2)
	ts.Close()

	_, _, jsonJWKKeySetSingle3, jsonJWKKeySetMultiple3 := GenerateTestKeysAndKeySets()
	ts = CreateTestServer(t, jsonJWKKeySetSingle3, jsonJWKKeySetMultiple3, 8081)
	time.Sleep(time.Duration(2) * time.Second)

	keySet3, err := getKeySet(w)
	assert.Nil(t, err)

	assert.NotEqualValues(t, keySet2, keySet3)
	ts.Close()
}
