package telemetry

import (
	"context"
	"hash"
	"net/http"
	"net/url"

	"github.com/fabiocicerchia/go-proxy-cache/telemetry/metrics"
	"github.com/fabiocicerchia/go-proxy-cache/telemetry/tracing"
	"github.com/go-http-utils/headers"
)

func RegisterRedirect(ctx context.Context, targetURL url.URL, statusCode int) {
	tracing.SpanFromContext(ctx).
		SetTag(tracing.TagResponseLocation, targetURL.String()).
		SetTag(tracing.TagResponseStatusCode, statusCode)
	metrics.IncStatusCode(statusCode)
}

func RegisterHostHealth(healthy int, unhealthy int) {
	metrics.SetHostHealthy(float64(healthy))
	metrics.SetHostUnhealthy(float64(unhealthy))
}

func RegisterEvent(ctx context.Context, name string) {
	tracing.AddEventsToSpan(tracing.SpanFromContext(ctx), name, map[string]string{})
}

func RegisterRequest(ctx context.Context, req http.Request) {
	metrics.IncRequestHost(req.Host)

	tracing.SpanFromContext(ctx).
		SetTag(tracing.TagRequestHost, req.Host).
		SetTag(tracing.TagRequestUrl, req.URL.String()).
		SetTag(tracing.TagRequestMethod, req.Method)

	metrics.IncHttpMethod(req.Method)
}

func RegisterRequestCall(ctx context.Context, reqID string, reqURL url.URL, scheme string, webSocket bool) {
	tracingSpan := tracing.SpanFromContext(ctx)

	tracingSpan.SetBaggageItem(tracing.BaggageRequestID, reqID)

	tracingSpan.
		SetTag(tracing.TagRequestId, reqID).
		SetTag(tracing.TagRequestFullUrl, reqURL.String()).
		SetTag(tracing.TagRequestScheme, scheme).
		SetTag(tracing.TagRequestWebsocket, webSocket)

	metrics.IncUrlScheme(scheme)
}

func RegisterStatusCode(ctx context.Context, statusCode int) {
	tracing.SpanFromContext(ctx).
		SetTag(tracing.TagResponseStatusCode, statusCode)
	metrics.IncStatusCode(statusCode)
}

func RegisterRequestCacheStatus(ctx context.Context, forceFresh bool, enableCachedResponse bool, cached string) {
	tracing.SpanFromContext(ctx).
		SetTag(tracing.TagCacheForcedFresh, forceFresh).
		SetTag(tracing.TagCacheCacheable, enableCachedResponse).
		SetTag(tracing.TagCacheCached, cached).
		SetTag(tracing.TagCacheStale, cached == "STALE") // TODO: Magic value
}

func RegisterCacheStaleOrHit(ctx context.Context, stale bool, statusCode int) {
	if stale {
		metrics.IncCacheStale()
	} else {
		metrics.IncCacheHit()
	}

	tracing.SpanFromContext(ctx).
		SetTag(tracing.TagCacheStale, stale).
		SetTag(tracing.TagResponseStatusCode, statusCode)
	metrics.IncStatusCode(statusCode)
}

func RegisterRequestUpstream(ctx context.Context, proxyURL url.URL, enableCachedResponse bool, cached string) {
	tracing.SpanFromContext(ctx).
		SetTag(tracing.TagProxyEndpoint, proxyURL.String()).
		SetTag(tracing.TagCacheForcedFresh, false).
		SetTag(tracing.TagCacheCacheable, enableCachedResponse).
		SetTag(tracing.TagCacheCached, cached).
		SetTag(tracing.TagCacheStale, false)
}

func RegisterLegitRequest(ctx context.Context, hostMatch bool, legitPort bool, hostname string, listeningPort string, confHostname string, confPort interface{}) {
	tracing.SpanFromContext(ctx).
		SetTag(tracing.TagRequestIsLegitHostnameMatches, hostMatch).
		SetTag(tracing.TagRequestIsLegitPortMatches, legitPort).
		SetTag(tracing.TagRequestIsLegitRequestHostname, hostname).
		SetTag(tracing.TagRequestIsLegitRequestPort, listeningPort).
		SetTag(tracing.TagRequestIsLegitConfHostname, confHostname).
		SetTag(tracing.TagRequestIsLegitConfPort, confPort)
}

func RegisterPurge(ctx context.Context, status bool, statusCode int, err error) {
	tracingSpan := tracing.SpanFromContext(ctx)

	tracingSpan.
		SetTag(tracing.TagPurgeStatus, status).
		SetTag(tracing.TagResponseStatusCode, statusCode)
	metrics.IncStatusCode(statusCode)

	if err != nil {
		tracing.AddErrorToSpan(tracingSpan, err)
		tracing.Fail(tracingSpan, "internal error")

		// TODO: Add tracing.Fail -> prometheus as failures
	}
}

func RegisterServeOriginal(ctx context.Context, hash hash.Hash, header http.Header, statusCode int, lenContent int) {
	tracing.SpanFromContext(ctx).
		SetTag(tracing.TagResponseMustServeOriginalResponseNoHashComputed, hash == nil).
		SetTag(tracing.TagResponseMustServeOriginalResponseEtagPresent, header.Get(headers.ETag)).
		SetTag(tracing.TagResponseMustServeOriginalResponseEtagAlreadyPresent, header.Get(headers.ETag) != "").
		SetTag(tracing.TagResponseMustServeOriginalResponseResponseStatusCode, statusCode).
		SetTag(tracing.TagResponseMustServeOriginalResponseResponseNot2xx, (statusCode < http.StatusOK || statusCode >= http.StatusMultipleChoices)).
		SetTag(tracing.TagResponseMustServeOriginalResponseResponse204, statusCode == http.StatusNoContent).
		SetTag(tracing.TagResponseMustServeOriginalResponseNoBufferedContent, lenContent == 0)
}
