# legacy-pos-adapter

Strangler Fig adapter that forwards completed Astra carts/orders to a legacy POS system while storing the submission result in Astra.

## Behavior

- Listens to `astra.order.created.v1` on NATS JetStream.
- When `LEGACY_POS_URL` is set, converts the order into a legacy POS payload and POSTs it to `LEGACY_POS_URL/v1/orders`.
- Persists the request, response, status code, and any error in a submission record.
- When `LEGACY_POS_URL` is empty, the adapter records submissions as skipped.

## Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `LEGACY_POS_URL` | `` | Legacy POS base URL. Empty disables forwarding. |
| `LEGACY_POS_API_KEY` | `` | Optional Bearer token sent to the legacy POS. |
| `LEGACY_POS_TIMEOUT` | `10s` | HTTP timeout for legacy POS requests. |
| `NATS_URL` | `nats://localhost:4222` | NATS server URL. |
| `GRPC_PORT` | `8087` | gRPC listener port. |
| `HTTP_PORT` | `8088` | HTTP/REST listener port. |

## Run

```bash
go run ./cmd/legacy-pos-adapter
```

## Test

```bash
go test ./...
```
