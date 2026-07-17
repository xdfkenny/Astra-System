# Astra-Service Architecture

## Document Control

| Field        | Value                                     |
| ------------ | ----------------------------------------- |
| Version      | 1.1                                       |
| Last Updated | 2026-07-05                                |
| Author       | Astra-Service Engineering Team            |
| Status       | Approved                                  |
| Related RFCs | RFC-001 Offline-First, RFC-004 Zero-Trust |

## Table of Contents

1. [System Overview](#system-overview)
2. [Offline-First Strategy](#offline-first-strategy)
3. [P2P Mesh and Raft Consensus](#p2p-mesh-and-raft-consensus)
4. [CRDTs and Hybrid Logical Clocks](#crdts-and-hybrid-logical-clocks)
5. [Event Sourcing and Transactional Outbox](#event-sourcing-and-transactional-outbox)
6. [Security Model](#security-model)
7. [Payment Flow](#payment-flow)
8. [Deployment and CI/CD](#deployment-and-cicd)
9. [Observability](#observability)
10. [Deep Improvements Summary](#deep-improvements-summary)
11. [Appendices](#appendices)

---

## System Overview

Astra-Service is a production-grade, offline-first automated self-checkout platform built for 24/7 retail environments. It enables zero-downtime store operation with **48 hours of offline resilience**, zero-trust security, and a peer-to-peer mesh sync layer between kiosks.

### Design Goals

| Goal               | Target                                                                 |
| ------------------ | ---------------------------------------------------------------------- |
| Offline resilience | 48 hours of autonomous operation with no cloud connectivity            |
| Latency            | < 200 ms menu load, < 500 ms P2P inventory sync, < 3 s leader failover |
| Availability       | 99.99% uptime for the cloud tier; 100% uptime during local-only mode   |
| Security           | Zero trust, mTLS everywhere, PCI-DSS compliant payment path            |
| Scale              | 1–10,000 kiosks per tenant; multi-region cloud deployment              |

### High-Level Topology

```text
┌─────────────────────────────────────────────────────────────────┐
│                         Cloud Tier                              │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐          │
│  │ API Gateway  │  │ Order Svc    │  │ Payment Svc  │          │
│  │ (Go/Fiber)   │  │ (Go)         │  │ (Go)         │          │
│  └──────────────┘  └──────────────┘  └──────────────┘          │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐          │
│  │ Inventory Svc│  │ Cart Svc     │  │ Sync Svc     │          │
│  │ (Go)         │  │ (Go)         │  │ (Go)         │          │
│  └──────────────┘  └──────────────┘  └──────────────┘          │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐          │
│  │ PostgreSQL 16│  │ Redis 7      │  │ NATS JetStream│          │
│  │ (Primary)    │  │ (Cache/Session│  │ (Event Bus)  │          │
│  └──────────────┘  └──────────────┘  └──────────────┘          │
└─────────────────────────────────────────────────────────────────┘
                              │
                    Internet (TLS 1.3)
                              │
┌─────────────────────────────────────────────────────────────────┐
│                    Store Edge / Kiosk Cluster                   │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐             │
│  │ Kiosk 1     │  │ Kiosk 2     │  │ Kiosk N     │             │
│  │ ┌─────────┐ │  │ ┌─────────┐ │  │ ┌─────────┐ │             │
│  │ │React 19 │ │  │ │React 19 │ │  │ │React 19 │ │             │
│  │ │UI Shell │ │  │ │UI Shell │ │  │ │UI Shell │ │             │
│  │ └─────────┘ │  │ └─────────┘ │  │ └─────────┘ │             │
│  │ ┌─────────┐ │  │ ┌─────────┐ │  │ ┌─────────┐ │             │
│  │ │Rust P2P │ │  │ │Rust P2P │ │  │ │Rust P2P │ │             │
│  │ │Sync Dmn │ │  │ │Sync Dmn │ │  │ │Sync Dmn │ │             │
│  │ └─────────┘ │  │ └─────────┘ │  │ └─────────┘ │             │
│  │ ┌─────────┐ │  │ ┌─────────┐ │  │ ┌─────────┐ │             │
│  │ │SQLite   │ │  │ │SQLite   │ │  │ │SQLite   │ │             │
│  │ │(offline)│ │  │ │(offline)│ │  │ │(offline)│ │             │
│  │ └─────────┘ │  │ └─────────┘ │  │ └─────────┘ │             │
│  └─────────────┘  └─────────────┘  └─────────────┘             │
│                              │                                  │
│                    Local Mesh (libp2p QUIC + mDNS)              │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐             │
│  │ Verifone    │  │ Thermal     │  │ Barcode/    │             │
│  │ Terminal    │  │ Printer     │  │ NFC/Scale   │             │
│  └─────────────┘  └─────────────┘  └─────────────┘             │
└─────────────────────────────────────────────────────────────────┘
```

### Service Inventory

| Service         | Language   | Responsibility                                   |
| --------------- | ---------- | ------------------------------------------------ |
| `api-gateway`   | Go         | Edge routing, authN/authZ, rate limiting         |
| `order-svc`     | Go         | Order lifecycle, cart persistence, fulfillment   |
| `payment-svc`   | Go         | Payment orchestration, token settlement          |
| `inventory-svc` | Go         | Stock levels, soft holds, catalog sync           |
| `cart-svc`      | Go         | Cart CRDT merge, ghost-cart resolution           |
| `sync-svc`      | Go         | Cloud-side mesh gateway and batch ingestion      |
| `astra-syncd`   | Rust       | Kiosk P2P daemon, CRDT sync, Verifone FFI bridge |
| `kiosk-shell`   | TypeScript | React 19 customer UI, peripheral integration     |
| `update-server` | Go         | Signed OTA manifest delivery                     |

---

## Offline-First Strategy

Astra-Service is designed to function **100% for up to 48 hours** without internet connectivity. Resilience is built in three layers:

### 1. Local Data Layer

Each kiosk maintains an encrypted SQLite database (SQLCipher) containing:

- Full menu catalog (categories, items, modifiers, prices)
- Current inventory counts with reserved-stock soft holds
- Pending transactions and offline payment tokens
- Employee biometric hashes (irreversible, for offline authentication)
- Cart state and Ghost Cart replicas

Catalogs are refreshed every 6 hours while online. If a kiosk has been offline for more than 6 hours, it continues to serve the cached catalog and marks prices as stale in the UI.

### 2. P2P Mesh Layer

Kiosks discover peers via mDNS on the local network, form a mesh using libp2p with QUIC transport and the Noise protocol, and replicate state with CRDTs. If 3+ kiosks are present, a Raft cluster elects a leader responsible for cloud upload when connectivity returns.

### 3. Graceful Degradation

| Capability       | Online behavior                           | Offline behavior                                         |
| ---------------- | ----------------------------------------- | -------------------------------------------------------- |
| Payment          | Verifone terminal auth + cloud settlement | Terminal auth + local signed token queued for settlement |
| Inventory        | Cloud DB + P2P sync                       | P2P CRDT sync only                                       |
| New orders       | Persisted to cloud + outbox               | Stored locally + P2P replicated                          |
| Employee auth    | WebAuthn / biometric sync with cloud      | Local biometric hash verification                        |
| Software updates | Download and apply automatically          | Deferred until online                                    |

### 48-Hour Resilience Design

- Inventory syncs every 30 seconds when online and every 5 seconds via P2P mesh.
- Payment tokens have a 48-hour TTL. They must be uploaded before expiry.
- If connectivity is restored after 48 hours, kiosks enter **reconciliation mode**. A store manager must verify the offline queue before settlement proceeds.

---

## P2P Mesh and Raft Consensus

### libp2p Mesh Network

| Property  | Implementation                                           |
| --------- | -------------------------------------------------------- |
| Transport | QUIC over UDP (low latency, connection migration, 0-RTT) |
| Security  | Noise XX handshake for authenticated encryption          |
| Discovery | mDNS on local LAN; optional DHT bootstrap nodes          |
| Protocol  | Custom `/astra-sync/1.0.0` protocol for sync messages    |
| Identity  | Ed25519 key pair provisioned during manufacturing        |

All P2P sync traffic is encrypted with **XChaCha20-Poly1305** via the Noise protocol. Every sync message is signed by the sender's identity key and includes a monotonic nonce for replay protection.

### Raft Leader Election

When 3+ kiosks are present, a Raft cluster forms. The leader is responsible for:

1. Uploading transaction batches to the cloud payment orchestrator.
2. Fetching cloud menu/inventory updates and distributing them to the mesh.
3. Reporting aggregate store health metrics.

Raft parameters:

- Heartbeat interval: 1 second
- Election timeout: 2–4 seconds (randomized)
- Expected leader failover: < 3 seconds

With 1–2 kiosks, no leader election occurs. Each kiosk queues independently and uploads when online.

### Partition Handling

If the mesh splits into two partitions, each partition continues to operate. When partitions reunite, CRDT merge rules resolve divergent state deterministically. Conflicts that require human judgment (e.g., price overrides, refunds) are flagged in the manager dashboard.

---

## CRDTs and Hybrid Logical Clocks

Astra-Service uses three CRDT strategies selected per data type.

### PN-Counter for Inventory

Inventory quantities are modeled as a **Positive-Negative Counter**. Each kiosk maintains two maps: one for increments (restocks) and one for decrements (sales/holds). The resulting count is deterministic regardless of merge order.

```text
count = sum(increments across all peers) - sum(decrements across all peers)
```

Soft holds are treated as decrements with a TTL. If a hold expires without conversion, the decrement is canceled via an increment entry.

### LWW-Register for Cart State

Cart state uses a **Last-Writer-Wins Register** with Hybrid Logical Clock (HLC) timestamps. Each mutation updates the cart's HLC timestamp. When two kiosks modify the same Ghost Cart, the higher HLC wins. If HLCs tie, the lexicographically larger kiosk ID wins.

### OR-Set for Transaction Logs

Transaction logs are stored as an **Observed-Removed Set**. Each transaction is uniquely identified by a UUID v7. Transactions are never deleted; they are marked with a tombstone. This provides an immutable audit trail compatible with Merkle tree verification.

### Hybrid Logical Clocks (HLC)

HLC combines physical wall-clock time with a logical counter to preserve causality without requiring perfect clock synchronization:

```text
HLC = (physical_time_ms, logical_counter)
```

- When a local event occurs, the counter increments.
- When a message is received, the peer's HLC is merged and the counter increments.
- HLC values are monotonic and capture the "happens-before" relationship across kiosks.

This lets Astra-Service order events consistently even when kiosks have drifted RTC clocks.

### Sync Priorities

| Priority | Data              | Latency Target | Transport                |
| -------- | ----------------- | -------------- | ------------------------ |
| 0        | Inventory changes | < 500 ms       | Direct P2P broadcast     |
| 1        | Transaction logs  | < 5 s          | Batched, zstd compressed |
| 2        | Analytics events  | < 60 s         | Bandwidth-aware batch    |

---

## Event Sourcing and Transactional Outbox

### Event Sourcing Model

The cloud services use event sourcing as the source of truth. Every domain mutation (order created, payment authorized, inventory decremented) is appended to an event stream in NATS JetStream. Projections in PostgreSQL and Redis are rebuilt from these streams and can be replayed for recovery.

Event schema:

```json
{
  "event_id": "uuidv7",
  "event_type": "order.payment_authorized",
  "aggregate_id": "order-123",
  "tenant_id": "tenant-456",
  "lane_id": "lane-7",
  "kiosk_id": "kiosk-9",
  "hlc": "(1712345678901,42)",
  "payload": { ... },
  "metadata": { "trace_id": "...", "user_agent": "..." }
}
```

### Transactional Outbox

Every database write is preceded by an outbox entry in the same transaction. A background worker polls the outbox and publishes events to NATS. An outbox entry is deleted only after NATS acknowledges the publish. This guarantees **exactly-once publication** even if the service crashes between the DB commit and the event publish.

Outbox table (simplified):

```sql
CREATE TABLE outbox (
  id UUID PRIMARY KEY,
  topic TEXT NOT NULL,
  payload JSONB NOT NULL,
  headers JSONB,
  created_at TIMESTAMPTZ NOT NULL,
  processed_at TIMESTAMPTZ
);
CREATE INDEX idx_outbox_created ON outbox(created_at) WHERE processed_at IS NULL;
```

Worker behavior:

1. `SELECT ... FOR UPDATE SKIP LOCKED` fetches unprocessed rows.
2. Publish to NATS with idempotency key = `event_id`.
3. On ACK, delete the row (or mark processed).
4. On failure, retry with exponential backoff up to 1 hour; alert after 10 failures.

---

## Security Model

### Zero-Trust Architecture

Every internal boundary is authenticated and encrypted. No service trusts another based on network location alone.

| Boundary                    | Mechanism                                         |
| --------------------------- | ------------------------------------------------- |
| Kiosk → API Gateway         | HMAC-SHA256 request signing with per-kiosk key    |
| API Gateway → Microservices | mTLS + service-account JWT                        |
| Service → Service           | gRPC with mTLS and per-service SPIFFE identity    |
| P2P Kiosk ↔ Kiosk           | Noise protocol with Ed25519 identity verification |
| Kiosk → Verifone Terminal   | PCI-PTS encrypted serial/Ethernet channel         |
| Kiosk → Local DB            | SQLCipher AES-256 + filesystem encryption (LUKS)  |

### mTLS PKI

A private Certificate Authority (CFSSL or Vault PKI) issues short-lived certificates:

- Service certificates: 24-hour TTL, auto-rotated.
- Kiosk certificates: 90-day TTL, renewed via update daemon.
- Client certificates: issued to store managers for admin access.

All certificates include SANs for service DNS names and `localhost` for local development.

### Secrets Management

| Environment | Store                                             | Rotation                 |
| ----------- | ------------------------------------------------- | ------------------------ |
| Cloud       | HashiCorp Vault with per-kiosk AppRole            | Daily or on lease expiry |
| Local dev   | SOPS + age encrypted YAML files                   | Manual                   |
| Kiosk OS    | OS keychain via 99designs/keyring abstraction     | On provisioning          |
| Runtime     | In-memory only; never written to unencrypted disk | On process restart       |

Offline kiosks use pre-provisioned 7-day emergency signing keys. Emergency key usage triggers an audit alert when the kiosk reconnects.

### PCI-DSS Compliance Path

- Card data (PAN, track data, CVV) **never** touches kiosk storage or application memory.
- The Verifone terminal handles all card data within its own PCI scope.
- The kiosk only receives: opaque token, authorization status, last-4 digits, and card brand.
- Payment tokens are encrypted at rest (SQLCipher) and in transit (TLS 1.3 / Noise).
- Audit logs are append-only with Merkle tree verification.

### Web Security

- Content Security Policy: `default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'`
- No `eval()`; no inline scripts.
- HSTS with preload (`max-age=63072000`).
- iframe sandboxing for any third-party content.
- CORS: strict origin allowlist; `*` is never used.

---

## Payment Flow

### Online Payment Flow

1. Customer taps to start a session; the kiosk creates a UUID v7 cart.
2. Items are added; cart totals are computed in a Web Worker.
3. At checkout, the kiosk calls `VxStartTransaction(amount, currency)` via the Rust Verifone FFI bridge.
4. The Verifone terminal prompts for card/NFC and performs online authorization.
5. On success, `VxProcessPayment` returns an opaque token and authorization code.
6. The kiosk records the transaction, prints the ESC/POS receipt, and decrements inventory via P2P sync.

### Offline Payment Flow

Steps 1–3 are identical. In offline mode:

4. The Verifone terminal performs authorization independently (it has its own connectivity for auth) or falls back to offline PIN authorization depending on acquirer config.
5. The kiosk signs the payment result with HMAC-SHA256 using the kiosk's daily signing key.
6. The signed token is queued in local SQLite.
7. The transaction is broadcast to all kiosks in the mesh; every kiosk stores it.
8. When connectivity returns, the Raft leader uploads the batch to the cloud payment orchestrator.
9. The payment orchestrator verifies signatures and settles with the acquirer. Failures are flagged for manual review.

### Payment Token Schema

```json
{
  "token_id": "uuidv7",
  "amount": 2499,
  "currency": "USD",
  "terminal_id": "verifone-abc123",
  "kiosk_id": "kiosk-9",
  "timestamp": "2026-07-05T07:28:59Z",
  "auth_code": "A1B2C3",
  "signature": "base64(hmac-sha256(payload, daily_key))"
}
```

### Verifone FFI Bridge

The Rust crate `astra-verifone-ffi` exposes a safe API over the vendor C SDK:

```rust
pub fn init_terminal() -> Result<TerminalHandle, VerifoneError>
pub fn start_transaction(amount: u64, currency: &str) -> Result<Transaction, VerifoneError>
pub fn wait_for_card() -> Result<CardPresented, VerifoneError>
pub fn process_payment() -> Result<PaymentToken, VerifoneError>
pub fn refund(transaction_id: &str) -> Result<RefundReceipt, VerifoneError>
pub fn close_terminal() -> Result<(), VerifoneError>
```

Error codes from the C SDK are mapped to a strongly typed `VerifoneError` enum.

---

## Deployment and CI/CD

### Kiosk Hardware

- Industrial ARM64/x64 kiosk with 8 GB RAM and 128 GB industrial SSD (-20°C to 60°C).
- Thermal printer (ESC/POS), barcode scanner, NFC reader, scale, Verifone terminal, camera.
- OS: Custom Linux (Yocto or Ubuntu Core) with A/B partition updates.
- Container runtime: Docker in rootless mode or Podman.
- Browser: Chromium in kiosk mode (`--kiosk --no-first-run --noerrdialogs`).
- Daemon: `astra-syncd` (Rust) runs as a systemd service with `Restart=always`.

### Cloud Infrastructure

| Component     | Technology                                 |
| ------------- | ------------------------------------------ |
| Orchestration | Kubernetes (EKS/GKE) with node autoscaling |
| Database      | PostgreSQL 16 with read replicas           |
| Cache         | Redis 7 Cluster                            |
| Event Bus     | NATS JetStream (3+ replicas)               |
| Observability | Prometheus + Grafana + Loki + Jaeger       |
| Secrets       | HashiCorp Vault with auto-unseal           |
| Edge Router   | Traefik with mTLS termination              |

### CI/CD Pipeline

The GitHub Actions workflow (`.github/workflows/ci.yml`) includes:

| Stage             | Details                                                        |
| ----------------- | -------------------------------------------------------------- |
| Lint              | Biome/ESLint for TS, `golangci-lint` for Go, `clippy` for Rust |
| Unit tests        | `pnpm test`, `go test -race ./...`, `cargo nextest`            |
| Integration tests | Docker Compose stack + test harness                            |
| E2E tests         | Playwright against kiosk-shell                                 |
| Docker builds     | Matrix: `linux/amd64`, `linux/arm64`; distroless base images   |
| Security audit    | `npm audit`, `cargo audit`, `govulncheck`, Trivy scan          |
| SBOM + signing    | Syft generates SPDX; cosign/Sigstore signs images              |

Containers are built with `docker/build-push-action` using `cache-from`/`cache-to` registry caching. OTA delta patches are generated via `bsdiff` for kiosk software updates.

### Deployment Topology

```text
┌──────────────────────────────────────────────────────────┐
│                     Internet                              │
│                    (TLS 1.3)                             │
└──────────────────────────────────────────────────────────┘
                          │
┌──────────────────────────────────────────────────────────┐
│              Cloud Load Balancer (Traefik)               │
│              WAF + DDoS Protection                      │
└──────────────────────────────────────────────────────────┘
                          │
┌──────────────────────────────────────────────────────────┐
│              Kubernetes Cluster (EKS/GKE)                │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐  │
│  │ API GW   │  │ Order    │  │ Payment  │  │ Inventory│  │
│  │ (3 pods) │  │ (3 pods) │  │ (3 pods) │  │ (3 pods) │  │
│  └──────────┘  └──────────┘  └──────────┘  └──────────┘  │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐                │
│  │ Cart     │  │ Sync     │  │ Admin    │                │
│  │ (3 pods) │  │ (2 pods) │  │ (2 pods) │                │
│  └──────────┘  └──────────┘  └──────────┘                │
└──────────────────────────────────────────────────────────┘
                          │
              Store VPN (WireGuard) or Direct TLS
                          │
┌──────────────────────────────────────────────────────────┐
│                    Store LAN (isolated)                   │
│  ┌──────────────────────────────────────────────────┐   │
│  │  Mesh Network: libp2p + mDNS + QUIC              │   │
│  │  ┌──────────┐  ┌──────────┐  ┌──────────┐       │   │
│  │  │ Kiosk 1  │  │ Kiosk 2  │  │ Kiosk N  │       │   │
│  │  │ 10.0.1.11│  │ 10.0.1.12│  │ 10.0.1.N │       │   │
│  │  └──────────┘  └──────────┘  └──────────┘       │   │
│  │                                                  │   │
│  │  Peripherals:                                    │   │
│  │  ┌──────────┐  ┌──────────┐  ┌──────────┐       │   │
│  │  │ Verifone │  │ Printer  │  │ Scanner  │       │   │
│  │  │ 10.0.1.21│  │ USB/Serial│  │ USB/Serial│       │   │
│  │  └──────────┘  └──────────┘  └──────────┘       │   │
│  └──────────────────────────────────────────────────┘   │
└──────────────────────────────────────────────────────────┘
```

---

## Observability

### Telemetry Signals

| Signal  | Instrumentation                                      | Backend        |
| ------- | ---------------------------------------------------- | -------------- |
| Metrics | Prometheus client libraries (Go, Rust, TS)           | Prometheus     |
| Traces  | OpenTelemetry SDK with OTLP export                   | Jaeger/Grafana |
| Logs    | Structured JSON with trace/context IDs; PII redacted | Loki           |
| Health  | `/health`, `/ready`, `/live` HTTP endpoints          | Kubernetes     |

### Key Metrics

| Metric                         | Description                                  |
| ------------------------------ | -------------------------------------------- |
| `astra_sync_lag_seconds`       | P2P sync lag per kiosk                       |
| `astra_offline_queue_depth`    | Pending offline transactions                 |
| `astra_payment_success_rate`   | Rolling 5-minute payment success rate        |
| `astra_mesh_peer_count`        | Connected peers per kiosk                    |
| `astra_cart_abandonment_rate`  | Percentage of carts abandoned before payment |
| `astra_thermal_printer_errors` | Printer fault count                          |

### Logging Conventions

Structured JSON logs include:

- `trace_id`, `span_id`, `lane_id`, `kiosk_id`, `tenant_id`
- Automatic redaction of PANs, biometric hashes, and full card numbers
- Levels: ERROR (alert), WARN (dashboard), INFO (routine), DEBUG (dev only)

### Alerting

| Priority | Condition                            | Action          |
| -------- | ------------------------------------ | --------------- |
| P0       | Payment failure rate > 5%            | Page on-call    |
| P1       | Sync partition detected (mesh split) | Page on-call    |
| P2       | Single kiosk offline > 10 minutes    | Email only      |
| P3       | Offline queue depth > 100            | SLA risk ticket |

---

## Deep Improvements Summary

Astra-Service includes the following advanced capabilities beyond a standard checkout system:

### 1. Ghost Cart

Customers can scan items on their phone via a WebRTC data channel, creating a "Ghost Cart" that floats in the P2P mesh. NFC bump transfers the cart to a physical kiosk, where LWW-CRDT merge resolves concurrent phone + kiosk edits.

### 2. Produce Recognition

A Rust + ONNX Runtime computer vision module uses the kiosk camera to identify produce and suggest PLU codes, removing the need for memorized lookup codes.

### 3. Lane Intelligence

An edge-deployed ONNX Runtime model (YOLOv8n) estimates queue length from camera feeds and dynamically switches the UI between express mode (fewer options, faster) and full mode.

### 4. Silent Assist

If a customer stalls for more than 45 seconds, the UI subtly highlights the next action with a pulsing primary button, respecting `prefers-reduced-motion`.

### 5. Transactional Outbox

Guarantees exactly-once event publication from cloud services, handling crashes between database commit and NATS publish.

### 6. WebAuthn / Passkeys

Employee authentication uses FIDO2/WebAuthn with biometric verification on the Verifone PIN pad. No passwords are stored or transmitted.

### 7. Circuit Breaker Dashboard

A real-time admin UI visualizes kiosk health, mesh topology, and payment lane status with a D3.js force-directed graph.

### 8. Customer-Facing Transparency

A "Why this price?" panel on every item and cart total shows item price, tax breakdown, loyalty discount, and environmental fees to build trust and reduce abandonment.

### 9. Dark Store Mode

The same codebase powers customer-facing kiosks (9:16) and employee handheld devices (16:9), with responsive layouts and role-specific workflows.

### 10. Differential Privacy

Aggregated analytics add Laplace noise to metrics, ensuring GDPR/CCPA compliance while still producing useful trend data.

### 11. Strangler Fig Pattern

For stores with a legacy POS, Astra-Service proxies requests through a REST adapter and message bridge, allowing gradual migration without a big-bang cutover.

### 12. Chaos Engineering

The CI pipeline injects random network partitions during integration tests to verify offline resilience, CRDT convergence, Raft leader election, and payment queueing.

---

## Appendices

### Appendix A: API Versioning

- URL-based versioning: `/v1/`, `/v2/`
- Deprecation headers: `Deprecation: true`, `Sunset: <date>`
- Minimum 6 months notice before API removal
- Version negotiation: clients send `Accept-Version`; server responds with active version

### Appendix B: Data Retention

| Data Class         | Retention                                         |
| ------------------ | ------------------------------------------------- |
| Transaction data   | 7 years (PCI/financial compliance)                |
| Cart/session data  | 90 days, then cold storage                        |
| Analytics events   | 30 days; aggregated metrics retained indefinitely |
| Audit logs         | 7 years, immutable, append-only                   |
| PII (email, phone) | Delete within 30 days of customer request         |

### Appendix C: Disaster Recovery

| Target | Value                                                                  |
| ------ | ---------------------------------------------------------------------- |
| RPO    | 5 minutes (NATS JetStream replication)                                 |
| RTO    | 15 minutes (automated failover to read replica)                        |
| Kiosk  | 48 hours offline operation                                             |
| Backup | PostgreSQL daily snapshots + WAL archiving to S3                       |
| DR     | Active-passive replication to secondary region; manual region failover |

### Appendix D: Development Environment

A Nix flake at `flake.nix` provides a reproducible shell with Node 22, Go 1.25, Rust 1.75, PostgreSQL 16, Redis 7, NATS, and Docker. Enter it with:

```bash
nix develop
```

Run the full local stack with:

```bash
docker compose up -d
pnpm dev          # kiosk-shell hot reload
go run ./...      # Go services
cargo run         # Rust sync daemon
```
