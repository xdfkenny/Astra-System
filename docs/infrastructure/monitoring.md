# Observability Stack

## Overview

Three pillars of observability implemented via **OpenTelemetry** collector with **Prometheus** (metrics), **Loki** (logs), **Jaeger** (traces), and **Grafana** (visualization).

## Architecture

```
┌─────────┐   ┌──────────────┐   ┌───────────┐
│ Service │──▶│ OpenTelemetry │──▶│  Jaeger   │ Traces
│  (Go)   │   │   Collector   │   │  (traces) │
└─────────┘   │  (0.111)     │   └───────────┘
              │              │──▶┌───────────┐
┌─────────┐   │              │   │Prometheus │ Metrics
│ Kiosk   │──▶│  OTLP gRPC   │   │ (metrics) │
│ (React) │   │  + HTTP      │   └───────────┘
└─────────┘   │              │──▶┌───────────┐
              │              │   │   Loki    │ Logs
┌─────────┐   │              │   │  (logs)   │
│ Syncd   │──▶│              │   └───────────┘
└─────────┘   └──────────────┘
                      │
                      ▼
              ┌──────────────┐
              │   Grafana    │
              │ (dashboards) │
              └──────────────┘
```

## Components

### OpenTelemetry Collector

**File:** `infra/otel/otel-collector.yml`

```yaml
receivers:
  otlp:
    protocols:
      grpc: 4317
      http: 4318
processors:
  batch: {}
exporters:
  otlp/jaeger: { endpoint: jaeger:4317 }
  prometheus: { endpoint: 0.0.0.0:8889 }
  loki: { endpoint: http://loki:3100/loki/api/v1/push }
```

### Prometheus

**File:** `infra/prometheus/prometheus.yml`

Scraping configuration for:
- All Go microservices (`/metrics` endpoint)
- Node exporter (host metrics)
- PostgreSQL exporter
- Redis exporter
- NATS exporter

### Grafana

**Location:** `infra/grafana/`

- **Dashboards:** Pre-built dashboards for service health, business metrics, P2P mesh status
- **Datasources:** Prometheus, Loki, Jaeger, PostgreSQL (for business queries)

### Loki

**File:** `infra/loki/loki.yml`

Log aggregation for:
- All container stdout/stderr
- Structured JSON logs parsed via json extraction
- 30-day retention

### Jaeger

OTLP-compatible trace collection:
- Distributed tracing across all Go services
- Trace sampling rate: configurable via `ASTRA_TRACE_SAMPLE_RATE`
- gRPC span context propagation

## Key Metrics

### Business Metrics
- `orders_total{status}` - Order counts by status
- `revenue_total{store}` - Revenue
- `cart_abandonment_rate` - Cart abandonment
- `payment_success_rate` - Payment success
- `offline_token_count` - Offline tokens pending settlement
- `lane_queue_depth` - Queue length per lane

### Infrastructure Metrics
- `service_request_duration_ms{p50,p95,p99}` - Latency
- `service_error_rate` - Error percentage
- `grpc_request_count{service,method}` - gRPC call volume
- `redis_cache_hit_ratio` - Cache efficiency
- `nats_queue_depth` - Event queue depth
- `p2p_mesh_peer_count` - Connected peers
- `p2p_sync_lag_ms` - Sync latency

### Resource Metrics
- `container_cpu_usage`, `container_memory_usage`
- `postgres_connections`, `postgres_query_duration`
- `disk_io`, `network_io`

## Health Endpoints

Every service exposes:
- `/health` - Overall health (includes dependency checks)
- `/live` - Liveness (is process alive?)
- `/ready` - Readiness (is service ready to serve?)

## Logging

**Format:** Structured JSON

```json
{
  "level": "info",
  "service": "order-service",
  "trace_id": "abc123",
  "message": "Order created",
  "order_id": "uuid",
  "amount_cents": 1500
}
```

**Log Levels:** `debug`, `info`, `warn`, `error`, `fatal`
