# P2P Mesh Sync

## Overview

The P2P sync daemon (`astra-service/sync-daemon/`) enables store-local, serverless synchronization between kiosks. It provides the middle tier of the three-tier offline resilience model, allowing kiosks to share state even when completely disconnected from the cloud.

## Architecture

```
┌──────────────────────────────────────────────────────┐
│                   Kiosk A                            │
│  ┌──────────────────────────────────────────────┐   │
│  │           Sync Daemon (syncd)                 │   │
│  │  ┌─────────┐ ┌──────────┐ ┌───────────────┐  │   │
│  │  │ libp2p  │ │  CRDT    │ │   SQLCipher   │  │   │
│  │  │ + QUIC  │ │  Engine  │ │   (Storage)   │  │   │
│  │  └────┬────┘ └────┬─────┘ └───────┬───────┘  │   │
│  │       │           │                │           │   │
│  │  ┌────┴────┐ ┌────┴─────┐  ┌──────┴───────┐  │   │
│  │  │ mDNS   │ │  Raft    │  │   HLC Clock  │  │   │
│  │  │Discvry │ │Consensus │  │              │  │   │
│  │  └────────┘ └──────────┘  └──────────────┘  │   │
│  └──────────────────────────────────────────────┘   │
└──────────────────────┬───────────────────────────────┘
                       │
              ┌────────┴────────┐
              │  QUIC (Noise)   │
              │  mDNS discovery │
              └────────┬────────┘
                       │
┌──────────────────────┴───────────────────────────────┐
│                   Kiosk B                            │
│               (same as Kiosk A)                       │
└──────────────────────────────────────────────────────┘
```

## Core Components

### 1. Transport: libp2p + QUIC

- **libp2p**: Peer identity, addressing, routing, protocol negotiation
- **QUIC**: Multiplexed streams, connection migration, 0-RTT reconnection
- **Noise Protocol**: XX handshake for authenticated key exchange

### 2. Peer Discovery: mDNS

- Link-local multicast DNS for zero-config peer discovery
- Service type: `_astra-sync._udp.local`
- Peers automatically discovered within broadcast domain

### 3. Consensus: Raft

- **Leader election:** When 3+ kiosks present, Raft elects a leader
- **Sub-3s failover:** If leader disconnects, replacement elected within 3 seconds
- **Leader role:** Coordinates sync batch uploads, manages replication
- **Log replication:** Only for cluster membership and sync coordination metadata

### 4. CRDT Engine

**File:** `sync-daemon/src/crdt/`

Conflict-free Replicated Data Types for state:

| CRDT Type | Usage | Principle |
|-----------|-------|-----------|
| PN-Counter | Inventory stock levels | Separate increments and decrements |
| LWW-Register | Cart item quantities | Last-writer-wins by HLC timestamp |
| OR-Set | Cart items, modifiers | Add-wins semantics |

**Delta State:** Only changed state is transmitted between peers (delta CRDTs), not full state.

### 5. Storage: SQLCipher

- **AES-256-GCM** encrypted SQLite database
- Stores: CRDT state, pending deltas, sync metadata
- Used exclusively by syncd (not shared with kiosk browser)
- Encrypted at rest with key derived from kiosk hardware identity

### 6. HLC (Hybrid Logical Clock)

Provides causal ordering without synchronized wall clocks:
```
Timestamp = (physical_ms, logical_counter, node_id)
Comparison: wall clock → logical → node ID
```

## Sync Protocol Flow

### Peer Discovery
```
1. Kiosk boots → syncd starts
2. mDNS broadcasts presence (service: _astra-sync)
3. Existing peers respond
4. libp2p connections established via QUIC
5. Noise handshake for authenticated encryption
```

### State Synchronization
```
1. Local mutation recorded in SQLCipher
2. CRDT delta generated
3. If leader: publish delta to all peers
4. If follower: forward delta to leader
5. Leader batches deltas
6. Periodic UploadBatch to cloud sync-service (if online)
7. DownloadBatch from cloud (if online)
8. CRDT merge applied locally
```

### Leader Election (3+ kiosks)
```
1. All kiosks start in follower state
2. Election timeout (150-300ms random) triggers
3. Candidate requests votes from peers
4. Majority vote → leader elected (~3s worst case)
5. Leader starts coordinating sync
6. Leader handles cloud communication on behalf of mesh
```

## Data Flow: End-to-End Sync

```
User adds item on Kiosk A (offline)
  → Valtio proxy → IndexedDB
  → CRDT delta generated
  → syncd stores delta in SQLCipher
  → P2P: delta transmitted to Kiosk B via QUIC
  → Kiosk B CRDT engine merges delta
  → Kiosk B applies to local state
  
When cloud reconnects:
  → Mesh leader UploadBatch to sync-service
  → Cloud stores in PostgreSQL
  → Other stores' kiosks DownloadBatch
  → Converged across all stores
```

## Sync Event Types

| Event | Description |
|-------|-------------|
| `sync_event_type: cart_updated` | Cart CRDT delta |
| `sync_event_type: order_created` | New order |
| `sync_event_type: payment_made` | Payment recorded |
| `sync_event_type: inventory_adjusted` | Stock change |

## Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `ASTRA_SYNCD_LISTEN_QUIC` | 0.0.0.0:4499 | QUIC listen address |
| `ASTRA_SYNCD_MDNS_SERVICE` | _astra-sync._udp.local | mDNS service type |
| `ASTRA_SYNCD_RAFT_ELECTION_TIMEOUT_MS` | 3000 | Raft election timeout |
| `ASTRA_SYNCD_DATA_DIR` | /var/lib/astra/syncd | Data directory |
| `ASTRA_SYNCD_BOOTSTRAP_PEERS` | - | Static peer addresses |
| `KIOSK_MESH_PSK` | - | Pre-shared key for mesh auth |

## Rust Crate

**Location:** `astra-service/sync-daemon/`

**Key Sub-modules:**
| Module | Purpose |
|--------|---------|
| `src/p2p/` | libp2p host, peer management, QUIC transport |
| `src/crdt/` | CRDT types (PN-Counter, LWW-Register, OR-Set) |
| `src/raft/` | Raft consensus implementation |
| `src/storage/` | SQLCipher persistence layer |
| `src/sync/` | Sync protocol, delta exchange |
| `src/network/` | Network management, reconnection |
| `src/crypto/` | Noise protocol, key management |
| `src/protocol/` | Wire protocol definitions |
| `src/cloud/` | Cloud sync gateway client |
| `src/grpc/` | gRPC server for local kiosk communication |
| `src/offline/` | Offline token management |
| `src/store/` | Local state store |
| `src/telemetry/` | OpenTelemetry integration |
| `src/verifone/` | Verifone payment bridge |
