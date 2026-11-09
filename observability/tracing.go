package observability

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"

	"go-commons/errorsx"
)

// Span limits. Bounds on per-span cardinality to protect the pipeline from a
// runaway service flooding the backend with attributes/events. These are a
// pipeline-safety default, not something services should tune per deployment.
const (
	linkCountLimit              = 32
	attributePerLinkCountLimit  = 32
	attributePerEventCountLimit = 32
	eventCountLimit             = 128
	attributeCountLimit         = 128
	attributeValueLengthLimit   = 256
)

// initTracer builds the OTel tracer provider.
func initTracer(ctx context.Context, cfg Config, res *resource.Resource) (func(context.Context) error, error) {
	exporter, err := otlptracehttp.New(ctx, otlpExporterOpts(cfg.OTLP)...)
	if err != nil {
		return nil, errorsx.Wrap(err, "observability: failed to create trace exporter")
	}

	ratio := 1.0
	if cfg.Tracing.SampleRatio != nil {
		ratio = *cfg.Tracing.SampleRatio
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithResource(res),
		sdktrace.WithBatcher(exporter),
		sdktrace.WithSampler(traceSampler(ratio)),
		sdktrace.WithRawSpanLimits(sdktrace.SpanLimits{
			LinkCountLimit:              linkCountLimit,
			EventCountLimit:             eventCountLimit,
			AttributeCountLimit:         attributeCountLimit,
			AttributeValueLengthLimit:   attributeValueLengthLimit,
			AttributePerLinkCountLimit:  attributePerLinkCountLimit,
			AttributePerEventCountLimit: attributePerEventCountLimit,
		}),
	)
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	return tp.Shutdown, nil
}

func traceSampler(ratio float64) sdktrace.Sampler {
	switch {
	case ratio >= 1.0:
		return sdktrace.AlwaysSample()
	case ratio <= 0.0:
		return sdktrace.NeverSample()
	default:
		return sdktrace.ParentBased(sdktrace.TraceIDRatioBased(ratio))
	}
}

func otlpExporterOpts(cfg OTLPConfig) []otlptracehttp.Option {
	return []otlptracehttp.Option{
		otlptracehttp.WithEndpointURL(cfg.Endpoint),
	}
}
