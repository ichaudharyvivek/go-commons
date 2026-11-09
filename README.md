# go-commons

Go packages shared across services: errors with stack traces, a JSON response envelope, a traced HTTP client, and OpenTelemetry and logging setup in a single `Init` call.

| Package                           | What it gives you                                                                                              |
| --------------------------------- | -------------------------------------------------------------------------------------------------------------- |
| [`errorsx`](#errorsx)             | Drop-in replacement for `errors`/`fmt.Errorf` that captures a stack trace at the point an error is created     |
| [`httpx`](#httpx)                 | Uniform JSON response envelope (`Builder`) and an `*http.Client` that propagates trace context                 |
| [`observability`](#observability) | `Init` for zerolog logging, OTLP tracing, OTLP metrics and Go runtime metrics, plus a basic-auth pprof handler |

## Install

```bash
go get github.com/ichaudharyvivek/go-commons@latest
```

```go
import (
	"github.com/ichaudharyvivek/go-commons/errorsx"
	"github.com/ichaudharyvivek/go-commons/httpx"
	"github.com/ichaudharyvivek/go-commons/observability"
)
```

`observability` imports `errorsx`; the other packages are independent of each other.

## errorsx

Wraps the standard library error API. `New`, `Newf`, `Wrap` and `Wrapf` record up to 32 stack frames (runtime frames are dropped), and the result still works with `errors.Is` and `errors.As`.

```go
func (r *Repo) GetByID(ctx context.Context, id string) (*User, error) {
	u, err := r.query(ctx, id)
	if err != nil {
		return nil, errorsx.Wrapf(err, "get user %s", id)
	}
	return u, nil
}
```

| Function                                        | Behaviour                                                                              |
| ----------------------------------------------- | -------------------------------------------------------------------------------------- |
| `New(msg)`, `Newf(format, args...)`             | New error with a stack trace                                                           |
| `Wrap(err, msg)`, `Wrapf(err, format, args...)` | Message becomes `msg: cause`; returns `nil` when `err` is `nil`                        |
| `Is`, `As`, `Unwrap`, `Join`                    | Thin forwards to the standard library, so one import is enough                         |
| `GetStack(err) (string, bool)`                  | Stack of the innermost `errorsx` error in the chain, i.e. where the failure originated |

Formatting:

- `%v` and `%s` print the message, `%q` prints it quoted.
- `%+v` prints the message followed by the origin stack trace.

When `Wrap` is given an error that does not come from `errorsx`, the trace starts at the `Wrap` call.

## httpx

### Response envelope

`httpx.Status(code)` starts a `Builder`; chain what you need and finish with `JSON(w)`.

```go
httpx.Status(http.StatusCreated).
	Message("User created successfully").
	Data(user).
	JSON(w)
```

```json
{
  "success": true,
  "message": "User created successfully",
  "data": { "id": "..." }
}
```

| Method                               | Effect                                                                              |
| ------------------------------------ | ----------------------------------------------------------------------------------- |
| `Message(string)`                    | Human-readable message                                                              |
| `Data(any)`                          | Payload                                                                             |
| `Error(string)` / `Errors([]string)` | Error text or a list (for example validation failures); forces `success` to `false` |
| `Meta(*Meta)`                        | Extra metadata, currently pagination                                                |
| `JSON(http.ResponseWriter)`          | Sets `Content-Type: application/json`, writes the status and the body               |

`success` is `true` for 2xx codes. Empty fields are omitted from the output. For lists, attach pagination:

```go
meta := &httpx.Meta{Pagination: &httpx.Pagination{
	Page: 1, PageSize: 20, Count: len(users), HasNext: hasNext,
}}
httpx.Status(http.StatusOK).Data(users).Meta(meta).JSON(w)
```

`httpx.Response` is exported so swaggo annotations can reference it, e.g. `@Success 200 {object} httpx.Response{data=UserResponse}`.

### Traced client

```go
client := httpx.NewTracedClient()
req, _ := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
resp, err := client.Do(req)
```

The transport is `otelhttp.NewTransport(http.DefaultTransport)`: it injects trace context into outbound requests and records client spans and metrics. Pass the incoming request context so the trace stays connected. The client has no `Timeout` set; set one yourself or use per-request contexts.

## observability

`observability.Init` validates the config, starts logging, then tracing and metrics if enabled. It returns one shutdown function.

```go
shutdown, err := observability.Init(ctx, observability.Config{
	OTLP: observability.OTLPConfig{
		Endpoint:       "http://otel-collector:4318",
		Environment:    "production",
		ServiceName:    "billing-api",
		ServiceVersion: version,
	},
	Logging: observability.LoggingConfig{Level: "info"},
	Tracing: observability.TracingConfig{Enabled: true},
	Metrics: observability.MetricsConfig{Enabled: true},
})
if err != nil {
	return err
}
defer func() {
	shutdownCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	defer cancel()
	_ = shutdown(shutdownCtx)
}()
```

### Config

| Field                                     | Notes                                                                                                                      |
| ----------------------------------------- | -------------------------------------------------------------------------------------------------------------------------- |
| `OTLP.ServiceName`                        | Required. `Init` fails without it                                                                                          |
| `OTLP.Endpoint`                           | Collector URL. Traces and metrics are exported over OTLP/HTTP                                                              |
| `OTLP.Environment`, `OTLP.ServiceVersion` | Resource attributes, also stamped on every log line                                                                        |
| `Logging.Level`                           | zerolog level name; empty or invalid falls back to `info`                                                                  |
| `Tracing.Enabled`                         | Takes effect only when `OTLP.Endpoint` is also set                                                                         |
| `Tracing.SampleRatio`                     | `*float64`. `nil` or `>= 1` samples everything, `<= 0` samples nothing, anything between uses a parent-based ratio sampler |
| `Metrics.Enabled`                         | Takes effect only when `OTLP.Endpoint` is also set                                                                         |
| `Profiling.Enabled`                       | Requires `BasicAuthUser` and `BasicAuthPass`, otherwise `Init` fails                                                       |

### What Init sets up

Logging is a process-wide zerolog logger that writes JSON to stdout with `env`, `service` and `version` fields. The `development`, `local` and `dev` environments get a console writer instead. The logger is created once, so later `Init` calls do not reconfigure it.

Tracing uses an OTLP/HTTP exporter with a batch processor and W3C `traceparent` and baggage propagation. Span limits are fixed (128 attributes and events per span, 32 links, 256 characters per attribute value) to keep a noisy service from flooding the backend.

Metrics use an OTLP/HTTP exporter with a periodic reader, plus the official Go runtime instrumentation for GC, goroutines, memory and the scheduler.

The OpenTelemetry resource carries the service name, version and environment, a random per-process instance ID, host, container, OS and Go runtime details, and anything in `OTEL_RESOURCE_ATTRIBUTES`. If building it fails partway, `Init` logs a warning and keeps going.

The shutdown function stops whatever was started, in reverse order, and joins any errors.

### Logging with trace correlation

`observability.Log(ctx)` returns the logger. When `ctx` carries a valid span, `trace_id` and `span_id` are added to every line. Before `Init` it is a no-op logger, so library code can call it safely.

```go
observability.Log(ctx).Error().Err(err).Str("order_id", id).Msg("charge failed")
```

`.Err()` is wired to `errorsx`. The `error` field becomes an object with `message` and `stacktrace` (the `%+v` output), so the origin stack shows up in your log backend without extra code.

### Profiling

`ProfilingHandler(cfg.Profiling)` returns the standard `net/http/pprof` endpoints behind HTTP basic auth (constant-time credential comparison). `Init` does not mount it, and pprof output can expose sensitive runtime details, so serve it on an internal-only port:

```go
if cfg.Profiling.Enabled {
	go func() {
		srv := &http.Server{
			Addr:              "127.0.0.1:6060",
			Handler:           observability.ProfilingHandler(cfg.Profiling),
			ReadHeaderTimeout: 5 * time.Second,
		}
		_ = srv.ListenAndServe()
	}()
}
```

## Dependencies

These are the versions the packages are used with in the go-boilerplate project: OpenTelemetry core `v1.44.0`, zerolog `v1.34.0`, and `go 1.26.5` in its `go.mod`.

| Used by         | Library                                                                                                                 |
| --------------- | ----------------------------------------------------------------------------------------------------------------------- |
| `observability` | `rs/zerolog`, `google/uuid`, OpenTelemetry SDK, OTLP/HTTP trace and metric exporters, `contrib/instrumentation/runtime` |
| `httpx`         | `contrib/instrumentation/net/http/otelhttp`                                                                             |
| `errorsx`       | standard library only                                                                                                   |

## Status

- No test files exist in these packages yet.
- No license file has been added to the repository.
