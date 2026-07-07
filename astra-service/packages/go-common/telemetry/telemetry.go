// Package telemetry configures OpenTelemetry tracing consistently across
// every Go service so trace IDs propagate end-to-end: kiosk browser (W3C
// traceparent header) -> Go API gateway -> Go microservices -> NATS message
// headers -> back into the Rust P2P daemon's own OTel instrumentation.
package telemetry

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace"
)

// Config controls exporter target and sampling behavior. Kiosk-edge
// deployments intentionally sample at a lower rate than the cloud tier to
// bound egress bandwidth over potentially metered store internet links.
type Config struct {
	ServiceName    string
	ServiceVersion string
	OTLPEndpoint   string
	SampleRatio    float64 // 1.0 = sample everything (cloud), 0.1 = 10% (kiosk edge)
}

// Init wires up a global TracerProvider with an OTLP gRPC exporter and W3C
// trace-context propagation. Returns a shutdown func for graceful drain on
// SIGTERM — dropped traces on ungraceful shutdown are an observability gap
// during exactly the incident window they'd be needed for.
func Init(ctx context.Context, cfg Config) (func(context.Context) error, error) {
	exporter, err := otlptracegrpc.New(ctx,
		otlptracegrpc.WithEndpoint(cfg.OTLPEndpoint),
		otlptracegrpc.WithInsecure(), // TLS terminated by the local OTel collector sidecar in prod
		otlptracegrpc.WithTimeout(5*time.Second),
	)
	if err != nil {
		return nil, fmt.Errorf("telemetry: create otlp exporter: %w", err)
	}

	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName(cfg.ServiceName),
			semconv.ServiceVersion(cfg.ServiceVersion),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("telemetry: build resource: %w", err)
	}

	ratio := cfg.SampleRatio
	if ratio <= 0 {
		ratio = 1.0
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter, sdktrace.WithBatchTimeout(5*time.Second)),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sdktrace.ParentBased(sdktrace.TraceIDRatioBased(ratio))),
	)

	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	return tp.Shutdown, nil
}

// Tracer returns a named tracer — call sites use this rather than
// otel.Tracer directly so a future switch to per-service tracer providers
// is a one-file change.
func Tracer(name string) trace.Tracer {
	return otel.Tracer(name)
}
