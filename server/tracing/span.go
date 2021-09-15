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

	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	"github.com/opentracing/opentracing-go/log"
)

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
	list := []log.Field{
		log.String("event", name),
	}

	for k, v := range events {
		list = append(list, log.String(k, v))
	}

	// tracingSpan.LogFields(list...)
}

// AddErrorToSpan adds a new error event to the tracing span.
func AddErrorToSpan(tracingSpan opentracing.Span, err error) {
	ext.LogError(tracingSpan, err)
}

// Fail flags the tracing span as failed, and adds an error label.
func Fail(tracingSpan opentracing.Span, msg string) {
	tracingSpan.SetTag("error.error", msg)
}

// SpanFromContext returns the current tracing span from a context.
func SpanFromContext(ctx context.Context) opentracing.Span {
	return opentracing.SpanFromContext(ctx)
}
