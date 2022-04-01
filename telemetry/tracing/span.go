package tracing

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
	"net/http"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/propagation"
)

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
func StartSpanFromRequest(operation string, r *http.Request) (trace.Span, context.Context) {
	return NewSpan(r.Context(), operation)
}

// NewSpan returns a new tracing span from the global tracer.
// Each tracing span must be followed by `defer tracingSpan.End()`.
func NewSpan(ctx context.Context, operation string) (trace.Span, context.Context) {
	ctx, span := otel.GetTracerProvider().Tracer("").Start(ctx, operation)
	return span, ctx
}

// NewChildSpan returns a new tracing child span from the global tracer.
// Each tracing span must be followed by `defer tracingSpan.End()`.
func NewChildSpan(ctx context.Context, operation string) trace.Span {
	ctx, span := otel.GetTracerProvider().Tracer("").Start(ctx, operation)
	return span
}

// Inject into the outbound HTTP Request the tracing span's context.
func Inject(ctx context.Context, request *http.Request) {
	otel.GetTextMapPropagator().Inject(ctx, propagation.HeaderCarrier(request.Header))
}

// AddEventsToSpan adds new events to the tracing span.
func AddEventsToSpan(tracingSpan trace.Span, name string, events map[string]string) {
	attributes := []attribute.KeyValue{}
	for k, v := range events {
		attributes = append(attributes, attribute.Key(k).String(v))
	}

	tracingSpan.AddEvent(name, trace.WithAttributes(attributes...))
}

// AddErrorToSpan adds a new error event to the tracing span.
func AddErrorToSpan(tracingSpan trace.Span, err error) {
	tracingSpan.RecordError(err)
}

// Fail flags the tracing span as failed, and adds an error label.
func Fail(tracingSpan trace.Span, msg string) {
	tracingSpan.SetStatus(codes.Error, msg)
}

// SpanFromContext returns the current tracing span from a context.
func SpanFromContext(ctx context.Context) trace.Span {
	return trace.SpanFromContext(ctx)
}

// AddBoolTag - Add a boolean tag to a tracing span.
func AddBoolTag(span trace.Span, key string, value bool) {
	span.SetAttributes(attribute.Key(key).Bool(value))
}

// SetErrorAndFail - Add an error and a custom failure message to a tracing span.
func SetErrorAndFail(span trace.Span, err error, msg string) {
	AddErrorToSpan(span, err)
	Fail(span, msg)
}
