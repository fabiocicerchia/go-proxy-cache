package handler

//                                                                         __
// .-----.-----.______.-----.----.-----.--.--.--.--.______.----.---.-.----|  |--.-----.
// |  _  |  _  |______|  _  |   _|  _  |_   _|  |  |______|  __|  _  |  __|     |  -__|
// |___  |_____|      |   __|__| |_____|__.__|___  |      |____|___._|____|__|__|_____|
// |_____|            |__|                   |_____|
//
// Copyright (c) 2022 Fabio Cicerchia. https://fabiocicerchia.it. MIT License
// Repo: https://github.com/fabiocicerchia/go-proxy-cache

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httputil"
	"strings"
	"time"

	"github.com/fabiocicerchia/go-proxy-cache/config"
	"github.com/fabiocicerchia/go-proxy-cache/logger"
	"github.com/fabiocicerchia/go-proxy-cache/server/response"
	"github.com/fabiocicerchia/go-proxy-cache/server/storage"
	"github.com/fabiocicerchia/go-proxy-cache/server/transport"
	"github.com/fabiocicerchia/go-proxy-cache/telemetry"
	"github.com/fabiocicerchia/go-proxy-cache/telemetry/metrics"
	"github.com/fabiocicerchia/go-proxy-cache/telemetry/tracing"
)

// CacheStatusHit - Value for HIT.
const CacheStatusHit = 1

// CacheStatusMiss - Value for MISS.
const CacheStatusMiss = 0

// CacheStatusStale - Value for STALE.
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
	tracingSpan := tracing.NewChildSpan(ctx, "handler.handle_http_request_and_proxy")
	defer tracingSpan.Finish()

	cached := CacheStatusMiss

	forceFresh := rc.Request.Header.Get(response.CacheBypassHeader) == "1"
	if forceFresh {
		escapedURL := strings.Replace(rc.Request.URL.String(), "\n", "", -1)
		escapedURL = strings.Replace(escapedURL, "\r", "", -1)

		rc.GetLogger().Warningf("Forcing Fresh Content on %s", escapedURL)
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
		logger.LogRequest(rc.Request, rc.Response.StatusCode, rc.Response.Content.Len(), rc.ReqID, cached != CacheStatusMiss, CacheStatusLabel[cached])
	}
}

func (rc RequestCall) serveCachedContent(ctx context.Context) int {
	tracingSpan := tracing.NewChildSpan(ctx, "handler.serve_cached_content")
	defer tracingSpan.Finish()

	rcDTO := ConvertToRequestCallDTO(rc)

	uriObj, err := storage.RetrieveCachedContent(ctx, rcDTO, rc.GetLogger())
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
	tracingSpan := tracing.NewChildSpan(ctx, "handler.serve_reverse_proxy_http")
	defer tracingSpan.Finish()

	proxyURL, err := rc.GetUpstreamURL()
	if err != nil {
		tracing.AddErrorToSpan(tracingSpan, err)
		tracing.Fail(tracingSpan, "internal error")

		rc.GetLogger().Errorf("Cannot process Upstream URL: %s", err.Error())
		return
	}

	escapedURL := strings.Replace(rc.Request.URL.String(), "\n", "", -1)
	escapedURL = strings.Replace(escapedURL, "\r", "", -1)

	rc.GetLogger().Debugf("ProxyURL: %s", proxyURL.String())
	rc.GetLogger().Debugf("Req URL: %s", escapedURL)
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

	tracingSpan := tracing.NewChildSpan(ctx, "handler.store_response")
	defer tracingSpan.Finish()

	rcDTO := ConvertToRequestCallDTO(rc)

	escapedURL := strings.Replace(rc.Request.URL.String(), "\n", "", -1)
	escapedURL = strings.Replace(escapedURL, "\r", "", -1)

	rc.GetLogger().Debugf("Sync Store Response: %s", escapedURL)
	stored, err := doStoreResponse(ctx, rcDTO, rc.DomainConfig.Cache)

	tracingSpan.SetTag(tracing.TagStorageCached, stored)

	if err != nil {
		tracing.AddErrorToSpan(tracingSpan, err)
	}
}

func doStoreResponse(ctx context.Context, rcDTO storage.RequestCallDTO, configCache config.Cache) (bool, error) {
	stored, err := storage.StoreGeneratedPage(ctx, rcDTO, configCache)
	if !stored || err != nil {
		logger.Log(rcDTO.Request, rcDTO.ReqID, fmt.Sprintf("Not Stored: %v", err))
	}

	return stored, err
}
