package observability

import (
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// DefaultRegistry is the Prometheus registry used by all Astra services.
// Using a custom registry avoids accidental registration collisions with
// third-party libraries that may also import the default global registry.
var DefaultRegistry = prometheus.NewRegistry()

var (
	httpRequestsTotal = promauto.With(DefaultRegistry).NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "astra",
			Name:      "http_requests_total",
			Help:      "Total HTTP requests by service, method, path and status.",
		},
		[]string{"service", "method", "path", "status"},
	)

	httpRequestDuration = promauto.With(DefaultRegistry).NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "astra",
			Name:      "http_request_duration_seconds",
			Help:      "HTTP request latency distribution by service, method and path.",
			Buckets:   prometheus.DefBuckets,
		},
		[]string{"service", "method", "path"},
	)

	natsMessagesConsumed = promauto.With(DefaultRegistry).NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "astra",
			Name:      "nats_messages_consumed_total",
			Help:      "Total NATS messages consumed by service, stream and subject.",
		},
		[]string{"service", "stream", "subject", "result"},
	)
)

// MetricsHandler returns an http.Handler that exposes DefaultRegistry on
// /metrics. Services using Fiber should adapt this with adaptor.HTTPHandler.
func MetricsHandler() http.Handler {
	return promhttp.HandlerFor(DefaultRegistry, promhttp.HandlerOpts{
		Registry: DefaultRegistry,
	})
}

// RecordHTTPRequest increments request counters and records latency.
func RecordHTTPRequest(service, method, path, status string, duration time.Duration) {
	httpRequestsTotal.WithLabelValues(service, method, path, status).Inc()
	httpRequestDuration.WithLabelValues(service, method, path).Observe(duration.Seconds())
}

// RecordHTTPRequestFromFiber is a convenience helper for Fiber handlers that
// already know the status code as an integer. It delegates to RecordHTTPRequest
// after stringifying the status.
func RecordHTTPRequestFromFiber(service, method, path string, status int, duration time.Duration) {
	RecordHTTPRequest(service, method, path, strconv.Itoa(status), duration)
}

// RecordNATSMessage increments the consumed message counter. result should be
// "ok", "error", or "nack".
func RecordNATSMessage(service, stream, subject, result string) {
	natsMessagesConsumed.WithLabelValues(service, stream, subject, result).Inc()
}

// MustRegister registers the supplied Prometheus collectors with
// DefaultRegistry and panics on duplicate registration. Use it for service-level
// custom metrics.
func MustRegister(cs ...prometheus.Collector) {
	DefaultRegistry.MustRegister(cs...)
}
