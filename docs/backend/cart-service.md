# Cart Service

## Overview

The Cart Service (`services/cart-service/`) manages shopping cart state using CRDT-based operations for conflict-free merging across kiosks.

## Responsibilities

- Cart CRDT operations (create, add, update, remove items)
- Cart versioning for optimistic concurrency
- Ghost cart merge (WebRTC-transferred carts)
- Cart finalization and checkout
- Event publishing for order service consumption

## gRPC Endpoints

| RPC | Description |
|-----|-------------|
| `CreateCart` | Create new cart for a kiosk/session |
| `GetCart` | Get cart by ID with version |
| `AddItem` | Add line item with modifiers |
| `UpdateItem` | Update quantity or modifiers |
| `RemoveItem` | Remove line item |
| `FinalizeCart` | Lock cart for checkout (prevents further mutations) |
| `MergeGhostCart` | Merge another cart's items (ghost cart transfer) |

## CRDT Integration

All cart mutations generate CRDT deltas that are:
1. Applied to the service-side cart state
2. Published as NATS events for sync-service
3. Replicated across kiosks via P2P mesh
