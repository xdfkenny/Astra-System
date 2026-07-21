# Offline-First Strategy

## Overview

Astra-System is designed to operate autonomously for **48 hours** without any cloud connectivity. This is achieved through a three-tier resilience model that ensures continuous operation even in complete network isolation.

## Three-Tier Resilience

```
┌──────────────────────────────────────────────────┐
│               Tier 3: Cloud Sync                  │
│  PostgreSQL (HA) │ NATS JetStream │ Redis        │
│  Sync Service (gRPC) for batch upload/download    │
└──────────────────────┬───────────────────────────┘
                       │ Internet (TLS)
┌──────────────────────┴───────────────────────────┐
│               Tier 2: P2P Mesh                    │
│  libp2p/QUIC │ Raft Consensus │ CRDT Replication │
│  Store-local sync between kiosks                  │
└──────────────────────┬───────────────────────────┘
                       │ Local Network
┌──────────────────────┴───────────────────────────┐
│               Tier 1: Local Kiosk                 │
│  SQLCipher (encrypted SQLite) │ IndexedDB        │
│  Service Worker (background sync queue)           │
│  Works 100% offline                               │
└──────────────────────────────────────────────────┘
```

## Key Components

### 1. Local Storage Layer (Tier 1)

**SQLCipher** (encrypted SQLite via `sync-daemon`):
- Encrypted at rest with AES-256
- Stores orders, cart state, inventory snapshots
- Syncs bidirectionally with cloud when online

**IndexedDB** (browser):
- Cart state via Valtio proxy subscriber
- Cached API responses via TanStack Query
- Background sync queue via Service Worker

**Service Worker** (`service-worker.ts`):
- **CacheFirst**: Kiosk images (30d TTL)
- **StaleWhileRevalidate**: Menu/catalog data
- **BackgroundSync**: Failed API requests (48h retry queue)

### 2. P2P Mesh Layer (Tier 2)

See [P2P Mesh Sync](../networking/p2p-mesh.md) for full details.

When multiple kiosks exist at a store, they form a P2P mesh:
- **libp2p** for peer identity and routing
- **QUIC** for transport (multiplexed, connection migration)
- **Noise Protocol** for encryption
- **mDNS** for link-local peer discovery
- **Raft** for leader election (3+ nodes, sub-3s failover)
- **CRDTs** for conflict-free state replication

### 3. Cloud Sync Layer (Tier 3)

**Sync Service** (`sync-service/`):
- **UploadBatch**: Kiosk sends CRDT deltas to cloud
- **DownloadBatch**: Kiosk fetches cloud changes
- **StreamHeartbeats**: Periodic health/liveness reporting

**Data Sync Flow:**
```
Offline Period (48h)
  ├── All transactions recorded locally in SQLCipher
  ├── CRDT deltas accumulated
  └── State: Local → eventual consistency within store mesh

Reconnection
  ├── Heartbeat sent to sync-service
  ├── CRDT delta batch uploaded (UploadBatch)
  ├── Cloud changes downloaded (DownloadBatch)
  ├── Conflicts resolved via CRDT merge rules
  └── State: Converged → consistent with cloud
```

## CRDT Implementation

### Types Used

| CRDT Type | Usage | Merge Rule |
|-----------|-------|------------|
| PN-Counter | Inventory stock levels | Value = incrementSum - decrementSum |
| LWW-Register | Cart item quantity | Last-writer-wins by HLC timestamp |
| OR-Set | Cart items, modifiers | Add observed, remove only if in set |

### HLC (Hybrid Logical Clock)

File: `packages/shared-types/src/hlc.ts`

```
Timestamp = (physicalWallClock, logicalCounter)
  - physicalWallClock: Unix milliseconds
  - logicalCounter: Monotonically increasing per node
  - Comparison: Wall clock first, then logical counter, then node ID
```

### CRDT Merge Worker

File: `apps/kiosk/src/workers/crdtWorker.ts`

- Runs in a Web Worker (off-main-thread)
- Lazily loads Rust WASM module for performance
- Functions: `merge_cart_ops`, `hash_event_chain`
- Triggered by Valtio proxy changes with 800ms debounce

## Offline Capabilities

| Feature | Offline Support | Sync When Online |
|---------|----------------|------------------|
| Menu Browsing | ✅ (cached) | StaleWhileRevalidate |
| Cart Operations | ✅ (CRDT + IndexedDB) | Debounced 800ms |
| Payment (Card) | ✅ (offline tokens) | Batch settlement |
| Payment (Cash) | ✅ (local record) | Immediate |
| Order Creation | ✅ (SQLCipher) | Background sync |
| Inventory Lookup | ✅ (local snapshot) | On reconnection |
| Employee Auth | ✅ (WebAuthn local) | Credential sync |
| Produce Recognition | ✅ (ONNX model local) | N/A |
| Admin Dashboard | ❌ (requires cloud) | N/A |

## Offline Token System

For card payments when offline, the system uses **offline payment tokens**:

```
1. Kiosk generates signed offline token
2. Token stored in offline_tokens table
3. Customer receives confirmation
4. When online: tokens submitted for batch settlement
5. Settlement handled by payment-orchestrator
```

**Configuration:**
- `PAYMENT_OFFLINE_TOKEN_SEED` - Seed for token generation
- `PAYMENT_MAX_OFFLINE_AMOUNT_CENTS` - Maximum offline transaction value
- `PAYMENT_OFFLINE_TTL_SECONDS` - Time window for settlement

## Reconnection Protocol

```
1. Network detection → useNetworkMonitor (5s poll at :4499/healthz)
2. Sync daemon reconnects to cloud gateway
3. Heartbeat sent → UploadBatch of accumulated CRDT deltas
4. Ack received → DownloadBatch of cloud changes
5. CRDT merge applied locally
6. Offline tokens submitted for settlement
7. State marked as "synced"
```
