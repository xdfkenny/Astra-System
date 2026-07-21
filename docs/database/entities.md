# Entity Relationships

## Domain Model

### Tenant → Store → Kiosk

```
Tenant (1) ──→ (N) Store
Store (1) ──→ (N) Kiosk
Tenant (1) ──→ (N) Location
Location (1) ──→ (N) Lane
```

**Key Relationships:**
- A Tenant owns multiple Stores and Locations
- A Store has multiple Kiosks (self-checkout devices)
- A Location has multiple Lanes (checkout lanes)

### Store → Menu

```
Store (1) ──→ (N) Category (hierarchical, parent_id → self)
Category (1) ──→ (N) Item
Item (N) ──→ (N) ModifierGroup (via item_modifier_groups)
ModifierGroup (1) ──→ (N) ModifierOption
```

### Store → Inventory

```
Store (1) ──→ (N) Inventory (per item)
Item (1) ──→ (1) Inventory (per store)
Inventory (1) ──→ (N) InventoryTransaction
Inventory (1) ──→ (N) InventoryReservation (TTL-based)
```

### Kiosk → Cart → Order

```
Kiosk (1) ──→ (N) Cart
Cart (1) ──→ (N) CartLine
Cart (1) ──→ (1) Order
Order (1) ──→ (N) OrderItem
```

**Cart Status Flow:** `active → finalized → converted`

**Order Status Flow:** `pending → confirmed → preparing → ready → fulfilled`
**Order Status Flow (alt):** `pending → cancelled`
**Order Status Flow (alt):** `fulfilled → refunded`

### Order → Payment

```
Order (1) ──→ (N) Payment
Payment (1) ──→ (N) Refund
Kiosk (1) ──→ (N) OfflineToken (when offline)
```

**Payment Status Flow:** `pending → authorized → captured → settled`
**Payment Status Flow (alt):** `pending → failed`
**Payment Status Flow (alt):** `settled → refunded | partially_refunded`

### Employee → Store

```
Store (1) ──→ (N) Employee
Employee (1) ──→ (1) Role

Tenant (1) ──→ (N) Role
Role (N) ──→ (N) Permission (via role_permissions)
Permission (1) ──→ (resource, action)
```

### Audit/Events

```
Any aggregate (polymorphic) ──→ AuditLog
Any aggregate (polymorphic) ──→ EventStore
Kiosk ──→ SyncEvent
```

## Key IDs

All primary keys use UUID v4. Foreign keys reference UUIDs throughout.

## Concurrency

- **Carts** use optimistic concurrency with a `version` integer field
- **Inventory** uses row-level locking (`SELECT ... FOR UPDATE`) during reservation
- **Audit log** uses hash chaining (`prev_hash` → `hash`) for immutability

## Cascading

- Deleting a `Store` cascades to `Kiosks`, `Categories`, `Items`, `Inventory`, `Carts`, `Orders`
- Deleting a `Cart` cascades to `CartLines`
- Deleting an `Order` cascades to `OrderItems`, `Payments`
- Deleting a `Payment` cascades to `Refunds`
