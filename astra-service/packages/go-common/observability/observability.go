package observability

import (
	"context"
	"log/slog"
	"os"

	"github.com/prometheus/client_golang/prometheus"
)

// Config bundles all observability initialization options.
type Config struct {
	ServiceName    string
	ServiceVersion string
	Environment    string
	OTLPEndpoint   string
	SampleRatio    float64
	LogLevel       slog.Level
}

// Init bootstraps the package-default logger, OpenTelemetry tracer provider,
// and Prometheus registry for a service. It returns a shutdown function that
// should be deferred in main(). If tracer setup fails, the error is logged but
// the service is allowed to continue operating without distributed tracing.
func Init(ctx context.Context, cfg Config) (func(context.Context) error, error) {
	if cfg.ServiceName == "" {
		cfg.ServiceName = "astra-service"
	}
	if cfg.ServiceVersion == "" {
		cfg.ServiceVersion = "unknown"
	}
	if cfg.Environment == "" {
		cfg.Environment = os.Getenv("ASTRA_ENV")
		if cfg.Environment == "" {
			cfg.Environment = "development"
		}
	}

	SetLogger(NewLogger(cfg.LogLevel))

	shutdown := func(context.Context) error { return nil }
	if cfg.OTLPEndpoint != "" || os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT") != "" {
		tracerShutdown, err := InitTracer(ctx, TracerConfig{
			ServiceName:    cfg.ServiceName,
			ServiceVersion: cfg.ServiceVersion,
			Environment:    cfg.Environment,
			OTLPEndpoint:   cfg.OTLPEndpoint,
			SampleRatio:    cfg.SampleRatio,
		})
		if err != nil {
			Error(ctx, "observability tracer init failed; continuing without tracing", err)
		} else {
			shutdown = tracerShutdown
		}
	}

	return shutdown, nil
}

// DefaultRegistryWrapper exposes the package-level Prometheus registry so that
// the Init function can initialize process collectors on it.
func init() {
	DefaultRegistry.MustRegister(prometheus.NewGoCollector())
	DefaultRegistry.MustRegister(prometheus.NewProcessCollector(prometheus.ProcessCollectorOpts{}))
}
