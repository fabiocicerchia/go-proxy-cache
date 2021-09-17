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
	"context"
	"fmt"
	"net/http"
	"net/http/httputil"
	"time"

	"github.com/fabiocicerchia/go-proxy-cache/config"
	"github.com/fabiocicerchia/go-proxy-cache/server/logger"
	"github.com/fabiocicerchia/go-proxy-cache/server/response"
	"github.com/fabiocicerchia/go-proxy-cache/server/storage"
	"github.com/fabiocicerchia/go-proxy-cache/server/transport"
	"github.com/fabiocicerchia/go-proxy-cache/telemetry"
	"github.com/fabiocicerchia/go-proxy-cache/telemetry/metrics"
	"github.com/fabiocicerchia/go-proxy-cache/telemetry/tracing"
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
func (rc RequestCall) HandleHTTPRequestAndProxy(ctx context.Context) {
	tracingSpan := tracing.NewChildSpan("handler.handle_http_request_and_proxy", ctx)
	defer tracingSpan.Finish()

	cached := CacheStatusMiss

	forceFresh := rc.Request.Header.Get(response.CacheBypassHeader) == "1"
	if forceFresh {
		rc.GetLogger().Warningf("Forcing Fresh Content on %v", rc.Request.URL.String())
	}

	if enableCachedResponse && !forceFresh {
		cached = rc.serveCachedContent(ctx)
	}

	telemetry.From(ctx).RegisterRequestCacheStatus(forceFresh, enableCachedResponse, CacheStatusLabel[cached])

	if cached == CacheStatusMiss {
		rc.Response.Header().Set(response.CacheStatusHeader, response.CacheStatusHeaderMiss)
		rc.serveReverseProxyHTTP(ctx)
	}

	if enableLoggingRequest {
		// HIT and STALE considered the same.
		logger.LogRequest(rc.Request, *rc.Response, rc.ReqID, cached != CacheStatusMiss, CacheStatusLabel[cached])
	}
}

func (rc RequestCall) serveCachedContent(ctx context.Context) int {
	tracingSpan := tracing.NewChildSpan("handler.serve_cached_content", ctx)
	defer tracingSpan.Finish()

	rcDTO := ConvertToRequestCallDTO(rc)

	uriObj, err := storage.RetrieveCachedContent(rcDTO, rc.GetLogger())
	if err != nil {
		rc.GetLogger().Warnf("Error on serving cached content: %s", err)
		metrics.IncCacheMiss()

		return CacheStatusMiss
	}

	cached := CacheStatusHit
	if uriObj.Stale {
		cached = CacheStatusStale
		rc.Response.Header().Set(response.CacheStatusHeader, response.CacheStatusHeaderStale)
	} else {
		rc.Response.Header().Set(response.CacheStatusHeader, response.CacheStatusHeaderHit)
	}

	telemetry.From(ctx).RegisterCacheStaleOrHit(uriObj.Stale)
	telemetry.From(ctx).RegisterStatusCode(rc.Response.StatusCode)

	transport.ServeCachedResponse(rc.Request.Context(), rc.Response, uriObj)

	return cached
}

func (rc RequestCall) serveReverseProxyHTTP(ctx context.Context) {
	tracingSpan := tracing.NewChildSpan("handler.serve_reverse_proxy_http", ctx)
	defer tracingSpan.Finish()

	proxyURL, err := rc.GetUpstreamURL()
	if err != nil {
		tracing.AddErrorToSpan(tracingSpan, err)
		tracing.Fail(tracingSpan, "internal error")

		rc.GetLogger().Errorf("Cannot process Upstream URL: %s", err.Error())
		return
	}

	rc.GetLogger().Debugf("ProxyURL: %s", proxyURL.String())
	rc.GetLogger().Debugf("Req URL: %s", rc.Request.URL.String())
	rc.GetLogger().Debugf("Req Host: %s", rc.Request.Host)

	telemetry.From(ctx).RegisterRequestUpstream(proxyURL, enableCachedResponse, CacheStatusLabel[CacheStatusMiss])

	proxy := httputil.NewSingleHostReverseProxy(&proxyURL)
	proxy.Transport = rc.patchProxyTransport()

	originalDirector := proxy.Director
	gpcDirector := rc.ProxyDirector(tracingSpan)
	proxy.Director = func(req *http.Request) {
		// the default director implementation returned by httputil.NewSingleHostReverseProxy
		// takes care of setting the request Scheme, Host, and Path.
		originalDirector(req)
		gpcDirector(req)
	}

	serveNotModified := rc.GetResponseWithETag(ctx, proxy)
	if serveNotModified {
		rc.SendNotModifiedResponse(ctx)
		return
	}

	if rc.DomainConfig.Server.GZip {
		WrapResponseForGZip(rc.Response, &rc.Request)
	}

	rc.SendResponse(ctx)
	rc.storeResponse(ctx)
}

func (rc RequestCall) storeResponse(ctx context.Context) {
	if !enableStoringResponse {
		return
	}

	tracingSpan := tracing.NewChildSpan("handler.store_response", ctx)
	defer tracingSpan.Finish()

	rcDTO := ConvertToRequestCallDTO(rc)

	rc.GetLogger().Debugf("Sync Store Response: %s", rc.Request.URL.String())
	stored, err := doStoreResponse(rcDTO, rc.DomainConfig.Cache)

	tracingSpan.SetTag(tracing.TagStorageCached, stored)

	if err != nil {
		tracing.AddErrorToSpan(tracingSpan, err)
	}
}

func doStoreResponse(rcDTO storage.RequestCallDTO, configCache config.Cache) (bool, error) {
	stored, err := storage.StoreGeneratedPage(rcDTO, configCache)
	if !stored || err != nil {
		logger.Log(rcDTO.Request, rcDTO.ReqID, fmt.Sprintf("Not Stored: %v", err))
	}

	return stored, err
}
