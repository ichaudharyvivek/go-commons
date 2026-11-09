package observability

import (
	"context"
	"fmt"
	"os"
	"slices"
	"sync"

	"github.com/rs/zerolog"
	"go.opentelemetry.io/otel/trace"
)

var once sync.Once
var (
	baseLogger    = zerolog.Nop()
	consoleWriter = []string{"development", "local", "dev"}
)

// initLogger gives the singleton instance of a logger.
func initLogger(cfg Config) {
	once.Do(func() {
		baseLogger = createLogger(cfg)
	})
}

// createLogger builds a zerolog.Logger form config.
func createLogger(cfg Config) zerolog.Logger {
	configureErrorMarshaling()

	level, err := zerolog.ParseLevel(cfg.Logging.Level)
	if err != nil || cfg.Logging.Level == "" {
		level = zerolog.InfoLevel
	}

	l := zerolog.New(os.Stdout).Level(level).With().
		Timestamp().
		Str("env", cfg.OTLP.Environment).
		Str("service", cfg.OTLP.ServiceName).
		Str("version", cfg.OTLP.ServiceVersion).
		Logger()

	if slices.Contains(consoleWriter, cfg.OTLP.Environment) {
		l = l.Output(zerolog.ConsoleWriter{Out: os.Stdout})
	}

	return l
}

// configureErrorMarshaling wires zerolog's .Err() call to automatically extract and log internal stack traces from errorsx object
func configureErrorMarshaling() {
	//nolint:reassign
	zerolog.ErrorMarshalFunc = func(err error) any {
		if err == nil {
			return nil
		}
		return map[string]any{
			"message":    err.Error(),
			"stacktrace": fmt.Sprintf("%+v", err),
		}
	}
}

// Log returns a logger with the current context's trace and span IDs.
// If no active span is found, it safely falls back to the base logger.
func Log(ctx context.Context) *zerolog.Logger {
	sc := trace.SpanContextFromContext(ctx)
	if !sc.IsValid() {
		return &baseLogger
	}

	l := baseLogger.With().
		Str("trace_id", sc.TraceID().String()).
		Str("span_id", sc.SpanID().String()).
		Logger()

	return &l
}
