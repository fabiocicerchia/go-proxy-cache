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
	"go.opentelemetry.io/otel/exporters/trace/jaeger"
	"go.opentelemetry.io/otel/propagation"
	sdkresource "go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/semconv"
	"go.opentelemetry.io/otel/trace"
)

// TODO: Make it customizable
const openTracingSampleRatio = 1.0

// Config is the configuration for OpenTelemetry (Jaeger).
type Config struct {
	JaegerEndpoint string
	ServiceName    string
	ServiceVersion string
	Environment    string
	Enabled        bool
}

// OpenTelemetryProvider represents the tracer provider.
type OpenTelemetryProvider struct {
	provider trace.TracerProvider
}

// Close shuts down the Jaeger provider.
func (otp OpenTelemetryProvider) Close(ctx context.Context) error {
	if prv, ok := otp.provider.(*sdktrace.TracerProvider); ok {
		return prv.Shutdown(ctx)
	}

	return nil
}

// Start the Jaeger provider.
func (otp OpenTelemetryProvider) Start() {
	otel.SetTracerProvider(otp.provider)
	propagator := propagation.NewCompositeTextMapPropagator(propagation.Baggage{}, propagation.TraceContext{})
	otel.SetTextMapPropagator(propagator)
}

// NewJaegerProvider returns a new instance of Jaeger.
func NewJaegerProvider(ctx context.Context, config Config) (OpenTelemetryProvider, error) {
	if !config.Enabled {
		return OpenTelemetryProvider{
			provider: trace.NewNoopTracerProvider(),
		}, nil
	}

	rawExp, err := jaeger.NewRawExporter(
		jaeger.WithCollectorEndpoint(jaeger.WithEndpoint(config.JaegerEndpoint)),
	)
	if err != nil {
		return OpenTelemetryProvider{}, err
	}

	resourceProcess, _ := sdkresource.New(context.Background(),
		sdkresource.WithProcess(),
	)
	resourceApp := sdkresource.NewWithAttributes(
		semconv.ServiceNameKey.String(config.ServiceName),
		semconv.ServiceVersionKey.String(config.ServiceVersion),
		semconv.DeploymentEnvironmentKey.String(config.Environment),
	)
	resource := sdkresource.Merge(resourceProcess, resourceApp)
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.TraceIDRatioBased(openTracingSampleRatio)),
		sdktrace.WithBatcher(rawExp),
		sdktrace.WithResource(resource),
	)

	return OpenTelemetryProvider{
		provider: tp,
	}, nil
}
