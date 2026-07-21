# System Design Patterns

## Architectural Patterns

### 1. Multi-Language Monorepo

The entire system is organized as a **Turborepo + pnpm workspaces** monorepo with four languages:

```
astra-service/           ← Turborepo root (pnpm workspace)
├── apps/                ← TypeScript (React 19 + Vite)
├── packages/            ← Shared: TS, Go, Rust
├── services/            ← Go microservices (13)
├── daemons/             ← Rust sidecar
├── sync-daemon/         ← Rust P2P
└── tools/               ← Rust chaos
```

**Orchestration:** `turbo.json` defines dependency-aware task pipelines (`build`, `dev`, `lint`, `test`, `typecheck`, `clean`).

**Versioning:** Changesets for semantic versioning with automated changelog generation.

### 2. Microservices Architecture (Go)

**13 Go microservices** communicate via gRPC with a Fiber-based API Gateway as the single entry point:

```
Client → API Gateway (Fiber:8080) → gRPC → Service → PostgreSQL/Redis/NATS
```

| Pattern | Implementation |
|---------|---------------|
| API Gateway | Fiber HTTP server, JWT auth, rate limiting, gRPC proxy |
| Service Discovery | Static via environment configuration |
| Inter-service Comm | gRPC with mutual TLS |
| Async Events | NATS JetStream with transactional outbox |
| Circuit Breaking | gobreaker-based middleware in gateway |
| Health Checks | `/health`, `/live`, `/ready` endpoints |

### 3. Micro-Frontend Architecture

**Module Federation** with Vite plugin composes independently-deployed frontends:

```
kiosk (host shell)
├── astra_menu    → kiosk-menu (remote)
├── astra_cart    → kiosk-cart (remote)
└── astra_payment → kiosk-payment (remote)
```

| Characteristic | Detail |
|---------------|--------|
| Framework | React 19 (shared singleton) |
| Composition | Dynamic script injection via `remote-modules.ts` |
| Versioning | Each MFE independently versioned with atomic rollback |
| Styling Isolation | CSS Modules + Tailwind CSS v4 (apps) / v3 (design system) |

### 4. Event Sourcing + Transactional Outbox

**Domain events** are persisted before publication using the outbox pattern:

```
Service → Write to DB + outbox_events table (same transaction)
       → Outbox relay reads and publishes to NATS JetStream
       → Consumers receive exactly-once delivery
```

**Event Store:** `event_store` table provides a complete audit trail of all domain events.

### 5. Offline-First with CRDTs

Three-tier resilience model:

```
Local (SQLCipher) ←→ P2P Mesh (libp2p/QUIC) ←→ Cloud (PostgreSQL)
    Tier 1                 Tier 2                   Tier 3
   Always available    Store-level sync     Cross-store sync
```

**CRDT Types Used:**
- **PN-Counter** - Inventory quantities (increment/decrement)
- **LWW-Register** - Cart item quantities (last-writer-wins)
- **OR-Set** - Cart items, modifier selections (observed-remove set)

**HLC (Hybrid Logical Clock):** Provides causal ordering without synchronized wall clocks.

### 6. P2P Mesh + Raft Consensus

| Component | Technology |
|-----------|------------|
| Transport | libp2p over QUIC |
| Encryption | Noise Protocol (XX handshake) |
| Peer Discovery | mDNS (link-local) |
| Leader Election | Raft (when 3+ kiosks) |
| State Replication | CRDT delta batches |

### 7. Strangler Fig Pattern

The **legacy-pos-adapter** service enables gradual migration from legacy POS systems:

```
Legacy POS ← Legacy-POS-Adapter → Astra Backend
                ↓
        Eventually decomissioned
```

### 8. Dual State Stores

Frontend state is split between **ephemeral session** and **persistent server** state:

| Store | Library | Purpose | Persistence |
|-------|---------|---------|-------------|
| Workflow | XState v5 | Kiosk UI state machine | Memory |
| Session | Zustand | Network status, lane mode, payment state | Memory |
| Cart | Valtio (proxy) | Reactive cart with CRDT merge | IndexedDB |
| Server Cache | TanStack Query | API data with offline-first mode | Memory + IndexedDB |

### 9. Web Worker Offloading

Heavy computations run off the main thread:

| Worker | Purpose |
|--------|---------|
| `totals.worker.ts` | Cart total computation |
| `crdtWorker.ts` | CRDT merge operations (WASM) |
| Service Worker | Image caching, menu cache, background sync |

### 10. Universal Finite State Machine (UFSM)

Kiosk workflow is modeled as a **single XState v5 machine**:

```
LANGUAGE_SELECT → ATTRACT → MENU → ITEM_DETAIL → CART → PAYMENT → PROCESSING → RECEIPT
                    ↓                                                           ↓
                 ADMIN ←─────────────────────────────────────────────────── OVERRIDE
```

Each stage maps to a micro-frontend screen via `WorkflowRouter.tsx`.

### 11. RBAC Authorization

Two-tier authorization model:

| Level | Mechanism | Scope |
|-------|-----------|-------|
| Employee | WebAuthn/FIDO2 passkeys | Store-level operations (overrides, voids) |
| Admin | JWT with RBAC claims | Admin dashboard (tenant-wide) |
| Inter-service | mTLS + SPIFFE identities | Service-to-service |
| API | JWT + HMAC request signing | External API access |

### 12. CQRS Pattern

Read and write paths are separated:

```
Write Path:  Client → gRPC → Service → PostgreSQL (aggregate tables + event store)
Read Path:   Client → REST/GraphQL → Service → PostgreSQL (read models) / Redis (cache)
```

## Data Flow Patterns

### Online Transaction Flow

```
Touch Input → XState Event → State Transition → Screen reads stores
  → apiClient (TanStack Query) → REST → API Gateway
  → gRPC → Domain Service → PostgreSQL → Response
  → Transactional Outbox → NATS → Event Consumers
  → Stores updated → UI re-renders
```

### Offline Cart Mutation

```
UI mutation → Valtio proxy → subscribe() detects change
  → CRDT merge worker (IndexedDB) → debounced 800ms
  → When online: API sync → service → PostgreSQL
```

### P2P Sync Flow

```
Mesh leader collects heartbeats + CRDT deltas
  → UploadBatch gRPC → Sync Service
  → NATS notification → Other services consume
  → DownloadBatch → Other kiosks apply deltas
```

### Ghost Cart Transfer

```
Kiosk A: QR code encodes WebRTC offer
Kiosk B: Scans QR → WebRTC answer → Data channel established
  → Cart snapshot transferred → Valtio merge
  → CRDT worker reconciles
```
