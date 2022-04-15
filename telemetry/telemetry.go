package telemetry

import (
	"context"
	"hash"
	"net/http"
	"net/url"

	"github.com/go-http-utils/headers"
	opentracing "github.com/opentracing/opentracing-go"

	"github.com/fabiocicerchia/go-proxy-cache/telemetry/metrics"
	"github.com/fabiocicerchia/go-proxy-cache/telemetry/tracing"
)

// TelemetryContext - Context holder for OpenTelemetry.
type TelemetryContext struct {
	ctx         context.Context
	tracingSpan opentracing.Span
}

// From - Retrieves the tracing span from a context.
func From(ctx context.Context) TelemetryContext {
	return TelemetryContext{
		ctx:         ctx,
		tracingSpan: tracing.SpanFromContext(ctx),
	}
}

// RegisterRedirect - Sends metrics / traces about a HTTP redirect event.
func (tc TelemetryContext) RegisterRedirect(targetURL url.URL) {
	tc.tracingSpan.
		SetTag(tracing.TagResponseLocation, targetURL.String())
}

// RegisterEvent - Registers a specific application event.
func (tc TelemetryContext) RegisterEvent(name string) {
	tracing.AddEventsToSpan(tc.tracingSpan, name, map[string]string{})
}

// RegisterEventWithData - Sends extra data about a specific application event.
func (tc TelemetryContext) RegisterEventWithData(name string, data map[string]string) {
	tracing.AddEventsToSpan(tc.tracingSpan, name, data)
}

// RegisterRequest - Sends metrics / traces about an incoming HTTP request event.
func (tc TelemetryContext) RegisterRequest(req http.Request) {
	tc.tracingSpan.
		SetTag(tracing.TagRequestHost, req.Host).
		SetTag(tracing.TagRequestUrl, req.URL.String()).
		SetTag(tracing.TagRequestMethod, req.Method)
}

// RegisterRequestCall - Sends extra metrics / traces about an incoming HTTP request event.
func (tc TelemetryContext) RegisterRequestCall(reqID string, reqURL url.URL, scheme string, webSocket bool) {
	tc.tracingSpan.
		SetTag(tracing.TagRequestId, reqID).
		SetTag(tracing.TagRequestFullUrl, reqURL.String()).
		SetTag(tracing.TagRequestScheme, scheme).
		SetTag(tracing.TagRequestWebsocket, webSocket)

	metrics.IncUrlScheme(scheme)
}

// RegisterStatusCode - Registers the response status code.
func (tc TelemetryContext) RegisterStatusCode(statusCode int) {
	tc.tracingSpan.
		SetTag(tracing.TagResponseStatusCode, statusCode)
	metrics.IncStatusCode(statusCode)
}

// RegisterWholeResponse - Registers the whole response.
func (tc TelemetryContext) RegisterWholeResponse(reqID string, req http.Request, statusCode int, contentLength int, scheme string, cached bool, stale bool) {
	tc.RegisterCacheStaleOrHit(stale)
	tc.RegisterStatusCode(statusCode)
}

// RegisterRequestCacheStatus - Registers extra metrics / traces about the cache status.
func (tc TelemetryContext) RegisterRequestCacheStatus(forceFresh bool, enableCachedResponse bool, cached string) {
	tc.tracingSpan.
		SetTag(tracing.TagCacheForcedFresh, forceFresh).
		SetTag(tracing.TagCacheCacheable, enableCachedResponse).
		SetTag(tracing.TagCacheCached, cached).
		SetTag(tracing.TagCacheStale, cached == "STALE") // TODO: Magic value
}

// RegisterCacheStaleOrHit - Registers metrics / traces about the cache status.
func (tc TelemetryContext) RegisterCacheStaleOrHit(stale bool) {
	if stale {
		metrics.IncCacheStale()
	} else {
		metrics.IncCacheHit()
	}

	tc.tracingSpan.
		SetTag(tracing.TagCacheStale, stale)
}

// RegisterRequestUpstream - Registers metrics / traces about the proxy upstream.
func (tc TelemetryContext) RegisterRequestUpstream(proxyURL url.URL, enableCachedResponse bool, cached string) {
	tc.tracingSpan.
		SetTag(tracing.TagProxyEndpoint, proxyURL.String()).
		SetTag(tracing.TagCacheForcedFresh, false).
		SetTag(tracing.TagCacheCacheable, enableCachedResponse).
		SetTag(tracing.TagCacheCached, cached).
		SetTag(tracing.TagCacheStale, false)
}

// RegisterLegitRequest - Registers debug details about a specific request whether matches internal configuration.
func (tc TelemetryContext) RegisterLegitRequest(hostMatch bool, legitPort bool, hostname string, listeningPort string, confHostname string, confPort interface{}) {
	tc.tracingSpan.
		SetTag(tracing.TagRequestIsLegitHostnameMatches, hostMatch).
		SetTag(tracing.TagRequestIsLegitPortMatches, legitPort).
		SetTag(tracing.TagRequestIsLegitRequestHostname, hostname).
		SetTag(tracing.TagRequestIsLegitRequestPort, listeningPort).
		SetTag(tracing.TagRequestIsLegitConfHostname, confHostname).
		SetTag(tracing.TagRequestIsLegitConfPort, confPort)
}

// RegisterPurge - Registers metrics / traces about the HTTP Purge event.
func (tc TelemetryContext) RegisterPurge(status bool, err error) {
	tc.tracingSpan.
		SetTag(tracing.TagPurgeStatus, status)

	if err != nil {
		tracing.AddErrorToSpan(tc.tracingSpan, err)
		tracing.Fail(tc.tracingSpan, "internal error")

		// TODO: Add tracing.Fail -> prometheus as failures
	}
}

// RegisterServeOriginal - Registers debug details about a specific request whether should have ETag from original request.
func (tc TelemetryContext) RegisterServeOriginal(hash hash.Hash, header http.Header, statusCode int, lenContent int) {
	tc.tracingSpan.
		SetTag(tracing.TagResponseMustServeOriginalResponseNoHashComputed, hash == nil).
		SetTag(tracing.TagResponseMustServeOriginalResponseEtagPresent, header.Get(headers.ETag)).
		SetTag(tracing.TagResponseMustServeOriginalResponseEtagAlreadyPresent, header.Get(headers.ETag) != "").
		SetTag(tracing.TagResponseMustServeOriginalResponseResponseStatusCode, statusCode).
		SetTag(tracing.TagResponseMustServeOriginalResponseResponseNot2xx, (statusCode < http.StatusOK || statusCode >= http.StatusMultipleChoices)).
		SetTag(tracing.TagResponseMustServeOriginalResponseResponse204, statusCode == http.StatusNoContent).
		SetTag(tracing.TagResponseMustServeOriginalResponseNoBufferedContent, lenContent == 0)
}

// RegisterHostHealth - Registers metrics about the handled domains' health status.
func RegisterHostHealth(healthy int, unhealthy int) {
	metrics.SetHostHealthy(float64(healthy))
	metrics.SetHostUnhealthy(float64(unhealthy))
}
