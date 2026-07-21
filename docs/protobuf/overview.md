# Protobuf Overview

## Purpose

Protocol Buffers define the service contracts for all gRPC-based inter-service communication. They serve as the single source of truth for API contracts between services.

## Location

- **Source definitions:** `proto/proto/*.proto` (12 files)
- **Generated Go code:** `proto/gen/go/` (11 packages)
- **Generator tool:** `protoc` + `buf`

## File Index

| File | Package | Services Defined |
|------|---------|-----------------|
| `common.proto` | `astra.common.v1` | None (shared types) |
| `auth.proto` | `astra.auth.v1` | AuthService |
| `cart.proto` | `astra.cart.v1` | CartService |
| `events.proto` | `astra.events.v1` | None (event types) |
| `inventory.proto` | `astra.inventory.v1` | InventoryService |
| `lane.proto` | `astra.lane.v1` | LaneService |
| `menu.proto` | `astra.menu.v1` | MenuService |
| `order.proto` | `astra.order.v1` | OrderService |
| `payment.proto` | `astra.payment.v1` | PaymentOrchestrator |
| `sync.proto` | `astra.sync.v1` | SyncService |
| `webauthn.proto` | `astra.webauthn.v1` | WebAuthnService |
| `location.proto` | `astra.location.v1` | LocationService |

## Code Generation

### With Buf (recommended)

```bash
cd proto
buf generate
```

### With protoc

```bash
protoc -I proto/proto \
  --go_out=gen/go --go_opt=paths=source_relative \
  --go-grpc_out=gen/go --go-grpc_opt=paths=source_relative \
  proto/proto/*.proto
```

### Generated Output

```
proto/gen/go/
├── auth/       → auth.pb.go, auth_grpc.pb.go
├── cart/       → cart.pb.go, cart_grpc.pb.go
├── common/     → common.pb.go
├── events/     → events.pb.go
├── inventory/  → inventory.pb.go, inventory_grpc.pb.go
├── lane/       → lane.pb.go, lane_grpc.pb.go
├── menu/       → menu.pb.go, menu_grpc.pb.go
├── order/      → order.pb.go, order_grpc.pb.go
├── payment/    → payment.pb.go, payment_grpc.pb.go
├── sync/       → sync.pb.go, sync_grpc.pb.go
├── webauthn/   → webauthn.pb.go, webauthn_grpc.pb.go
└── location/   → location.pb.go, location_grpc.pb.go
```

## Conventions

- **Package:** `astra.{service}.v1`
- **Messages:** PascalCase, snake_case fields
- **Enums:** Prefix with service name, UPPER_SNAKE values
- **Services:** PascalCase with `Service` suffix
- **RPCs:** PascalCase, streaming via `stream` keyword
- **Comments:** Document all fields, messages, and RPCs

## CI Validation

The CI pipeline runs Buf checks on every PR modifying `proto/`:
- `buf format` — style check
- `buf lint` — best practices
- `buf breaking` — backward compatibility check
