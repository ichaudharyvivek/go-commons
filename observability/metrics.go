package observability

import (
	"context"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/runtime"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"

	"github.com/ichaudharyvivek/go-commons/errorsx"
)

// initMeter builds the OTel meter provider, installs it as the global meter.
func initMeter(ctx context.Context, cfg Config, res *resource.Resource) (func(context.Context) error, error) {
	exporter, err := otlpmetrichttp.New(ctx, otlpMetricExporterOpts(cfg.OTLP)...)
	if err != nil {
		return nil, errorsx.Wrap(err, "failed to create metrics exporter")
	}

	mp := metric.NewMeterProvider(
		metric.WithResource(res),
		metric.WithReader(metric.NewPeriodicReader(exporter)),
	)
	otel.SetMeterProvider(mp)

	// Official runtime instrumentation uses the runtime/metrics API (no
	// stop-the-world ReadMemStats) and covers GC, goroutines, memory, and
	// scheduler stats.
	if startErr := runtime.Start(
		runtime.WithMeterProvider(mp),
		runtime.WithMinimumReadMemStatsInterval(time.Second),
	); startErr != nil {
		Log(ctx).Warn().Err(startErr).Msg("failed to start runtime metrics")
	}

	return mp.Shutdown, nil
}

func otlpMetricExporterOpts(cfg OTLPConfig) []otlpmetrichttp.Option {
	return []otlpmetrichttp.Option{
		otlpmetrichttp.WithEndpointURL(cfg.Endpoint),
	}
}
