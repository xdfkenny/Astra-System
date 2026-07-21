# Order Service

## Overview

The Order Service (`services/order-service/`) manages the complete order lifecycle: creation, fulfillment, cancellation, and refunds.

## Responsibilities

- Order creation from finalized carts
- Order status management
- Idempotent order creation (`Idempotency-Key`)
- Order item price snapshots
- Integration with inventory service for stock reservation

## Endpoints

| Method | Path | Description |
|--------|------|-------------|
| POST | `/v1/orders` | Create order (Idempotency-Key required) |
| GET | `/v1/orders` | List orders (filterable) |
| GET | `/v1/orders/{id}` | Get order details |
| PATCH | `/v1/orders/{id}/status` | Update order status |
| POST | `/v1/orders/{id}/fulfill` | Fulfill order |
| POST | `/v1/orders/{id}/refund` | Refund order |

## Order Status Model

```
pending → confirmed → preparing → ready → fulfilled
    ↓                                        ↓
cancelled                                refunded
```

## Events Published

- `OrderCreated` → inventory service (reserve stock)
- `OrderFulfilled` → inventory service (deduct stock)
- `PaymentInitiated` (via payment orchestrator)
