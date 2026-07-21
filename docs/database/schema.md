# Database Schema

## Overview

**Database:** PostgreSQL 16
**ORM:** Drizzle (TypeScript) in `database/schemas/drizzle.ts`
**Migrator:** Raw SQL migrations in `database/migrations/`
**Mirror:** Go structs in `database/schemas/go_structs.go`

## Entity Relationship Diagram (Domains)

```
Tenants ──→ Locations ──→ Lanes
              │
              ├──→ Stores ──→ Kiosks
              │
              ├──→ Categories ──→ Items ──→ ModifierGroups ──→ ModifierOptions
              │
              ├──→ Inventory ──→ InventoryTransactions
              │         └──→ InventoryReservations
              │
              ├──→ Carts ──→ CartLines
              │
              ├──→ Orders ──→ OrderItems
              │
              ├──→ Payments ──→ Refunds
              │         └──→ OfflineTokens
              │
              ├──→ Employees ──→ Roles ──→ RolePermissions ──→ Permissions
              │
              └──→ Users
```

## Tables (22)

### Tenant/Location/Lane Hierarchy

| Table | Key Columns | Purpose |
|-------|-------------|---------|
| `tenants` | id, name, slug, plan (enum), billing_info | Multi-tenant organizations |
| `locations` | id, tenant_id, name, address, timezone | Physical locations |
| `lanes` | id, location_id, name, lane_type, queue_depth | Checkout lanes |

### Stores/Kiosks

| Table | Key Columns | Purpose |
|-------|-------------|---------|
| `stores` | id, tenant_id, name, address, settings | Retail stores |
| `kiosks` | id, store_id, hardware_id, sync_status (enum), is_leader, signing_key_hash | Kiosk devices |

### Menu System

| Table | Key Columns | Purpose |
|-------|-------------|---------|
| `categories` | id, store_id, parent_id, name, sort_order | Menu categories (hierarchical) |
| `items` | id, store_id, category_id, name, price_cents, barcode, plu, tax_category (enum), weight_supported | Sellable items |
| `modifier_groups` | id, name, min_select, max_select | Option groups |
| `modifier_options` | id, group_id, name, price_delta_cents | Individual options |
| `item_modifier_groups` | item_id, modifier_group_id | Many-to-many join |

### Inventory

| Table | Key Columns | Purpose |
|-------|-------------|---------|
| `inventory` | store_id, item_id, available_qty, reserved_qty, reorder_point | Stock levels |
| `inventory_transactions` | id, store_id, item_id, qty_change, type (enum) | Stock movement ledger |
| `inventory_reservations` | id, store_id, item_id, qty, expires_at, session_id | Soft holds with TTL |

### Carts

| Table | Key Columns | Purpose |
|-------|-------------|---------|
| `carts` | id, store_id, kiosk_id, status (enum), version, items_json, totals_json | Active/finalized carts |
| `cart_lines` | id, cart_id, item_id, quantity, unit_price_cents, modifiers_json | Line items |

### Orders

| Table | Key Columns | Purpose |
|-------|-------------|---------|
| `orders` | id, cart_id, store_id, kiosk_id, status (enum), total_cents, payment_status | Completed orders |
| `order_items` | id, order_id, item_id, quantity, unit_price_cents | Order line items |

### Payments

| Table | Key Columns | Purpose |
|-------|-------------|---------|
| `payments` | id, order_id, method (enum), status (enum), amount_cents, verifone_token, idempotency_key | Payment records |
| `refunds` | id, payment_id, amount_cents, reason, status (enum) | Refund records |
| `offline_tokens` | id, store_id, kiosk_id, amount_cents, expires_at, signature, settled_at | Offline settlement queue |

### Employees/Users/RBAC

| Table | Key Columns | Purpose |
|-------|-------------|---------|
| `employees` | id, store_id, name, role (enum), biometric_hash, webauthn_credential_id | Store staff |
| `roles` | id, tenant_id, name, description | RBAC roles |
| `permissions` | id, resource, action, description | Fine-grained permissions |
| `role_permissions` | role_id, permission_id | Role-permission mapping |
| `users` | id, tenant_id, email, webauthn_credentials | Admin users |

### Audit/Events/Sync/Analytics

| Table | Key Columns | Purpose | Partitioning |
|-------|-------------|---------|--------------|
| `audit_logs` | id, event_type (enum), actor_id, resource_type, resource_id, old_values, new_values, hash, prev_hash | Immutable audit trail | Monthly |
| `event_store` | id, event_type, aggregate_id, aggregate_type, payload, metadata | Domain event history | None |
| `outbox_events` | id, event_type, payload, status, created_at | Transactional outbox | None |
| `sync_events` | id, event_type (enum), source_kiosk_id, target_kiosk_id, crdt_delta | CRDT sync log | None |
| `analytics_events` | id, event_type, store_id, payload, created_at | Anonymized analytics | Monthly |

## PostgreSQL Enums (13)

| Enum Name | Values |
|-----------|--------|
| `tenant_plan` | free, starter, business, enterprise |
| `kiosk_sync_status` | online, offline, syncing, error |
| `item_tax_category` | standard, reduced, zero, exempt |
| `weight_unit` | g, kg, lb, oz, each |
| `cart_status` | active, finalized, abandoned, merged, converted |
| `order_status` | pending, confirmed, preparing, ready, fulfilled, cancelled, refunded |
| `payment_method` | credit_card, debit_card, cash, mobile_wallet, gift_card, store_credit |
| `payment_status` | pending, authorized, captured, settled, failed, refunded, partially_refunded |
| `employee_role` | cashier, supervisor, manager, admin |
| `audit_event_type` | create, update, delete, login, logout, override, payment, refund |
| `inventory_transaction_type` | received, sold, returned, adjusted, transferred, counted, damaged |
| `sync_event_type` | cart_updated, order_created, payment_made, inventory_adjusted |
| `refund_status` | pending, approved, rejected, processed |

## Indexes

Key indexes include:
- Composite indexes on `(store_id, item_id)` for inventory lookups
- Composite indexes on `(tenant_id, slug)` for tenant lookups
- Indexes on `(order_id, status)` for order queries
- `(expires_at)` on `inventory_reservations` for TTL cleanup
- `(idempotency_key)` on `payments` for idempotency
- `(event_type, created_at)` on partitioned tables

## Outbox Pattern

The `outbox_events` table enables reliable event publishing:
1. Service writes to DB in same transaction as domain operation
2. Outbox relay reads unpublished events
3. Publishes to NATS JetStream
4. Marks as published

## Partitioning

`audit_logs` and `analytics_events` are partitioned monthly by `created_at` for query performance and data management.

**File:** `database/migrations/0004_partitioning.sql`
