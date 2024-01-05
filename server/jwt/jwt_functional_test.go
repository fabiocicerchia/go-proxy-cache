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
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/fabiocicerchia/go-proxy-cache/cache/engine"
	"github.com/fabiocicerchia/go-proxy-cache/config"
	logger "github.com/fabiocicerchia/go-proxy-cache/logger"
	"github.com/fabiocicerchia/go-proxy-cache/server/balancer"
	"github.com/fabiocicerchia/go-proxy-cache/server/handler"
	"github.com/fabiocicerchia/go-proxy-cache/telemetry/tracing"
	"github.com/fabiocicerchia/go-proxy-cache/utils"
	circuit_breaker "github.com/fabiocicerchia/go-proxy-cache/utils/circuit-breaker"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func getCommonConfig() config.Configuration {

	return config.Configuration{
		Server: config.Server{
			Upstream: config.Upstream{
				Host:      "www.testing.local",
				Scheme:    "http",
				Endpoints: []string{utils.GetEnv("NGINX_HOST_80", "localhost:40080")},
			},
		},
		CircuitBreaker: circuit_breaker.CircuitBreaker{
			Threshold:   2,                // after 2nd request, if meet FailureRate goes open.
			FailureRate: 0.5,              // 1 out of 2 fails, or more
			Interval:    time.Duration(1), // clears counts immediately
			Timeout:     time.Duration(1), // clears state immediately
		},
		Jwt: config.Jwt{
			Context: context.Background(),
			Logger:  log.New(),
		},
	}
}

func initConfig() {
	domainID := config.Config.Server.Upstream.GetDomainID()
	balancer.InitRoundRobin(domainID, config.Config.Server.Upstream, false)
	circuit_breaker.InitCircuitBreaker(domainID, config.Config.CircuitBreaker, logger.GetGlobal())
	engine.InitConn(domainID, config.Config.Cache, log.StandardLogger())

	engine.GetConn(domainID).Close()
}

func getRequest() (*http.Request, error) {
	req, err := http.NewRequest("GET", "/", nil)
	req.URL.Scheme = config.Config.Server.Upstream.Scheme
	req.URL.Host = config.Config.Server.Upstream.Host
	req.Host = config.Config.Server.Upstream.Host

	return req, err
}

func setHttpConfig() (*httptest.ResponseRecorder, http.Handler) {
	rr := httptest.NewRecorder()
	mux := http.NewServeMux()
	mux.HandleFunc("/", tracing.HTTPHandlerFunc(handler.HandleRequest, "handle_request"))
	var muxMiddleware http.Handler = mux
	h := JWTHandler(muxMiddleware)

	return rr, h
}

func initLogs() {
	log.SetReportCaller(true)
	log.SetLevel(log.DebugLevel)
	log.SetFormatter(&log.TextFormatter{
		ForceColors:     true,
		FullTimestamp:   true,
		TimestampFormat: "2006/01/02 15:04:05",
	})
}

func getCommonConfigWithCache() config.Configuration {
	initLogs()

	return config.Configuration{
		Cache: config.Cache{
			Hosts: []string{utils.GetEnv("REDIS_HOSTS", "localhost:6379")},
			DB:    0,
		},
		CircuitBreaker: circuit_breaker.CircuitBreaker{
			Threshold:   2,   // after 2nd request, if meet FailureRate goes open.
			FailureRate: 0.5, // 1 out of 2 fails, or more
			Interval:    time.Duration(1),
			Timeout:     time.Duration(1), // clears state immediately
		},
		Jwt: config.Jwt{
			Context: context.Background(),
			Logger:  log.New(),
		},
	}
}

func tearDownHTTPFunctional() {
	config.Config = config.Configuration{}
}

func TestJWTValidationWithExcludedPathInDomainConfig(t *testing.T) {
	config.Config = getCommonConfig()
	config.Config.Domains = make(config.Domains)
	domainConf := config.Config
	config.InitJWT(&config.Config.Jwt)
	config.InitJWT(&domainConf.Jwt)
	domainConf.Jwt.ExcludedPaths = []string{"/"}
	config.Config.Domains["testing_local"] = domainConf
	initConfig()
	req, err := getRequest()
	assert.Nil(t, err)
	rr := httptest.NewRecorder()
	mux := http.NewServeMux()
	mux.HandleFunc("/", tracing.HTTPHandlerFunc(handler.HandleRequest, "handle_request"))
	var muxMiddleware http.Handler = mux
	h := JWTHandler(muxMiddleware)

	h.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusMovedPermanently, rr.Code)
	tearDownHTTPFunctional()
}

func TestJWTValidationWithExcludedPathInCommonConfigWithDomain(t *testing.T) {
	config.Config = getCommonConfig()
	config.Config.Domains = make(config.Domains)
	config.Config.Jwt.ExcludedPaths = []string{"/"}
	initConfig()
	req, err := getRequest()
	assert.Nil(t, err)
	rr := httptest.NewRecorder()
	mux := http.NewServeMux()
	mux.HandleFunc("/", tracing.HTTPHandlerFunc(handler.HandleRequest, "handle_request"))
	var muxMiddleware http.Handler = mux
	h := JWTHandler(muxMiddleware)
	domainConf := config.Config
	config.InitJWT(&config.Config.Jwt)
	config.InitJWT(&domainConf.Jwt)
	config.Config.Domains["testing_local"] = domainConf

	h.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusMovedPermanently, rr.Code)
	tearDownHTTPFunctional()
}

func TestJWTValidationWithExcludedPathInCommonConfigWithoutDomain(t *testing.T) {
	config.Config = getCommonConfig()
	config.Config.Jwt.ExcludedPaths = []string{"/"}
	initConfig()
	req, err := getRequest()
	assert.Nil(t, err)
	config.InitJWT(&config.Config.Jwt)
	rr := httptest.NewRecorder()
	mux := http.NewServeMux()
	mux.HandleFunc("/", tracing.HTTPHandlerFunc(handler.HandleRequest, "handle_request"))
	var muxMiddleware http.Handler = mux
	h := JWTHandler(muxMiddleware)

	h.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusMovedPermanently, rr.Code)
	tearDownHTTPFunctional()
}

func TestJWTConfigInDomainConfig(t *testing.T) {
	config.Config = getCommonConfig()
	initConfig()
	req, err := getRequest()
	assert.Nil(t, err)
	jwkKeySingle, _, jsonJWKKeySetSingle, jsonJWKKeySetMultiple := GenerateTestKeysAndKeySets()
	token, _ := GenerateTestJWT(jwkKeySingle, "scope", false)
	req.Header.Add("Authorization", "Bearer "+token)
	ts := CreateTestServer(t, jsonJWKKeySetSingle, jsonJWKKeySetMultiple, 0)
	defer ts.Close()
	config.Config.Domains = make(config.Domains)
	domainConf := config.Config
	domainConf.Jwt.AllowedScopes = []string{"scope1", "scope2"}
	domainConf.Jwt.JwksUrl = ts.URL + "/.well-known-single/jwks.json"
	domainConf.Jwt.JwksRefreshInterval = 15
	config.InitJWT(&config.Config.Jwt)
	config.InitJWT(&domainConf.Jwt)
	config.Config.Domains["testing_local"] = domainConf
	rr, h := setHttpConfig()

	h.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusMovedPermanently, rr.Code)
	assert.Equal(t, "https://testing.local/", rr.HeaderMap["Location"][0])
	assert.Contains(t, rr.Body.String(), `Moved Permanently`)
	tearDownHTTPFunctional()
}

func TestJWTConfigInCommonConfigWithDomain(t *testing.T) {
	config.Config = getCommonConfig()
	initConfig()
	req, err := getRequest()
	assert.Nil(t, err)
	jwkKeySingle, _, jsonJWKKeySetSingle, jsonJWKKeySetMultiple := GenerateTestKeysAndKeySets()
	token, _ := GenerateTestJWT(jwkKeySingle, "scope", false)
	req.Header.Add("Authorization", "Bearer "+token)
	ts := CreateTestServer(t, jsonJWKKeySetSingle, jsonJWKKeySetMultiple, 0)
	defer ts.Close()
	config.Config.Jwt.ExcludedPaths = []string{"/"}
	config.Config.Jwt.AllowedScopes = []string{"scope1"}
	config.Config.Jwt.JwksUrl = ts.URL + "/.well-known-single/jwks.json"
	config.Config.Jwt.JwksRefreshInterval = 15
	config.Config.Domains = make(config.Domains)
	domainConf := config.Config
	config.InitJWT(&config.Config.Jwt)
	config.InitJWT(&domainConf.Jwt)
	config.Config.Domains["testing_local"] = domainConf
	rr, h := setHttpConfig()

	h.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusMovedPermanently, rr.Code)
	assert.Equal(t, "https://testing.local/", rr.HeaderMap["Location"][0])
	assert.Contains(t, rr.Body.String(), `Moved Permanently`)
	tearDownHTTPFunctional()
}

func TestJWTConfigInCommonConfigWithoutDomain(t *testing.T) {
	config.Config = getCommonConfig()
	initConfig()
	req, err := getRequest()
	assert.Nil(t, err)
	jwkKeySingle, _, jsonJWKKeySetSingle, jsonJWKKeySetMultiple := GenerateTestKeysAndKeySets()
	token, _ := GenerateTestJWT(jwkKeySingle, "scope", false)
	req.Header.Add("Authorization", "Bearer "+token)
	ts := CreateTestServer(t, jsonJWKKeySetSingle, jsonJWKKeySetMultiple, 0)
	defer ts.Close()
	config.Config.Jwt.ExcludedPaths = []string{"/"}
	config.Config.Jwt.AllowedScopes = []string{"scope1"}
	config.Config.Jwt.JwksUrl = ts.URL + "/.well-known-single/jwks.json"
	config.Config.Jwt.JwksRefreshInterval = 15
	config.InitJWT(&config.Config.Jwt)
	rr, h := setHttpConfig()

	h.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusMovedPermanently, rr.Code)
	assert.Equal(t, "https://testing.local/", rr.HeaderMap["Location"][0])
	assert.Contains(t, rr.Body.String(), `Moved Permanently`)
	tearDownHTTPFunctional()
}

func TestJWTMiddlewareValidatesWithToken(t *testing.T) {
	config.Config = getCommonConfigWithCache()
	config.Config.Jwt.AllowedScopes = []string{"scope1"}
	jwkKeySingle, _, jsonJWKKeySetSingle, jsonJWKKeySetMultiple := GenerateTestKeysAndKeySets()
	token, _ := GenerateTestJWT(jwkKeySingle, "scp", false)
	ts := CreateTestServer(t, jsonJWKKeySetSingle, jsonJWKKeySetMultiple, 0)
	defer ts.Close()
	config.Config.Jwt.JwksUrl = ts.URL + "/.well-known-single/jwks.json"
	config.Config.Jwt.JwksRefreshInterval = 15
	initConfig()
	req, err := http.NewRequest("GET", "/", nil)
	assert.Nil(t, err)
	config.InitJWT(&config.Config.Jwt)
	req.Header.Add("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()
	mux := http.NewServeMux()
	mux.HandleFunc("/", tracing.HTTPHandlerFunc(handler.HandleRequest, "handle_request"))
	var muxMiddleware http.Handler = mux
	h := JWTHandler(muxMiddleware)

	h.ServeHTTP(rr, req)

	assert.Nil(t, err)
	assert.Equal(t, http.StatusBadGateway, rr.Code)
}

func TestJWTMiddlewareWithoutJWTValidation(t *testing.T) {
	config.Config = getCommonConfigWithCache()
	config.Config.Jwt.ExcludedPaths = []string{"/"}
	initConfig()
	config.InitJWT(&config.Config.Jwt)
	req, err := http.NewRequest("GET", "/", nil)
	assert.Nil(t, err)
	rr := httptest.NewRecorder()
	mux := http.NewServeMux()
	mux.HandleFunc("/", tracing.HTTPHandlerFunc(handler.HandleRequest, "handle_request"))
	var muxMiddleware http.Handler = mux
	h := JWTHandler(muxMiddleware)

	h.ServeHTTP(rr, req)

	assert.Nil(t, err)
	assert.Equal(t, http.StatusBadGateway, rr.Code)
}
