# Menu Service

The `menu-service` owns the catalog read model for Astra-Service: categories,
items, and modifier groups. It exposes a gRPC API defined in
`proto/proto/menu.proto` and an optional REST gateway.

## Features

- gRPC implementation of `astra.menu.v1.MenuService`
- REST gateway on port 8085 (proxies to gRPC on port 50051)
- PostgreSQL repository using `pgx/v5` via `database/sql` with prepared statements
- Redis caching for menus and category lists
- Transactional outbox: every write also inserts into `outbox_events`
- Outbox relay publishes `MenuUpdated` and `ItemPriceChanged` events to NATS JetStream

## Configuration

Copy `.env.example` to `.env` and adjust values.

| Variable | Default | Description |
|----------|---------|-------------|
| `MENU_SERVICE_GRPC_PORT` | 50051 | gRPC listen port |
| `MENU_SERVICE_HTTP_PORT` | 8085 | REST gateway listen port |
| `DATABASE_URL` | `postgresql://astra:astra@localhost:5432/astra?sslmode=disable` | Postgres DSN |
| `REDIS_URL` | `localhost:6379` | Redis address |
| `REDIS_PASSWORD` | "" | Redis password |
| `REDIS_DB` | 0 | Redis database |
| `NATS_URL` | `nats://localhost:4222` | NATS server URL |
| `CACHE_TTL` | 5m | Redis cache TTL |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | "" | OpenTelemetry collector endpoint |
| `ASTRA_ENV` | development | Environment name |

## Running locally

```bash
cd astra-service/services/menu-service
export DATABASE_URL="postgresql://astra:astra@localhost:5432/astra?sslmode=disable"
export REDIS_URL="localhost:6379"
export NATS_URL="nats://localhost:4222"
go run ./cmd/menu-service
```

## Tests

```bash
go test -race ./...
```

Repository tests spin up a real PostgreSQL container using
testcontainers-go and apply the platform migrations from `database/migrations`.
Cache tests use an embedded Redis (miniredis).

## NATS Events

The service writes the following outbox event types, which the relay publishes
to NATS JetStream under the `ASTRA_MENU` stream:

- `MenuUpdated` → `astra.menu.updated.v1`
- `ItemPriceChanged` → `astra.menu.item.price_changed.v1`

## Regenerating protobuf code

From the `proto/` directory:

```bash
buf generate
```

or with `protoc`:

```bash
protoc \
  --proto_path=proto \
  --go_out=gen/go --go_opt=paths=source_relative \
  --go-grpc_out=gen/go --go-grpc_opt=paths=source_relative \
  --grpc-gateway_out=gen/go --grpc-gateway_opt=paths=source_relative \
  proto/common.proto proto/menu.proto
```
