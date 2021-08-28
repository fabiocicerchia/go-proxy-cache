package handler

//                                                                         __
// .-----.-----.______.-----.----.-----.--.--.--.--.______.----.---.-.----|  |--.-----.
// |  _  |  _  |______|  _  |   _|  _  |_   _|  |  |______|  __|  _  |  __|     |  -__|
// |___  |_____|      |   __|__| |_____|__.__|___  |      |____|___._|____|__|__|_____|
// |_____|            |__|                   |_____|
//
// Copyright (c) 2020 Fabio Cicerchia. https://fabiocicerchia.it. MIT License
// Repo: https://github.com/fabiocicerchia/go-proxy-cache

import (
	"fmt"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/fabiocicerchia/go-proxy-cache/config"
	"github.com/fabiocicerchia/go-proxy-cache/server/balancer"
	"github.com/fabiocicerchia/go-proxy-cache/server/logger"
	"github.com/fabiocicerchia/go-proxy-cache/server/response"
	"github.com/fabiocicerchia/go-proxy-cache/server/storage"
	"github.com/fabiocicerchia/go-proxy-cache/server/transport"
	"github.com/fabiocicerchia/go-proxy-cache/utils"
	"github.com/fabiocicerchia/go-proxy-cache/utils/queue"
)

// CacheStatusHit - Value for HIT.
const CacheStatusHit = 1

// CacheStatusHit - Value for MISS.
const CacheStatusMiss = 0

// CacheStatusHit - Value for STALE.
const CacheStatusStale = -1

// CacheStatusLabel - Labels used for displaying HIT/MISS based on cache usage.
var CacheStatusLabel = map[int]string{
	CacheStatusHit:   "HIT",
	CacheStatusMiss:  "MISS",
	CacheStatusStale: "STALE",
}

var enableStoringResponse = true
var enableCachedResponse = true
var enableLoggingRequest = true

// DefaultTransportMaxIdleConns - Default value used for http.Transport.MaxIdleConns.
var DefaultTransportMaxIdleConns int = 1000

// DefaultTransportMaxIdleConnsPerHost - Default value used for http.Transport.MaxIdleConnsPerHost.
var DefaultTransportMaxIdleConnsPerHost int = 1000

// DefaultTransportMaxConnsPerHost - Default value used for http.Transport.MaxConnsPerHost.
var DefaultTransportMaxConnsPerHost int = 1000

// DefaultTransportDialTimeout - Default value used for net.Dialer.Timeout
var DefaultTransportDialTimeout time.Duration = 15 * time.Second

// HandleHTTPRequestAndProxy - Handles the HTTP requests and proxies to backend server.
func (rc RequestCall) HandleHTTPRequestAndProxy() {
	cached := CacheStatusMiss

	if enableCachedResponse {
		cached = rc.serveCachedContent()
	}

	if cached == CacheStatusMiss {
		rc.serveReverseProxyHTTP()
	}

	if enableLoggingRequest {
		// HIT and STALE considered the same.
		logger.LogRequest(*rc.Request, *rc.Response, cached != CacheStatusMiss, CacheStatusLabel[cached])
	}
}

func (rc RequestCall) serveCachedContent() int {
	rcDTO := ConvertToRequestCallDTO(rc)

	uriObj, err := storage.RetrieveCachedContent(rcDTO)
	if err != nil {
		rc.Response.Header().Set(response.CacheStatusHeader, response.CacheStatusHeaderMiss)

		log.Warnf("Error on serving cached content: %s", err)

		return CacheStatusMiss
	}

	cached := CacheStatusHit
	if uriObj.Stale {
		cached = CacheStatusStale
		rc.Response.Header().Set(response.CacheStatusHeader, response.CacheStatusHeaderStale)
	} else {
		rc.Response.Header().Set(response.CacheStatusHeader, response.CacheStatusHeaderHit)
	}

	transport.ServeCachedResponse(rc.Request.Context(), rc.Response, uriObj)

	return cached
}

func (rc RequestCall) serveReverseProxyHTTP() {
	upstream := rc.DomainConfig.Server.Upstream
	proxyURL := rc.patchRequestForReverseProxy(upstream)

	log.Debugf("ProxyURL: %s", proxyURL.String())
	log.Debugf("Req URL: %s", rc.Request.URL.String())
	log.Debugf("Req Host: %s", rc.Request.Host)

	proxy := httputil.NewSingleHostReverseProxy(proxyURL)
	proxy.Transport = rc.patchProxyTransport()

	director := proxy.Director

	proxy.Director = func(req *http.Request) {
		// the default director implementation returned by httputil.NewSingleHostReverseProxy
		// takes care of setting the request Scheme, Host, and Path.
		director(req)

		// TODO: Move patchRequestForReverseProxy in here
	}

	proxy.ServeHTTP(rc.Response, rc.Request)

	rc.storeResponse()
}

func (rc RequestCall) storeResponse() {
	if !enableStoringResponse {
		return
	}

	// Make it sync for testing
	// TODO: Make it customizable?
	if os.Getenv("GPC_SYNC_STORING") == "1" {
		log.Debugf("Sync Store Response: %s", rc.Request.URL.String())

		rc.doStoreResponse()
		return
	}

	log.Debugf("Async Store Response: %s", rc.Request.URL.String())
	queue.Dispatcher.Do(func() {
		rc.doStoreResponse()
	})
}

func (rc RequestCall) doStoreResponse() {
	rcDTO := ConvertToRequestCallDTO(rc)

	stored, err := storage.StoreGeneratedPage(rcDTO, rc.DomainConfig.Cache)
	if !stored || err != nil {
		logger.Log(*rc.Request, fmt.Sprintf("Not Stored: %v", err))
	}
}

// FixRequest - Fixes the Request in order to use the load balanced host.
func (rc *RequestCall) FixRequest(url url.URL, upstream config.Upstream) {
	scheme := upstream.Scheme
	if scheme == config.SchemeWildcard {
		scheme = rc.GetScheme()
	}
	host := utils.IfEmpty(upstream.Host, url.Host)

	lbID := upstream.Host + utils.StringSeparatorOne + upstream.Scheme
	balancedHost := balancer.GetLBRoundRobin(lbID, url.Host)
	overridePort := getOverridePort(balancedHost, upstream.Port, scheme)

	// The value of r.URL.Host and r.Host are almost always different. On a
	// proxy server, r.URL.Host is the host of the target server and r.Host is
	// the host of the proxy server itself.
	// Ref: https://stackoverflow.com/a/42926149/888162
	rc.Request.Header.Set("X-Forwarded-Host", rc.Request.Header.Get("Host"))

	rc.Request.Header.Set("X-Forwarded-Proto", rc.GetScheme())

	previousXForwardedFor := rc.Request.Header.Get("X-Forwarded-For")
	clientIP := utils.StripPort(rc.Request.RemoteAddr)

	xForwardedFor := net.ParseIP(clientIP).String()
	if previousXForwardedFor != "" {
		xForwardedFor = previousXForwardedFor + ", " + xForwardedFor
	}

	rc.Request.Header.Set("X-Forwarded-For", xForwardedFor)

	rc.Request.URL.Host = balancedHost + overridePort
	rc.Request.URL.Scheme = scheme
	rc.Request.Host = host
}
