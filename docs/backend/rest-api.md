# REST API Reference

## Base URL

- Development: `http://localhost:8080`
- Production: `https://{gateway-endpoint}`

All endpoints are proxied through the API Gateway.

## Authentication

See [Authentication](../security/authentication.md) for details.

## Health Endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET | `/health` | Overall health check |
| GET | `/live` | Liveness probe |
| GET | `/ready` | Readiness probe |
| GET | `/metrics` | Prometheus metrics |

## Menu Service

Proxied at `/v1/menu/*`

| Method | Path | Description |
|--------|------|-------------|
| GET | `/v1/menu/{store_id}` | Get full store menu |
| GET | `/v1/stores/{store_id}/menu` | Get store menu (alt) |
| GET | `/v1/categories/{store_id}` | Get categories |
| GET | `/v1/items/{item_id}` | Get single item |
| GET | `/v1/items/search?q=` | Search items |
| GET | `/v1/menu/stream` | SSE menu updates |

**Response Types:**
- `Category`: id, store_id, name, description, sort_order, parent_id, image_url
- `Item`: id, store_id, name, description, price_cents, barcode, plu, tax_category, weight_supported, image_url, modifier_group_ids
- `ModifierGroup`: id, name, min_select, max_select
- `ModifierOption`: id, group_id, name, price_delta_cents

## Cart Service

Proxied at `/v1/cart/*`

| Method | Path | Description |
|--------|------|-------------|
| GET | `/v1/carts/{cart_id}` | Get cart |
| POST | `/v1/carts` | Create cart |
| POST | `/v1/carts/{cart_id}/items` | Add item |
| PATCH | `/v1/carts/{cart_id}/items/{line_id}` | Update item |
| DELETE | `/v1/carts/{cart_id}/items/{line_id}` | Remove item |
| POST | `/v1/carts/{cart_id}/checkout` | Finalize cart |

**Request/Response:**
```json
{
  "cart_id": "uuid",
  "store_id": "uuid",
  "items": [{
    "line_id": "uuid",
    "item_id": "uuid",
    "quantity": 2,
    "modifiers": [{"group_id": "uuid", "option_id": "uuid"}],
    "unit_price_cents": 499,
    "line_total_cents": 998
  }],
  "totals": {
    "subtotal_cents": 998,
    "tax_cents": 70,
    "fee_cents": 10,
    "total_cents": 1078
  },
  "version": 5,
  "status": "active"
}
```

## Order Service

Proxied at `/v1/order/*`

| Method | Path | Description |
|--------|------|-------------|
| POST | `/v1/orders` | Create order (Idempotency-Key required) |
| GET | `/v1/orders` | List orders |
| GET | `/v1/orders/{id}` | Get order |
| PATCH | `/v1/orders/{id}/status` | Update order status |
| POST | `/v1/orders/{id}/fulfill` | Fulfill order |
| POST | `/v1/orders/{id}/refund` | Refund order |

**Order Statuses:** `pending`, `confirmed`, `preparing`, `ready`, `fulfilled`, `cancelled`, `refunded`

## Inventory Service

Proxied at `/v1/inventory/*`

| Method | Path | Description |
|--------|------|-------------|
| GET | `/stock?store_id=&item_id=` | Get stock level |
| POST | `/reserve` | Reserve stock |
| POST | `/release` | Release reservation |
| POST | `/adjust` | Adjust stock (count) |
| GET | `/stream` | SSE stock updates |

## Payment Orchestrator

Proxied at `/v1/payment/*`

| Method | Path | Description |
|--------|------|-------------|
| POST | `/v1/payments/` | Create payment intent |
| POST | `/v1/payments/{id}/capture` | Capture payment |
| POST | `/v1/payments/{id}/settle` | Settle payment |
| POST | `/v1/payments/{id}/refund` | Refund payment |
| POST | `/v1/payments/webhooks/verifone` | Verifone webhook |
| POST | `/v1/offline-tokens/settle` | Batch offline settlement |

**Payment Methods:** `credit_card`, `debit_card`, `cash`, `mobile_wallet`, `gift_card`, `store_credit`
**Payment Statuses:** `pending`, `authorized`, `captured`, `settled`, `failed`, `refunded`, `partially_refunded`

## Sync Service

Proxied at `/v1/sync/*`

| Method | Path | Description |
|--------|------|-------------|
| POST | `/v1/sync/upload` | Upload CRDT delta batch |
| POST | `/v1/sync/download` | Download cloud changes |
| POST | `/v1/sync/heartbeat` | Kiosk heartbeat |

## WebAuthn Service

| Method | Path | Description |
|--------|------|-------------|
| POST | `/v1/auth/webauthn/begin` | Begin WebAuthn assertion |
| POST | `/v1/auth/webauthn/verify` | Verify WebAuthn assertion |
| POST | `/v1/auth/override/validate` | Validate override token |
| POST | `/v1/webauthn/register/begin` | Start credential registration |
| POST | `/v1/webauthn/register/finish` | Complete credential registration |
| POST | `/v1/webauthn/authenticate/begin` | Start authentication |
| POST | `/v1/webauthn/authenticate/finish` | Complete authentication |

## Admin GraphQL

| Method | Path | Description |
|--------|------|-------------|
| POST | `/graphql` | Admin GraphQL queries |

Requires JWT with `is_admin: true` claim.

**Available Queries:**
- `menus(storeId)`, `menu(id)`
- `inventory(storeId)`, `stockLevel(itemId, storeId)`
- `orders(status, dateRange)`, `order(id)`
- `payments(dateRange)`, `payment(id)`
- `employees(storeId)`, `employee(id)`
- `kiosks(storeId)`, `kiosk(id)`
- `auditLogs(dateRange, eventType)`

## Update Server

| Method | Path | Description |
|--------|------|-------------|
| GET | `/manifest.json` | Signed OTA update manifest |
| POST | `/webhook/health` | Kiosk health report |

## Error Responses

All errors follow a consistent format:

```json
{
  "error": {
    "code": "NOT_FOUND",
    "message": "Cart not found",
    "details": {"cart_id": "abc-123"}
  }
}
```

**Standard Error Codes:**
- `NOT_FOUND` - Resource not found (404)
- `BAD_REQUEST` - Invalid input (400)
- `UNAUTHORIZED` - Missing/invalid auth (401)
- `FORBIDDEN` - Insufficient permissions (403)
- `CONFLICT` - Version conflict (409)
- `RATE_LIMITED` - Too many requests (429)
- `INTERNAL_ERROR` - Server error (500)
- `SERVICE_UNAVAILABLE` - Circuit breaker open (503)

## Idempotency

State-changing endpoints support idempotency via the `Idempotency-Key` header:

```
POST /v1/orders
Idempotency-Key: unique-client-generated-id
```

- Keys expire after 24 hours
- Same key within expiry returns original response
- Used for order creation, payment operations
