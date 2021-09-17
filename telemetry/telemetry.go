package telemetry

import (
	"context"
	"hash"
	"net/http"
	"net/url"

	"github.com/fabiocicerchia/go-proxy-cache/telemetry/metrics"
	"github.com/fabiocicerchia/go-proxy-cache/telemetry/tracing"
	"github.com/go-http-utils/headers"
	"github.com/opentracing/opentracing-go"
)

type TelemetryContext struct {
	ctx         context.Context
	tracingSpan opentracing.Span
}

func From(ctx context.Context) TelemetryContext {
	return TelemetryContext{
		ctx:         ctx,
		tracingSpan: tracing.SpanFromContext(ctx),
	}
}

func (tc TelemetryContext) RegisterRedirect(targetURL url.URL) {
	tc.tracingSpan.
		SetTag(tracing.TagResponseLocation, targetURL.String())
}

func (tc TelemetryContext) RegisterEvent(name string) {
	tracing.AddEventsToSpan(tc.tracingSpan, name, map[string]string{})
}

func (tc TelemetryContext) RegisterRequest(req http.Request) {
	metrics.IncRequestHost(req.Host)

	tc.tracingSpan.
		SetTag(tracing.TagRequestHost, req.Host).
		SetTag(tracing.TagRequestUrl, req.URL.String()).
		SetTag(tracing.TagRequestMethod, req.Method)

	metrics.IncHttpMethod(req.Method)
}

func (tc TelemetryContext) RegisterRequestCall(reqID string, reqURL url.URL, scheme string, webSocket bool) {
	tc.tracingSpan.SetBaggageItem(tracing.BaggageRequestID, reqID)

	tc.tracingSpan.
		SetTag(tracing.TagRequestId, reqID).
		SetTag(tracing.TagRequestFullUrl, reqURL.String()).
		SetTag(tracing.TagRequestScheme, scheme).
		SetTag(tracing.TagRequestWebsocket, webSocket)

	metrics.IncUrlScheme(scheme)
}

func (tc TelemetryContext) RegisterStatusCode(statusCode int) {
	tc.tracingSpan.
		SetTag(tracing.TagResponseStatusCode, statusCode)
	metrics.IncStatusCode(statusCode)
}

func (tc TelemetryContext) RegisterRequestCacheStatus(forceFresh bool, enableCachedResponse bool, cached string) {
	tc.tracingSpan.
		SetTag(tracing.TagCacheForcedFresh, forceFresh).
		SetTag(tracing.TagCacheCacheable, enableCachedResponse).
		SetTag(tracing.TagCacheCached, cached).
		SetTag(tracing.TagCacheStale, cached == "STALE") // TODO: Magic value
}

func (tc TelemetryContext) RegisterCacheStaleOrHit(stale bool) {
	if stale {
		metrics.IncCacheStale()
	} else {
		metrics.IncCacheHit()
	}

	tc.tracingSpan.
		SetTag(tracing.TagCacheStale, stale)
}

func (tc TelemetryContext) RegisterRequestUpstream(proxyURL url.URL, enableCachedResponse bool, cached string) {
	tc.tracingSpan.
		SetTag(tracing.TagProxyEndpoint, proxyURL.String()).
		SetTag(tracing.TagCacheForcedFresh, false).
		SetTag(tracing.TagCacheCacheable, enableCachedResponse).
		SetTag(tracing.TagCacheCached, cached).
		SetTag(tracing.TagCacheStale, false)
}

func (tc TelemetryContext) RegisterLegitRequest(hostMatch bool, legitPort bool, hostname string, listeningPort string, confHostname string, confPort interface{}) {
	tc.tracingSpan.
		SetTag(tracing.TagRequestIsLegitHostnameMatches, hostMatch).
		SetTag(tracing.TagRequestIsLegitPortMatches, legitPort).
		SetTag(tracing.TagRequestIsLegitRequestHostname, hostname).
		SetTag(tracing.TagRequestIsLegitRequestPort, listeningPort).
		SetTag(tracing.TagRequestIsLegitConfHostname, confHostname).
		SetTag(tracing.TagRequestIsLegitConfPort, confPort)
}

func (tc TelemetryContext) RegisterPurge(status bool, err error) {
	tc.tracingSpan.
		SetTag(tracing.TagPurgeStatus, status)

	if err != nil {
		tracing.AddErrorToSpan(tc.tracingSpan, err)
		tracing.Fail(tc.tracingSpan, "internal error")

		// TODO: Add tracing.Fail -> prometheus as failures
	}
}

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

func RegisterHostHealth(healthy int, unhealthy int) {
	metrics.SetHostHealthy(float64(healthy))
	metrics.SetHostUnhealthy(float64(unhealthy))
}
