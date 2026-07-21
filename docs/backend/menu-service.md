# Menu Service

## Overview

The Menu Service (`services/menu-service/`) manages the product catalog: categories, items, modifier groups, and options. It serves menu data with Redis caching for low-latency access.

## Responsibilities

- Full menu retrieval by store
- Category hierarchy management
- Item CRUD with modifier associations
- Item search (barcode, PLU, name)
- Menu cache invalidation
- SSE streaming for menu updates

## Endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET | `/v1/menu/{store_id}` | Full menu with categories, items, modifiers |
| GET | `/v1/categories/{store_id}` | Category tree only |
| GET | `/v1/items/{item_id}` | Single item detail |
| GET | `/v1/items/search?q=` | Search by name, barcode, PLU |

## Cache Strategy

- **Redis:** Full menu cached by store ID (TTL: 300s)
- **TanStack Query (client):** 60s stale time, 30min GC
- **Service Worker:** StaleWhileRevalidate for menu pages
- **Cache invalidation:** On menu update → NATS event → cache clear
