# inventory-service

The Astra-Service inventory microservice owns real-time stock levels and the
soft-reservation pattern used by carts. It exposes gRPC and REST interfaces,
persists inventory as an insert-only Postgres ledger, caches derived levels in
Redis, and publishes `InventoryReserved`, `InventoryReleased`, and
`InventoryAdjusted` events via NATS through the transactional outbox pattern.

## Features

- **Ledger-style inventory**: all quantity changes are append-only rows in
  `inventory_transactions`; available and reserved quantities are computed from
  the ledger plus active reservations.
- **Soft reservations**: stock is held for a cart with a TTL and released by a
  background worker when the TTL expires or by an explicit release call.
- **Redis cache**: derived `StockLevel`s are cached under
  `inventory:{store_id}:{item_id}`.
- **Outbox + NATS**: every reserve, release, and adjust operation writes an
  outbox event in the same database transaction; a relay publishes to NATS
  JetStream subjects under `astra.inventory.*`.

## API

### gRPC

`InventoryService` is defined in `proto/proto/inventory.proto`.

- `GetStock`
- `ReserveStock`
- `ReleaseStock`
- `AdjustStock`
- `StreamStockUpdates`

### REST

| Method | Path | Description |
|--------|------|-------------|
| GET | `/health` | Health check |
| GET | `/live` | Liveness check |
| GET | `/stock?store_id=&item_id=` | Get stock level |
| POST | `/reserve` | Reserve stock |
| POST | `/release` | Release stock |
| POST | `/adjust` | Adjust stock |

## Configuration

Configuration is read from environment variables. See `.env.example` for the
full list.

Key variables:

- `PORT` — HTTP port (default `8082`)
- `GRPC_PORT` — gRPC port (default `9092`)
- `DATABASE_URL` — Postgres connection string
- `REDIS_URL` — Redis connection string
- `NATS_URL` — NATS connection string
- `ASTRA_RESERVATION_TTL` — soft-reservation TTL (default `5m`)
- `ASTRA_RESERVATION_SWEEP` — expiry worker interval (default `30s`)

## Running locally

```bash
cd astra-service/services/inventory-service
export DATABASE_URL="postgresql://astra:astra@localhost:5432/astra?sslmode=disable"
export REDIS_URL="redis://localhost:6379/0"
export NATS_URL="nats://localhost:4222"
go run ./cmd/inventory-service
```

## Tests

```bash
go test -race ./...
```

Tests use in-memory repository, cache, and publisher implementations so no
external services are required.

## Database schema

The service auto-creates the following tables on startup:

- `inventory` — base record per store/item (on-order, reorder points, location)
- `inventory_transactions` — insert-only ledger of every quantity change
- `inventory_reservations` — active soft holds with `expires_at_ms`
- `outbox_events` — transactional outbox (managed by `go-common/outbox`)
