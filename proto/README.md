# Astra-Service Protobuf Definitions

This directory contains the Protocol Buffer v3 schema and generated-code layout
for the Astra-Service platform.

## Layout

```text
proto/
├── proto/              # .proto source files
│   ├── auth.proto
│   ├── cart.proto
│   ├── common.proto
│   ├── events.proto
│   ├── inventory.proto
│   ├── lane.proto
│   ├── menu.proto
│   ├── order.proto
│   ├── payment.proto
│   ├── sync.proto
│   └── webauthn.proto
├── gen/go/             # Generated Go packages (protoc output)
│   ├── auth/
│   ├── cart/
│   ├── common/
│   ├── events/
│   ├── inventory/
│   ├── lane/
│   ├── menu/
│   ├── order/
│   ├── payment/
│   ├── sync/
│   └── webauthn/
├── buf.yaml            # Buf module configuration
├── buf.gen.yaml        # Buf code-generation configuration
├── generate.go         # go:generate entry point
├── go.mod
└── go.sum
```

## Packages

| Domain     | Proto package        | Go import path                                                |
| ---------- | -------------------- | ------------------------------------------------------------- |
| auth       | `astra.auth.v1`      | `github.com/astra-systems/astra-service/proto/gen/go/auth`    |
| cart       | `astra.cart.v1`      | `github.com/astra-systems/astra-service/proto/gen/go/cart`    |
| common     | `astra.common.v1`    | `github.com/astra-systems/astra-service/proto/gen/go/common`  |
| events     | `astra.events.v1`    | `github.com/astra-systems/astra-service/proto/gen/go/events`  |
| inventory  | `astra.inventory.v1` | `github.com/astra-systems/astra-service/proto/gen/go/inventory` |
| lane       | `astra.lane.v1`      | `github.com/astra-systems/astra-service/proto/gen/go/lane`    |
| menu       | `astra.menu.v1`      | `github.com/astra-systems/astra-service/proto/gen/go/menu`    |
| order      | `astra.order.v1`     | `github.com/astra-systems/astra-service/proto/gen/go/order`   |
| payment    | `astra.payment.v1`   | `github.com/astra-systems/astra-service/proto/gen/go/payment` |
| sync       | `astra.sync.v1`      | `github.com/astra-systems/astra-service/proto/gen/go/sync`    |
| webauthn   | `astra.webauthn.v1`  | `github.com/astra-systems/astra-service/proto/gen/go/webauthn` |

## Regenerating Code

### Prerequisites

- Go 1.25+
- `buf` CLI
- `protoc-gen-go` and `protoc-gen-go-grpc`:

```bash
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
```

### Using go generate

From the module root:

```bash
cd /Users/xdfke/Documents/MOTHER/Astra-System/proto
go generate ./...
```

### Using buf directly

```bash
cd /Users/xdfke/Documents/MOTHER/Astra-System/proto
buf generate
```

### Using protoc directly

If you prefer `protoc` without `buf`:

```bash
cd /Users/xdfke/Documents/MOTHER/Astra-System/proto
mkdir -p gen/go

protoc \
  --proto_path=proto \
  --go_out=gen/go --go_opt=paths=source_relative \
  --go-grpc_out=gen/go --go-grpc_opt=paths=source_relative \
  proto/auth.proto \
  proto/cart.proto \
  proto/common.proto \
  proto/events.proto \
  proto/inventory.proto \
  proto/lane.proto \
  proto/menu.proto \
  proto/order.proto \
  proto/payment.proto \
  proto/sync.proto \
  proto/webauthn.proto
```

## Go Module

The module is `github.com/astra-systems/astra-service/proto`. Generated packages
live under `gen/go/<domain>` and import `google.golang.org/protobuf` and
`google.golang.org/grpc`.

After generating real `.pb.go` files, run `go mod tidy` to populate `go.sum`
with checksums for `google.golang.org/grpc` and `google.golang.org/protobuf`.
Until then, the placeholder packages compile without importing those modules,
so `go.sum` only contains the previously required `github.com/google/uuid`.
