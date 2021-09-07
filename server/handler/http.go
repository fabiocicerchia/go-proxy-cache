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
	"net/http"
	"net/http/httputil"
	"os"
	"time"

	"github.com/fabiocicerchia/go-proxy-cache/config"
	"github.com/fabiocicerchia/go-proxy-cache/server/logger"
	"github.com/fabiocicerchia/go-proxy-cache/server/response"
	"github.com/fabiocicerchia/go-proxy-cache/server/storage"
	"github.com/fabiocicerchia/go-proxy-cache/server/transport"
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

const enableStoringResponse = true
const enableCachedResponse = true
const enableLoggingRequest = true

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

	forceFresh := rc.Request.Header.Get(response.CacheBypassHeader) == "1"
	if forceFresh {
		rc.GetLogger().Warningf("Forcing Fresh Content on %v", rc.Request.URL.String())
	}

	if enableCachedResponse && !forceFresh {
		cached = rc.serveCachedContent()
	}

	if cached == CacheStatusMiss {
		rc.Response.Header().Set(response.CacheStatusHeader, response.CacheStatusHeaderMiss)
		rc.serveReverseProxyHTTP()
	}

	if enableLoggingRequest {
		// HIT and STALE considered the same.
		logger.LogRequest(rc.Request, *rc.Response, rc.ReqID, cached != CacheStatusMiss, CacheStatusLabel[cached])
	}
}

func (rc RequestCall) serveCachedContent() int {
	rcDTO := ConvertToRequestCallDTO(rc)

	uriObj, err := storage.RetrieveCachedContent(rcDTO, rc.GetLogger())
	if err != nil {
		rc.GetLogger().Warnf("Error on serving cached content: %s", err)

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
	proxyURL, err := rc.GetUpstreamURL()
	if err != nil {
		rc.GetLogger().Errorf("Cannot process Upstream URL: %s", err.Error())
		return
	}

	rc.GetLogger().Debugf("ProxyURL: %s", proxyURL.String())
	rc.GetLogger().Debugf("Req URL: %s", rc.Request.URL.String())
	rc.GetLogger().Debugf("Req Host: %s", rc.Request.Host)

	proxy := httputil.NewSingleHostReverseProxy(&proxyURL)
	proxy.Transport = rc.patchProxyTransport()

	originalDirector := proxy.Director
	gpcDirector := rc.ProxyDirector
	proxy.Director = func(req *http.Request) {
		// the default director implementation returned by httputil.NewSingleHostReverseProxy
		// takes care of setting the request Scheme, Host, and Path.
		originalDirector(req)
		gpcDirector(req)
	}

	serveNotModified := rc.GetResponseWithETag(proxy)
	if serveNotModified {
		rc.Response.SendNotModifiedResponse()
		return
	}

	if rc.DomainConfig.Server.GZip {
		WrapResponseForGZip(rc.Response, &rc.Request)
	}

	rc.Response.SendResponse()

	rc.storeResponse()
}

func (rc RequestCall) storeResponse() {
	if !enableStoringResponse {
		return
	}

	rcDTO := ConvertToRequestCallDTO(rc)

	// Make it sync for testing
	// TODO: Make it customizable?
	if os.Getenv("GPC_SYNC_STORING") == "1" {
		rc.GetLogger().Debugf("Sync Store Response: %s", rc.Request.URL.String())

		doStoreResponse(rcDTO, rc.DomainConfig.Cache)
		return
	}

	rc.GetLogger().Debugf("Async Store Response: %s", rc.Request.URL.String())
	go doStoreResponse(rcDTO, rc.DomainConfig.Cache)
}

func doStoreResponse(rcDTO storage.RequestCallDTO, configCache config.Cache) {
	stored, err := storage.StoreGeneratedPage(rcDTO, configCache)
	if !stored || err != nil {
		logger.Log(rcDTO.Request, rcDTO.ReqID, fmt.Sprintf("Not Stored: %v", err))
	}
}
