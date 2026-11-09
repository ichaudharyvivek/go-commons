package httpx

import (
	"net/http"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

// NewTracedClient returns an *http.Client whose transport propagates trace
// context on outbound calls and records client spans/metrics. Use it for all
// service-to-service HTTP so distributed traces stay connected.
func NewTracedClient() *http.Client {
	return &http.Client{
		Transport: otelhttp.NewTransport(http.DefaultTransport),
	}
}
