package tracing

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
	"net/http"

	opentracing "github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	"github.com/opentracing/opentracing-go/log"
)

// BaggageRequestID - Key for OpenTracing baggage Request ID.
const BaggageRequestID = "request.id"

// TagErrorError - Error
const TagErrorError = "error.error"

// TagProxyEndpoint - Upstream URL
const TagProxyEndpoint = "proxy.endpoint"

// TagPurgeStatus - Status of PURGE on resource
const TagPurgeStatus = "purge.status"

// TagStorageCached -
const TagStorageCached = "storage.cached"

// TagCacheCacheable - Is Resource Cacheable?
const TagCacheCacheable = "cache.cacheable"

// TagCacheCached - Was the Response Cached?
const TagCacheCached = "cache.cached"

// TagCacheForcedFresh - Bypass Cached Content (requested by user)
const TagCacheForcedFresh = "cache.forced_fresh"

// TagCacheStale - Was the Cached Response Stale?
const TagCacheStale = "cache.stale"

// TagRequestFullUrl - Full Request URL
const TagRequestFullUrl = "request.full_url"

// TagRequestHost - Request's Host
const TagRequestHost = "request.host"

// TagRequestId - Request Unique ID
const TagRequestId = "request.id"

// TagRequestIsLegitConfHostname - Configuration Host value
const TagRequestIsLegitConfHostname = "request.is_legit.conf_hostname"

// TagRequestIsLegitConfPort - Configuration port value
const TagRequestIsLegitConfPort = "request.is_legit.conf_port"

// TagRequestIsLegitHostnameMatches - Request is legit as request's host matches the configuration
const TagRequestIsLegitHostnameMatches = "request.is_legit.hostname_matches"

// TagRequestIsLegitPortMatches - Request is legit as request's port matches the configuration
const TagRequestIsLegitPortMatches = "request.is_legit.port_matches"

// TagRequestIsLegitRequestHostname - Request's Host value
const TagRequestIsLegitRequestHostname = "request.is_legit.req_hostname"

// TagRequestIsLegitRequestPort - Request's port value
const TagRequestIsLegitRequestPort = "request.is_legit.req_port"

// TagRequestMethod - Request's HTTP Request Method
const TagRequestMethod = "request.method"

// TagRequestScheme - Request's URL Scheme
const TagRequestScheme = "request.scheme"

// TagRequestUrl - Request's URL
const TagRequestUrl = "request.url"

// TagRequestWebsocket - Is Request using a WebSocket?
const TagRequestWebsocket = "request.websocket"

// TagResponseLocation - HTTP 301 HTTP Location
const TagResponseLocation = "response.location"

// TagResponseMustServeOriginalResponseEtagAlreadyPresent - Should serve original response if an ETag is already set in upstream
const TagResponseMustServeOriginalResponseEtagAlreadyPresent = "response.must_serve_original_response.etag_already_present"

// TagResponseMustServeOriginalResponseEtagPresent - Upstream ETag HTTP Header value
const TagResponseMustServeOriginalResponseEtagPresent = "response.must_serve_original_response.etag_present"

// TagResponseMustServeOriginalResponseNoBufferedContent - Should serve original response if there is no content
const TagResponseMustServeOriginalResponseNoBufferedContent = "response.must_serve_original_response.no_buffered_content"

// TagResponseMustServeOriginalResponseNoHashComputed - Should serve original response if no ETag has been generated
const TagResponseMustServeOriginalResponseNoHashComputed = "response.must_serve_original_response.no_hash_computed"

// TagResponseMustServeOriginalResponseResponse204 - Should serve original response if response is 204 no content
const TagResponseMustServeOriginalResponseResponse204 = "response.must_serve_original_response.response_204"

// TagResponseMustServeOriginalResponseResponseNot2xx - Should serve original response if response is not successful
const TagResponseMustServeOriginalResponseResponseNot2xx = "response.must_serve_original_response.response_not_2xx"

// TagResponseMustServeOriginalResponseResponseStatusCode - Upstream HTTP Status Code
const TagResponseMustServeOriginalResponseResponseStatusCode = "response.must_serve_original_response.response_status_code"

// TagResponseStatusCode - Response Status Code
const TagResponseStatusCode = "response.status_code"

// StartSpanFromRequest retrieves a tracing span from the inbound HTTP request.
// The tracing can be continued with a child span (if any) so there will be continuity in the tracing.
func StartSpanFromRequest(operation string, r *http.Request) opentracing.Span {
	spanCtx, _ := Extract(r)
	return opentracing.GlobalTracer().StartSpan(operation, ext.RPCServerOption(spanCtx))
}

// NewSpan returns a new tracing span from the global tracer.
// Each tracing span must be followed by `defer tracingSpan.Finish()`.
func NewSpan(operation string) opentracing.Span {
	return opentracing.GlobalTracer().StartSpan(operation)
}

// NewChildSpan returns a new tracing child span from the global tracer.
// Each tracing span must be followed by `defer tracingSpan.Finish()`.
func NewChildSpan(ctx context.Context, operation string) opentracing.Span {
	spanCtx := SpanFromContext(ctx).Context()
	return opentracing.GlobalTracer().StartSpan(operation, opentracing.ChildOf(spanCtx))
}

// Inject into the outbound HTTP Request the tracing span's context.
func Inject(span opentracing.Span, request *http.Request) error {
	return span.Tracer().Inject(
		span.Context(),
		opentracing.HTTPHeaders,
		opentracing.HTTPHeadersCarrier(request.Header))
}

// Extract from the inbound HTTP Request the parent tracing span.
func Extract(r *http.Request) (opentracing.SpanContext, error) {
	return opentracing.GlobalTracer().Extract(
		opentracing.HTTPHeaders,
		opentracing.HTTPHeadersCarrier(r.Header))
}

// AddEventsToSpan adds new events to the tracing span.
func AddEventsToSpan(tracingSpan opentracing.Span, name string, events map[string]string) {
	tracingSpan.LogFields(log.String("event", name))

	for k, v := range events {
		tracingSpan.LogFields(log.String(k, v))
	}
}

// AddErrorToSpan adds a new error event to the tracing span.
func AddErrorToSpan(tracingSpan opentracing.Span, err error) {
	ext.LogError(tracingSpan, err)
}

// Fail flags the tracing span as failed, and adds an error label.
func Fail(tracingSpan opentracing.Span, msg string) {
	tracingSpan.SetTag(TagErrorError, msg)
}

// SpanFromContext returns the current tracing span from a context.
func SpanFromContext(ctx context.Context) opentracing.Span {
	return opentracing.SpanFromContext(ctx)
}
