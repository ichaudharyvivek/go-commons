package observability

import (
	"strings"

	"github.com/ichaudharyvivek/go-commons/errorsx"
)

type Config struct {
	OTLP      OTLPConfig
	Logging   LoggingConfig
	Metrics   MetricsConfig
	Tracing   TracingConfig
	Profiling ProfilingConfig
}

type OTLPConfig struct {
	Endpoint       string
	Environment    string
	ServiceName    string
	ServiceVersion string
}

type LoggingConfig struct {
	Level string
}

type MetricsConfig struct {
	Enabled bool
}

type TracingConfig struct {
	Enabled     bool
	SampleRatio *float64
}

type ProfilingConfig struct {
	Enabled       bool
	BasicAuthUser string
	BasicAuthPass string
}

func (c *Config) Validate() error {
	if strings.TrimSpace(c.OTLP.ServiceName) == "" {
		return errorsx.New("servicename is required")
	}

	if c.Profiling.Enabled {
		noUser := strings.TrimSpace(c.Profiling.BasicAuthUser) == ""
		noPass := strings.TrimSpace(c.Profiling.BasicAuthPass) == ""
		if noUser || noPass {
			return errorsx.New("profiling needs BasicAuthUser and BasicAuthPass when enabled")
		}
	}

	return nil
}

func (c *Config) isTracingEnabled() bool {
	return c.Tracing.Enabled && strings.TrimSpace(c.OTLP.Endpoint) != ""
}

func (c *Config) isMetricsEnabled() bool {
	return c.Metrics.Enabled && strings.TrimSpace(c.OTLP.Endpoint) != ""
}
