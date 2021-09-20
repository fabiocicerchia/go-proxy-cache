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
	"io"
	"os"

	opentracing "github.com/opentracing/opentracing-go"
	jaeger "github.com/uber/jaeger-client-go"
	jaegerConfig "github.com/uber/jaeger-client-go/config"
)

// TODO: Move from OpenTracing to OpenTelemetry

// NewJaegerProvider returns a new instance of Jaeger.
func NewJaegerProvider(appVersion string, jaegerEndpoint string, enabled bool, sampleRatio float64) (opentracing.Tracer, io.Closer, error) {
	cfg := jaegerConfig.Configuration{
		ServiceName: "go-proxy-cache",
		Disabled:    !enabled,
		Tags: []opentracing.Tag{
			{Key: "service.version", Value: appVersion},
			{Key: "service.env", Value: os.Getenv("TRACING_ENV")},
		},
		Sampler: &jaegerConfig.SamplerConfig{
			Type:  "probabilistic",
			Param: sampleRatio,
		},
		Reporter: &jaegerConfig.ReporterConfig{
			LocalAgentHostPort: jaegerEndpoint,
		},
	}

	tracer, closer, err := cfg.NewTracer(jaegerConfig.Logger(jaeger.StdLogger))
	return tracer, closer, err
}
