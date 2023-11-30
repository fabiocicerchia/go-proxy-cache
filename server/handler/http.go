package handler

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
	"fmt"
	"net/http"
	"net/http/httputil"
	"strconv"
	"strings"
	"time"

	"github.com/fabiocicerchia/go-proxy-cache/config"
	"github.com/fabiocicerchia/go-proxy-cache/logger"
	"github.com/fabiocicerchia/go-proxy-cache/server/cache"
	"github.com/fabiocicerchia/go-proxy-cache/server/response"
	"github.com/fabiocicerchia/go-proxy-cache/server/storage"
	"github.com/fabiocicerchia/go-proxy-cache/server/transport"
	"github.com/fabiocicerchia/go-proxy-cache/telemetry"
	"github.com/fabiocicerchia/go-proxy-cache/telemetry/metrics"
	"github.com/fabiocicerchia/go-proxy-cache/telemetry/tracing"
)

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
	defer tracingSpan.End()

	cached := cache.StatusMiss

	forceFresh := rc.Request.Header.Get(response.CacheBypassHeader) == "1"
	if forceFresh {
		escapedURL := strings.Replace(rc.Request.URL.String(), "\n", "", -1)
		escapedURL = strings.Replace(escapedURL, "\r", "", -1)

		rc.GetLogger().Warningf("Forcing Fresh Content on %s", escapedURL)
	}

	if enableCachedResponse && !forceFresh {
		cached = rc.serveCachedContent(ctx)
	}

	telemetry.From(ctx).RegisterRequestCacheStatus(forceFresh, enableCachedResponse, cache.StatusLabel[cached])

	if cached == cache.StatusMiss {
		rc.Response.Header().Set(response.CacheStatusHeader, response.CacheStatusHeaderMiss)
		rc.serveReverseProxyHTTP(ctx)
	}

	if enableLoggingRequest {
		// HIT and STALE considered the same.
		logger.LogRequest(rc.Request, rc.Response.StatusCode, rc.Response.Content.Len(), rc.ReqID, cached)
	}
}

func (rc RequestCall) serveCachedContent(ctx context.Context) int {
	tracingSpan := tracing.NewChildSpan(ctx, "handler.serve_cached_content")
	defer tracingSpan.End()

	rcDTO := ConvertToRequestCallDTO(rc)

	uriObj, err := storage.RetrieveCachedContent(ctx, rcDTO, rc.GetLogger())
	if err != nil {
		rc.GetLogger().Warnf("Error on serving cached content: %s", err)
		metrics.IncCacheMiss(rc.GetHostname())

		return cache.StatusMiss
	}

	cached := cache.StatusHit
	if uriObj.Stale {
		cached = cache.StatusStale
		rc.Response.Header().Set(response.CacheStatusHeader, response.CacheStatusHeaderStale)
	} else {
		rc.Response.Header().Set(response.CacheStatusHeader, response.CacheStatusHeaderHit)
	}

	telemetry.From(ctx).RegisterWholeResponse(rc.ReqID, rc.Request, rc.Response.StatusCode, rc.Response.Content.Len(), rc.RequestTime, rc.GetScheme(), cached == cache.StatusHit, uriObj.Stale)

	transport.ServeCachedResponse(rc.Request.Context(), rc.Response, uriObj)

	return cached
}

func (rc RequestCall) serveReverseProxyHTTP(ctx context.Context) {
	tracingSpan := tracing.NewChildSpan(ctx, "handler.serve_reverse_proxy_http")
	defer tracingSpan.End()

	proxyURL, err := rc.GetUpstreamURL()
	if err != nil {
		tracing.SetErrorAndFail(tracingSpan, err, "internal error")

		rc.GetLogger().Errorf("Cannot process Upstream URL: %s", err.Error())
		return
	}

	escapedURL := strings.Replace(rc.Request.URL.String(), "\n", "", -1)
	escapedURL = strings.Replace(escapedURL, "\r", "", -1)

	rc.GetLogger().Debugf("ProxyURL: %s", proxyURL.String())
	rc.GetLogger().Debugf("Req URL: %s", escapedURL)
	rc.GetLogger().Debugf("Req Host: %s", rc.Request.Host)

	telemetry.From(ctx).RegisterRequestUpstream(proxyURL, enableCachedResponse, cache.StatusLabel[cache.StatusMiss])

	proxy := httputil.NewSingleHostReverseProxy(&proxyURL)
	proxy.Transport = rc.patchProxyTransport()

	originalDirector := proxy.Director
	gpcDirector := rc.ProxyDirector(ctx)
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

	metrics.IncUpstreamServerResponses(rc.Response.StatusCode, rc.GetHostname(), rc.GetUpstreamHost())
	len, _ := strconv.ParseFloat(rc.Response.Header().Get("Content-Length"), 64)
	metrics.IncUpstreamServerSent(rc.GetHostname(), rc.GetUpstreamHost(), len)
	metrics.IncUpstreamServerResponseTime(rc.GetHostname(), rc.GetUpstreamHost(), float64(time.Since(rc.RequestTime).Milliseconds()))
}

func (rc RequestCall) storeResponse(ctx context.Context) {
	if !enableStoringResponse {
		return
	}

	tracingSpan := tracing.NewChildSpan(ctx, "handler.store_response")
	defer tracingSpan.End()

	rcDTO := ConvertToRequestCallDTO(rc)

	escapedURL := strings.Replace(rc.Request.URL.String(), "\n", "", -1)
	escapedURL = strings.Replace(escapedURL, "\r", "", -1)

	rc.GetLogger().Debugf("Sync Store Response: %s", escapedURL)
	stored, err := doStoreResponse(ctx, rcDTO, rc.DomainConfig.Cache)

	tracing.AddBoolTag(tracingSpan, tracing.TagStorageCached, stored)

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
