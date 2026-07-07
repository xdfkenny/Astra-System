# payment-orchestrator

The Go payment orchestrator service for Astra-Service coordinates card-present
payments through the Verifone sidecar, maintains a durable payment state machine,
and publishes payment domain events to NATS JetStream via the transactional
outbox pattern.

## Responsibilities

- gRPC implementation of `astra.payment.v1.PaymentOrchestrator`.
- REST API for payment lifecycle and webhook ingestion.
- Idempotency keyed on `Idempotency-Key` (Redis lock + Postgres uniqueness).
- Payment state machine: `PENDING → AUTHORIZING → CAPTURED → SETTLED → FAILED`.
- Verifone client abstraction with gRPC-to-Rust-syncd preference and HTTP fallback.
- Asynchronous Verifone webhook handling with HMAC-SHA256 verification.
- Offline token settlement with HMAC verification and batch settlement.
- Transactional outbox + NATS publishing for `PaymentInitiated`,
  `PaymentConfirmed`, and `PaymentFailed`.

## Module

```
github.com/astra-systems/astra-service/services/payment-orchestrator
```

## Configuration

Copy `.env.example` to `.env` and adjust values. The service reads environment
variables (see `internal/config/config.go`).

## Running locally

```bash
cd astra-service
export PATH="$PWD/../.toolchain/go/bin:$PWD/../.toolchain/protoc/bin:$PATH"
go work sync
cd services/payment-orchestrator
go run ./cmd/payment-orchestrator
```

## Tests

```bash
go test -race ./...
```

## API

### gRPC

- `astra.payment.v1.PaymentOrchestrator` on port `50086` (default).

### REST

| Method | Path                               | Description                          |
|--------|------------------------------------|--------------------------------------|
| POST   | `/v1/payments/`                    | Initiate a payment                   |
| POST   | `/v1/payments/:id/capture`         | Capture an authorizing payment       |
| POST   | `/v1/payments/:id/settle`          | Settle a captured payment            |
| POST   | `/v1/payments/:id/refund`          | Refund a captured/settled payment    |
| POST   | `/v1/payments/webhooks/verifone`   | Verifone async notification webhook  |
| POST   | `/v1/offline-tokens/settle`        | Batch settle offline tokens          |
| GET    | `/health`, `/live`, `/ready`       | Health probes                        |
| GET    | `/metrics`                         | Prometheus metrics                   |

## Webhook signatures

Verifone notifications must include an `X-Verifone-Signature` header containing
the HMAC-SHA256 hex digest of the request body signed with `PAYMENT_WEBHOOK_SECRET`.
