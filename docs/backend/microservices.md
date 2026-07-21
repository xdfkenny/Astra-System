# Microservices Overview

## Architecture

13 Go microservices + 1 Python ML service + 2 standalone Go services, all communicating via gRPC with REST transcoding.

## Service Map

```
                         ┌─────────────┐
                         │   Clients    │
                         └──────┬──────┘
                                │ HTTP/REST
                         ┌──────┴──────┐
                         │API Gateway  │ ← Fiber:8080
                         │(gateway/)   │
                         └──┬───┬───┬──┘
                    ┌───────┘   │   └───────────┐
                    │  gRPC/    │   gRPC/        │
                    │  mTLS     │   mTLS         │
              ┌─────┴─────┐ ┌──┴────────┐  ┌────┴──────┐
              │Menu Service│ │Cart Service│  │Order     │
              │:8085      │ │:8081      │  │Service:83│
              └─────┬─────┘ └─────┬─────┘  └────┬──────┘
                    │              │              │
              ┌─────┴─────────────┴──────────────┴──────┐
              │           NATS JetStream (Event Bus)     │
              └─────┬─────────────┬──────────────┬──────┘
                    │              │              │
         ┌──────────┴──┐   ┌──────┴──────┐  ┌───┴──────────┐
         │Inventory    │   │Payment      │  │Sync Service  │
         │Service:8082 │   │Orchestrator │  │:8087         │
         └──────────┬──┘   │:8086        │  └───┬──────────┘
                    │      └──────┬──────┘      │
                    │             │              │
         ┌──────────┴─────────────┴──────────────┴──────┐
         │              PostgreSQL 16 (Primary DB)       │
         └───────────────────────────────────────────────┘
```

## Services Table

| Service | Port | Language | Role | Dependencies |
|---------|------|----------|------|--------------|
| `gateway` | 8080 | Go | API Gateway (Fiber) | Redis, NATS, all services |
| `menu-service` | 8085 | Go | Menu/catalog CRUD | PostgreSQL, Redis |
| `cart-service` | 8081 | Go | Cart CRDT operations | PostgreSQL, NATS |
| `order-service` | 8083 | Go | Order lifecycle | PostgreSQL, NATS, cart-service |
| `inventory-service` | 8082 | Go | Stock management | PostgreSQL, NATS |
| `payment-orchestrator` | 8086 | Go | Payment flow, offline tokens | PostgreSQL, NATS, Verifone |
| `payment-service` | - | Go | Payment processing | PostgreSQL |
| `sync-service` | 8087 | Go | Cloud sync gateway | PostgreSQL, NATS |
| `webauthn-service` | 8090 | Go | FIDO2 authentication | PostgreSQL |
| `admin-graphql` | 8092 | Go | Admin GraphQL API | PostgreSQL |
| `legacy-pos-adapter` | - | Go | Legacy POS bridge | PostgreSQL, NATS |
| `ml-lane-intel` | 8088 | Python | Lane queue estimation | Redis, ONNX model |
| `api-gateway` | - | Go | Legacy gateway (not production) | None |

## Standalone Services

| Service | Location | Language | Role |
|---------|----------|----------|------|
| `update-server` | `services/update-server/` | Go | OTA update manifest delivery |
| `astra-installer` | `installer/astra-installer/` | Go | Kiosk system installer |
| `astra-updater` | `installer/astra-updater/` | Go | Kiosk auto-updater |

## Inter-Service Communication

### Primary: gRPC (with mTLS)
```
Gateway ←→ All services (gRPC client calls)
Services ←→ Services (direct gRPC when needed)
```

### Event Bus: NATS JetStream
```
Any service → outbox_events (DB) → Outbox Relay → NATS → Consumers
```

Topics:
- `astra.cart.*` - Cart events
- `astra.order.*` - Order events
- `astra.inventory.*` - Inventory events
- `astra.payment.*` - Payment events
- `astra.sync.*` - Sync events

### Cache: Redis 7
```
Gateway → Redis (rate limiting, session cache)
menu-service → Redis (menu cache)
```

## Service Template Structure

Each Go service follows a consistent structure:

```
service-name/
├── cmd/
│   └── service-name/
│       └── main.go        # Entry point: config, DI, server start
├── internal/
│   ├── config/            # Configuration loading
│   ├── middleware/        # HTTP/gRPC middleware
│   ├── router/            # Route definitions
│   ├── service/           # Business logic
│   ├── repository/        # Data access layer
│   ├── models/            # Domain models
│   └── server/            # Server initialization
├── go.mod
└── go.sum
```

## Service Dependencies Graph

```
gateway
├── menu-service
├── cart-service
├── order-service → cart-service
├── inventory-service
├── payment-orchestrator → payment-service, Verifone
├── sync-service → all services (via NATS)
├── webauthn-service
└── admin-graphql → menu-service, inventory-service, order-service

NATS JetStream (event bus connecting all services)
PostgreSQL 16 (shared database)
Redis 7 (cache + rate limiting)
```

## Go Workspace

File: `astra-service/go.work`

The Go workspace includes 16 modules spanning services, shared libraries, and tools, enabling local `replace` directives for development without publishing.
