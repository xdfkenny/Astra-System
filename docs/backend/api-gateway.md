# API Gateway

## Overview

The API Gateway (`services/gateway/`) is the single entry point for all external requests. Built with **Fiber** (Go HTTP framework), it handles:

- HTTP routing and proxying to backend services
- JWT authentication and authorization
- Rate limiting (Redis-backed sliding window token bucket)
- Circuit breaking (per-service)
- HMAC request signing verification
- CORS management
- Request logging and tracing

## Entry Point

**File:** `cmd/gateway/main.go`

```go
func main() {
    config.Load()
    // Initialize Redis, NATS, OpenTelemetry
    // Register routes
    // Start Fiber server on :8080
}
```

## Routes

| Path | Method(s) | Handler | Auth |
|------|-----------|---------|------|
| `/health` | GET | HealthCheck | None |
| `/live` | GET | Liveness | None |
| `/ready` | GET | Readiness | None |
| `/metrics` | GET | Prometheus | None |
| `/docs/*` | GET | Swagger UI | None |
| `/v1/menu` | GET | Proxy to menu-service | Optional |
| `/v1/menu/*` | ALL | Proxy to menu-service | Optional |
| `/v1/menu/stream` | GET | SSE proxy | JWT |
| `/v1/cart/*` | ALL | Proxy to cart-service | JWT |
| `/v1/order/*` | ALL | Proxy to order-service | JWT |
| `/v1/inventory/*` | ALL | Proxy to inventory-service | JWT |
| `/v1/payment/*` | ALL | Proxy to payment-orchestrator | JWT |
| `/v1/sync/*` | ALL | Proxy to sync-service | mTLS |

## Middleware

### Rate Limiter

**File:** `internal/middleware/ratelimit.go`

Redis-backed sliding window token bucket using Lua scripting:
- Default: 100 req/s with 200 burst
- Configurable per route via `GATEWAY_RATE_LIMIT_RPS` and `GATEWAY_RATE_LIMIT_BURST`

### Request Signing

**File:** `internal/middleware/signing.go`

HMAC-SHA256 request signing for sensitive endpoints:
- Client signs: `HMAC(body + timestamp + nonce, key)`
- Server verifies: recomputes HMAC, checks timestamp skew (<30s), nonce uniqueness
- Key sourced from `GATEWAY_HMAC_SIGNING_KEY`

### Circuit Breaker

**File:** `internal/middleware/circuitbreaker.go`

Per-service circuit breaker using `gobreaker`:
- Failure threshold: 5 consecutive failures
- Timeout: 30s half-open
- Returns 503 when open

## Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `GATEWAY_PORT` | 8080 | HTTP listen port |
| `GATEWAY_JWT_ISSUER` | astra-system | JWT issuer claim |
| `GATEWAY_HMAC_SIGNING_KEY` | - | HMAC key for request signing |
| `GATEWAY_RATE_LIMIT_RPS` | 100 | Requests per second |
| `GATEWAY_RATE_LIMIT_BURST` | 200 | Burst allowance |
| `GATEWAY_CORS_ORIGINS` | * | Allowed CORS origins |
| `REDIS_URL` | localhost:6379 | Redis for rate limiting |
| `NATS_URL` | nats://localhost:4222 | NATS for events |
