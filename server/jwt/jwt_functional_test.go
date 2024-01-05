//go:build all || functional
// +build all functional

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
	"crypto/tls"
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

func TestJWTValidationPathConfig(t *testing.T) {
	// With JWT validation path config in domain config
	config.Config = config.Configuration{
		Server: config.Server{
			Upstream: config.Upstream{
				Host:   "example.com",
				Scheme: "https",
			},
		},
		CircuitBreaker: circuit_breaker.CircuitBreaker{
			Threshold:   2,                // after 2nd request, if meet FailureRate goes open.
			FailureRate: 0.5,              // 1 out of 2 fails, or more
			Interval:    time.Duration(1), // clears counts immediately
			Timeout:     time.Duration(1), // clears state immediately
		},
	}
	config.Config.Domains = make(config.Domains)
	domainConf := config.Config
	domainConf.Jwt.IncludedPaths = []string{"/"}
	config.Config.Domains["example_com"] = domainConf

	domainID := config.Config.Server.Upstream.GetDomainID()
	balancer.InitRoundRobin(domainID, config.Config.Server.Upstream, false)
	circuit_breaker.InitCircuitBreaker(domainID, config.Config.CircuitBreaker, logger.GetGlobal())
	engine.InitConn(domainID, config.Config.Cache, log.StandardLogger())

	engine.GetConn(domainID).Close()

	req, err := http.NewRequest("GET", "/", nil)
	req.URL.Scheme = config.Config.Server.Upstream.Scheme
	req.URL.Host = config.Config.Server.Upstream.Host
	req.Host = config.Config.Server.Upstream.Host
	req.TLS = &tls.ConnectionState{} // mock a fake https
	assert.Nil(t, err)
	rr := httptest.NewRecorder()
	mux := http.NewServeMux()
	mux.HandleFunc("/", tracing.HTTPHandlerFunc(handler.HandleRequest, "handle_request"))
	var muxMiddleware http.Handler = mux
	h := JWTHandler(muxMiddleware)

	h.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusUnauthorized, rr.Code)

	// With JWT validation path config in common config (with domain)
	domainConf = config.Config
	domainConf.Jwt.IncludedPaths = nil
	config.Config.Domains["example_com"] = domainConf
	config.Config.Jwt.IncludedPaths = []string{"/"}

	h.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusUnauthorized, rr.Code)

	// With JWT validation path config in common config (without domain)
	domainConf = config.Config
	config.Config.Jwt.IncludedPaths = []string{"/"}
	config.Config.Domains = make(map[string]config.Configuration)

	h.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusUnauthorized, rr.Code)

	tearDownHTTPFunctional()
}

func TestHTTPEndToEndCallWithoutCacheWithJWTCompleteConfig(t *testing.T) {
	// With JWT complete config in domain config
	config.Config = config.Configuration{
		Server: config.Server{
			Upstream: config.Upstream{
				Host:   "example.com",
				Scheme: "https",
			},
		},
		CircuitBreaker: circuit_breaker.CircuitBreaker{
			Threshold:   2,                // after 2nd request, if meet FailureRate goes open.
			FailureRate: 0.5,              // 1 out of 2 fails, or more
			Interval:    time.Duration(1), // clears counts immediately
			Timeout:     time.Duration(1), // clears state immediately
		},
	}

	domainID := config.Config.Server.Upstream.GetDomainID()
	balancer.InitRoundRobin(domainID, config.Config.Server.Upstream, false)
	circuit_breaker.InitCircuitBreaker(domainID, config.Config.CircuitBreaker, logger.GetGlobal())
	engine.InitConn(domainID, config.Config.Cache, log.StandardLogger())

	engine.GetConn(domainID).Close()

	req, err := http.NewRequest("GET", "/", nil)

	jwkKeySingle, _, jsonJWKKeySetSingle, jsonJWKKeySetMultiple := GenerateTestKeysAndKeySets()
	token, _ := GenerateTestJWT(jwkKeySingle, "scope", false)
	req.Header.Add("Authorization", "Bearer "+token)
	ts := CreateTestServer(t, jsonJWKKeySetSingle, jsonJWKKeySetMultiple)
	defer ts.Close()

	config.Config.Domains = make(config.Domains)
	domainConf := config.Config
	domainConf.Jwt.IncludedPaths = []string{"/"}
	domainConf.Jwt.AllowedScopes = []string{"scope1", "scope2"}
	domainConf.Jwt.JwksUrl = ts.URL + "/.well-known-single/jwks.json"
	config.Config.Domains["example_com"] = domainConf

	req.URL.Scheme = config.Config.Server.Upstream.Scheme
	req.URL.Host = config.Config.Server.Upstream.Host
	req.Host = config.Config.Server.Upstream.Host
	req.TLS = &tls.ConnectionState{} // mock a fake https
	assert.Nil(t, err)
	rr := httptest.NewRecorder()
	mux := http.NewServeMux()
	mux.HandleFunc("/", tracing.HTTPHandlerFunc(handler.HandleRequest, "handle_request"))
	var muxMiddleware http.Handler = mux
	h := JWTHandler(muxMiddleware)

	h.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "MISS", rr.HeaderMap["X-Go-Proxy-Cache-Status"][0])

	// With JWT complete config in common config (with domain)
	domainConf = config.Config
	domainConf.Jwt.IncludedPaths = nil
	domainConf.Jwt.AllowedScopes = nil
	domainConf.Jwt.JwksUrl = ""
	config.Config.Domains["example_com"] = domainConf
	config.Config.Jwt.IncludedPaths = []string{"/"}
	config.Config.Jwt.AllowedScopes = []string{"scope1"}
	config.Config.Jwt.JwksUrl = ts.URL + "/.well-known-single/jwks.json"

	h.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "MISS", rr.HeaderMap["X-Go-Proxy-Cache-Status"][0])

	// With JWT complete config in common config (without domain)
	domainConf = config.Config
	config.Config.Jwt.IncludedPaths = []string{"/"}
	config.Config.Jwt.AllowedScopes = []string{"scope1"}
	config.Config.Jwt.JwksUrl = ts.URL + "/.well-known-single/jwks.json"
	config.Config.Domains = make(map[string]config.Configuration)

	h.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	assert.Equal(t, "MISS", rr.HeaderMap["X-Go-Proxy-Cache-Status"][0])

	tearDownHTTPFunctional()
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

func getCommonConfig() config.Configuration {
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
			IncludedPaths: []string{"/"},
		},
	}
}

func TestJWTMiddlewareValidatesWithNoToken(t *testing.T) {
	config.Config = getCommonConfig()

	domainID := config.Config.Server.Upstream.GetDomainID()
	circuit_breaker.InitCircuitBreaker(domainID, config.Config.CircuitBreaker, logger.GetGlobal())
	engine.InitConn(domainID, config.Config.Cache, log.StandardLogger())
	engine.GetConn(domainID).Close()

	req, err := http.NewRequest("GET", "/", nil)
	assert.Nil(t, err)

	rr := httptest.NewRecorder()
	mux := http.NewServeMux()
	mux.HandleFunc("/", tracing.HTTPHandlerFunc(handler.HandleRequest, "handle_request"))
	var muxMiddleware http.Handler = mux
	h := JWTHandler(muxMiddleware)

	h.ServeHTTP(rr, req)

	assert.Nil(t, err)
	assert.Equal(t, http.StatusUnauthorized, rr.Code)

	engine.InitConn(domainID, config.Config.Cache, log.StandardLogger())
}

func TestJWTMiddlewareValidatesWithToken(t *testing.T) {
	config.Config = getCommonConfig()
	config.Config.Jwt.AllowedScopes = []string{"scope1"}
	jwkKeySingle, _, jsonJWKKeySetSingle, jsonJWKKeySetMultiple := GenerateTestKeysAndKeySets()
	token, _ := GenerateTestJWT(jwkKeySingle, "scp", false)
	ts := CreateTestServer(t, jsonJWKKeySetSingle, jsonJWKKeySetMultiple)
	defer ts.Close()
	config.Config.Jwt.JwksUrl = ts.URL + "/.well-known-single/jwks.json"

	domainID := config.Config.Server.Upstream.GetDomainID()
	circuit_breaker.InitCircuitBreaker(domainID, config.Config.CircuitBreaker, logger.GetGlobal())
	engine.InitConn(domainID, config.Config.Cache, log.StandardLogger())
	engine.GetConn(domainID).Close()

	req, err := http.NewRequest("GET", "/", nil)
	assert.Nil(t, err)
	req.Header.Add("Authorization", "Bearer "+token)

	rr := httptest.NewRecorder()
	mux := http.NewServeMux()
	mux.HandleFunc("/", tracing.HTTPHandlerFunc(handler.HandleRequest, "handle_request"))
	var muxMiddleware http.Handler = mux
	h := JWTHandler(muxMiddleware)

	h.ServeHTTP(rr, req)

	assert.Nil(t, err)
	assert.Equal(t, http.StatusBadGateway, rr.Code)

	engine.InitConn(domainID, config.Config.Cache, log.StandardLogger())
}

func TestJWTMiddlewareWithoutJWTValidation(t *testing.T) {
	config.Config = getCommonConfig()
	config.Config.Jwt.IncludedPaths = nil

	domainID := config.Config.Server.Upstream.GetDomainID()
	circuit_breaker.InitCircuitBreaker(domainID, config.Config.CircuitBreaker, logger.GetGlobal())
	engine.InitConn(domainID, config.Config.Cache, log.StandardLogger())
	engine.GetConn(domainID).Close()

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

	engine.InitConn(domainID, config.Config.Cache, log.StandardLogger())
}

func tearDownHTTPFunctional() {
	config.Config = config.Configuration{}
}
