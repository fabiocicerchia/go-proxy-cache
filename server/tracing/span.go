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

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// NewSpan returns a new tracing span from the global tracer.
// Each tracing span must be followed by `defer tracingSpan.End()`.
func NewSpan(ctx context.Context, name string) (context.Context, trace.Span) {
	return otel.Tracer("go-proxy-cache").Start(ctx, name)
}

// AddEventsToSpan adds new events to the tracing span.
func AddEventsToSpan(tracingSpan trace.Span, name string, events map[string]string) {
	list := []trace.EventOption{}

	for k, v := range events {
		list = append(list, trace.WithAttributes(attribute.Key(k).String(v)))
	}

	tracingSpan.AddEvent(name, list...)
}

// AddTagsToSpan adds new tags to the tracing span.
func AddTagsToSpan(tracingSpan trace.Span, tags map[string]string) {
	list := []attribute.KeyValue{}

	for k, v := range tags {
		list = append(list, attribute.Key(k).String(v))
	}

	tracingSpan.SetAttributes(list...)
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
