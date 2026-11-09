// Package observability wires up tracing, metrics, logging, and profiling for microservices.
//
// See README.md for the full usage guide.
package observability

import (
	"context"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.34.0"

	"rest-w-filter/pkg/errorsx"
)

// Init validates cfg, wires up logging and optionally, tracing and metrics.
// It returns a shutdown func that tears down whatever was started, in
// reverse order, aggregating any errors.
func Init(ctx context.Context, cfg Config) (func(context.Context) error, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	initLogger(cfg)

	res, err := createResource(ctx, cfg.OTLP)
	if err != nil {
		// Even if error is returned, resource is created, log the error and move on.
		// Observability shouldn't stop the program from running.
		Log(ctx).Warn().Err(err).Msg("resource creation completed with warnings")
	}

	var shutdownFuncs []func(context.Context) error

	if cfg.isTracingEnabled() {
		tracerShutdown, err := initTracer(ctx, cfg, res)
		if err != nil {
			return nil, errorsx.Wrap(err, "failed to initialize tracing")
		}
		shutdownFuncs = append(shutdownFuncs, tracerShutdown)
	}

	if cfg.isMetricsEnabled() {
		meterShutdown, err := initMeter(ctx, cfg, res)
		if err != nil {
			return nil, errorsx.Wrap(err, "failed to initialize metrics")
		}
		shutdownFuncs = append(shutdownFuncs, meterShutdown)
	}

	Log(ctx).Info().
		Bool("tracing", cfg.isTracingEnabled()).
		Bool("metrics", cfg.isMetricsEnabled()).
		Bool("profiling", cfg.Profiling.Enabled).
		Msg("observability initialized")

	return combinedShutdown(shutdownFuncs), nil
}

// combinedShutdown tears down every initialized component in reverse order,
// aggregating any errors via errorsx.Join.
func combinedShutdown(funcs []func(context.Context) error) func(context.Context) error {
	return func(ctx context.Context) error {
		var errs []error
		for i := len(funcs) - 1; i >= 0; i-- {
			if err := funcs[i](ctx); err != nil {
				errs = append(errs, err)
			}
		}
		if len(errs) > 0 {
			return errorsx.Join(errs...)
		}
		return nil
	}
}

func createResource(ctx context.Context, otlp OTLPConfig) (*resource.Resource, error) {
	return resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName(otlp.ServiceName),
			semconv.ServiceVersion(otlp.ServiceVersion),
			semconv.DeploymentEnvironmentName(otlp.Environment),
			semconv.ServiceInstanceID(generateInstanceID()),
		),
		resource.WithProcessRuntimeName(),
		resource.WithProcessRuntimeVersion(),
		resource.WithProcessRuntimeDescription(),
		resource.WithHost(),
		resource.WithContainer(),
		resource.WithOS(),
		resource.WithFromEnv(), // honours OTEL_RESOURCE_ATTRIBUTES .env variable
		resource.WithTelemetrySDK(),
	)
}

func generateInstanceID() string {
	return uuid.New().String()
}
