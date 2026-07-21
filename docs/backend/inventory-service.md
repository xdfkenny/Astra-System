# Inventory Service

## Overview

The Inventory Service (`services/inventory-service/`) manages stock levels, reservations with TTL, and inventory transactions.

## Responsibilities

- Stock level queries per store/item
- Soft reservations with configurable TTL
- Stock adjustments (receiving, counting, damage)
- Real-time stock updates via SSE streaming
- CRDT-based sync for offline inventory accuracy

## Endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET | `/stock?store_id=&item_id=` | Get stock level |
| POST | `/reserve` | Reserve stock (soft hold with TTL) |
| POST | `/release` | Release reservation |
| POST | `/adjust` | Adjust stock (count, receive) |
| GET | `/stream` | SSE: real-time stock updates |

## gRPC Endpoints

| RPC | Description |
|-----|-------------|
| `GetStock` | Query stock level |
| `ReserveStock` | Create time-limited reservation |
| `ReleaseStock` | Release reservation |
| `AdjustStock` | Adjust inventory count |
| `StreamStockUpdates` | Server-sent streaming updates |

## Reservation Lifecycle

1. Reservation created with TTL (default: 15 minutes)
2. If not converted to order within TTL → automatic release
3. On order creation → reservation converted to deduction
4. On order cancellation → stock added back
