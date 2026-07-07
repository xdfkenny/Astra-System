# Astra-Service Cart Service

The `cart-service` owns the shopping cart aggregate for Astra-Service. It
exposes `astra.cart.v1.CartService` over gRPC with a REST gateway, persists
carts to PostgreSQL using optimistic locking, caches active sessions in Redis,
reserves inventory via the inventory-service gRPC client, and publishes domain
events (`ItemAddedToCart`, `CartFinalized`) through the transactional outbox +
NATS.

## Responsibilities

- Create, read, and mutate active carts (`AddItem`, `UpdateItem`, `RemoveItem`).
- Finalize carts and emit `CartFinalized` events.
- Resolve ghost carts created on offline kiosks using a CRDT merge strategy.
- Cache active sessions in Redis at `cart:{lane_id}:{session_id}` with a 30m TTL.
- Reserve stock through `inventory-service` when items are added or finalized.

## Project Layout

```
cmd/cart-service          # service entrypoint
internal/config           # environment configuration
internal/server           # gRPC + REST gateway bootstrap
internal/service          # CartService gRPC implementation
internal/repository       # Postgres repository with optimistic locking
internal/crdt             # Ghost-cart CRDT merge logic
internal/cache            # Redis session cache
internal/inventory        # inventory-service gRPC client
internal/outbox           # outbox event builders
internal/cart             # Cart aggregate + proto conversion
```

## Running Locally

1. Copy `.env.example` to `.env` and adjust values.
2. Ensure PostgreSQL, Redis, and NATS are running.
3. Run migrations in `database/migrations/`.
4. Start the service:

```bash
go run ./cmd/cart-service
```

## Tests

Unit and integration tests include cart merge logic, repository optimistic
locking, and the inventory reservation client:

```bash
go test -race ./...
```

## gRPC / REST

- gRPC: `localhost:50051`
- REST: `localhost:8081`

The service registers the gRPC reflection API for tools like `grpcurl`.
