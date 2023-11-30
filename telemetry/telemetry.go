package telemetry

import (
	"context"
	"hash"
	"net/http"
	"net/url"
	"time"

	"github.com/go-http-utils/headers"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"github.com/fabiocicerchia/go-proxy-cache/telemetry/metrics"
	"github.com/fabiocicerchia/go-proxy-cache/telemetry/tracing"
)

// TelemetryContext - Context holder for OpenTelemetry.
type TelemetryContext struct {
	ctx         context.Context
	tracingSpan trace.Span
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
		SetAttributes(attribute.Key(tracing.TagResponseLocation).String(targetURL.String()))
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
		SetAttributes(attribute.Key(tracing.TagRequestHost).String(req.Host))
	tc.tracingSpan.
		SetAttributes(attribute.Key(tracing.TagRequestUrl).String(req.URL.String()))
	tc.tracingSpan.
		SetAttributes(attribute.Key(tracing.TagRequestMethod).String(req.Method))
}

// RegisterRequestCall - Sends extra metrics / traces about an incoming HTTP request event.
func (tc TelemetryContext) RegisterRequestCall(reqID string, req http.Request, reqURL url.URL, scheme string, webSocket bool) {
	tc.tracingSpan.
		SetAttributes(attribute.Key(tracing.TagRequestId).String(reqID))
	tc.tracingSpan.
		SetAttributes(attribute.Key(tracing.TagRequestFullUrl).String(reqURL.String()))
	tc.tracingSpan.
		SetAttributes(attribute.Key(tracing.TagRequestScheme).String(scheme))
	tc.tracingSpan.
		SetAttributes(attribute.Key(tracing.TagRequestWebsocket).Bool(webSocket))

	metrics.IncWholeRequest(reqID, req, scheme)
}

// RegisterStatusCode - Registers the response status code.
func (tc TelemetryContext) RegisterStatusCode(statusCode int) {
	tc.tracingSpan.
		SetAttributes(attribute.Key(tracing.TagResponseStatusCode).Int(statusCode))
	metrics.IncStatusCode(statusCode)
}

// RegisterWholeResponse - Registers the whole response.
func (tc TelemetryContext) RegisterWholeResponse(reqID string, req http.Request, statusCode int, contentLength int, requestStartTime time.Time, scheme string, cached bool, stale bool) {
	tc.RegisterCacheStaleOrHit(req.Host, stale)
	tc.RegisterStatusCode(statusCode)

	duration := time.Since(requestStartTime).Milliseconds()
	metrics.IncWholeResponse(reqID, req, statusCode, contentLength, duration, scheme, cached, stale)
}

// RegisterRequestCacheStatus - Registers extra metrics / traces about the cache status.
func (tc TelemetryContext) RegisterRequestCacheStatus(forceFresh bool, enableCachedResponse bool, cached string) {
	tc.tracingSpan.
		SetAttributes(attribute.Key(tracing.TagCacheForcedFresh).Bool(forceFresh))
	tc.tracingSpan.
		SetAttributes(attribute.Key(tracing.TagCacheCacheable).Bool(enableCachedResponse))
	tc.tracingSpan.
		SetAttributes(attribute.Key(tracing.TagCacheCached).String(cached))
	tc.tracingSpan.
		SetAttributes(attribute.Key(tracing.TagCacheStale).Bool(cached == "STALE")) // TODO: Magic value
}

// RegisterCacheStaleOrHit - Registers metrics / traces about the cache status.
func (tc TelemetryContext) RegisterCacheStaleOrHit(server string, stale bool) {
	if stale {
		metrics.IncCacheStale(server)
	} else {
		metrics.IncCacheHit(server)
	}

	tc.tracingSpan.
		SetAttributes(attribute.Key(tracing.TagCacheStale).Bool(stale))
}

// RegisterRequestUpstream - Registers metrics / traces about the proxy upstream.
func (tc TelemetryContext) RegisterRequestUpstream(proxyURL url.URL, enableCachedResponse bool, cached string) {
	tc.tracingSpan.
		SetAttributes(attribute.Key(tracing.TagProxyEndpoint).String(proxyURL.String()))
	tc.tracingSpan.
		SetAttributes(attribute.Key(tracing.TagCacheForcedFresh).Bool(false))
	tc.tracingSpan.
		SetAttributes(attribute.Key(tracing.TagCacheCacheable).Bool(enableCachedResponse))
	tc.tracingSpan.
		SetAttributes(attribute.Key(tracing.TagCacheCached).String(cached))
	tc.tracingSpan.
		SetAttributes(attribute.Key(tracing.TagCacheStale).Bool(false))
}

// RegisterLegitRequest - Registers debug details about a specific request whether matches internal configuration.
func (tc TelemetryContext) RegisterLegitRequest(hostMatch bool, legitPort bool, hostname string, listeningPort string, confHostname string, confPort string) {
	tc.tracingSpan.
		SetAttributes(attribute.Key(tracing.TagRequestIsLegitHostnameMatches).Bool(hostMatch))
	tc.tracingSpan.
		SetAttributes(attribute.Key(tracing.TagRequestIsLegitPortMatches).Bool(legitPort))
	tc.tracingSpan.
		SetAttributes(attribute.Key(tracing.TagRequestIsLegitRequestHostname).String(hostname))
	tc.tracingSpan.
		SetAttributes(attribute.Key(tracing.TagRequestIsLegitRequestPort).String(listeningPort))
	tc.tracingSpan.
		SetAttributes(attribute.Key(tracing.TagRequestIsLegitConfHostname).String(confHostname))
	tc.tracingSpan.
		SetAttributes(attribute.Key(tracing.TagRequestIsLegitConfPort).String(confPort))
}

// RegisterPurge - Registers metrics / traces about the HTTP Purge event.
func (tc TelemetryContext) RegisterPurge(status bool, err error) {
	tc.tracingSpan.
		SetAttributes(attribute.Key(tracing.TagPurgeStatus).Bool(status))

	if err != nil {
		tracing.AddErrorToSpan(tc.tracingSpan, err)
		tracing.Fail(tc.tracingSpan, "internal error")

		// TODO: Add tracing.Fail -> prometheus as failures
	}
}

// RegisterServeOriginal - Registers debug details about a specific request whether should have ETag from original request.
func (tc TelemetryContext) RegisterServeOriginal(hash hash.Hash, header http.Header, statusCode int, lenContent int) {
	tc.tracingSpan.
		SetAttributes(attribute.Key(tracing.TagResponseMustServeOriginalResponseNoHashComputed).Bool(hash == nil))
	tc.tracingSpan.
		SetAttributes(attribute.Key(tracing.TagResponseMustServeOriginalResponseEtagPresent).String(header.Get(headers.ETag)))
	tc.tracingSpan.
		SetAttributes(attribute.Key(tracing.TagResponseMustServeOriginalResponseEtagAlreadyPresent).Bool(header.Get(headers.ETag) != ""))
	tc.tracingSpan.
		SetAttributes(attribute.Key(tracing.TagResponseMustServeOriginalResponseResponseStatusCode).Int(statusCode))
	tc.tracingSpan.
		SetAttributes(attribute.Key(tracing.TagResponseMustServeOriginalResponseResponseNot2xx).Bool((statusCode < http.StatusOK || statusCode >= http.StatusMultipleChoices)))
	tc.tracingSpan.
		SetAttributes(attribute.Key(tracing.TagResponseMustServeOriginalResponseResponse204).Bool(statusCode == http.StatusNoContent))
	tc.tracingSpan.
		SetAttributes(attribute.Key(tracing.TagResponseMustServeOriginalResponseNoBufferedContent).Bool(lenContent == 0))
}

// RegisterHostHealth - Registers metrics about the handled domains' health status.
func RegisterHostHealth(healthy int, unhealthy int) {
	metrics.SetHostHealthy(float64(healthy))
	metrics.SetHostUnhealthy(float64(unhealthy))
}
