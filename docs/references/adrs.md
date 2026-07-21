# Architecture Decision Records

## ADR List

| ID | Title | Status | Date |
|----|-------|--------|------|
| ADR-001 | Offline-First Architecture | Approved | 2025-11-10 |
| ADR-002 | CRDTs over Consensus for Cart State | Approved | 2025-11-15 |
| ADR-003 | Multi-Language Monorepo (Turborepo) | Approved | 2025-12-01 |
| ADR-004 | Zero-Trust Security Model | Approved | 2025-12-10 |
| ADR-005 | Micro-Frontends via Module Federation | Approved | 2026-01-05 |
| ADR-006 | NATS JetStream as Event Bus | Approved | 2026-01-15 |
| ADR-007 | gRPC for Inter-Service Communication | Approved | 2026-02-01 |
| ADR-008 | SQLCipher for Kiosk Offline Storage | Approved | 2026-02-10 |
| ADR-009 | Raft for P2P Consensus (3+ Kiosks) | Approved | 2026-03-01 |
| ADR-010 | Rust for Edge Daemons | Approved | 2026-03-10 |
| ADR-011 | XState v5 for Kiosk Workflow | Approved | 2026-04-01 |
| ADR-012 | Multi-Store State Management | Approved | 2026-04-15 |

---

## ADR-001: Offline-First Architecture

**Status:** Approved  
**Date:** 2025-11-10  
**Context:** Retail kiosks frequently lose internet connectivity. Traditional client-server models fail when the cloud is unreachable.  
**Decision:** Design the entire system to operate offline by default, with cloud as a sync target, not a dependency.  
**Consequences:** More complex data model (CRDTs), but 48-hour autonomous operation.

## ADR-002: CRDTs over Consensus for Cart State

**Status:** Approved  
**Date:** 2025-11-15  
**Context:** Carts are frequently modified on multiple kiosks (ghost cart transfer, mobile → kiosk). Consensus would block operations.  
**Decision:** Use CRDTs (PN-Counter, LWW-Register, OR-Set) with HLC timestamps for conflict-free merging.  
**Consequences:** Eventual consistency, no blocking, but requires careful CRDT type selection per data category.

## ADR-003: Multi-Language Monorepo (Turborepo)

**Status:** Approved  
**Date:** 2025-12-01  
**Context:** Different problems require different tools: UI (TypeScript), services (Go), edge (Rust), ML (Python).  
**Decision:** Organize as a single monorepo with Turborepo + pnpm workspaces for TypeScript, Go workspace for Go modules, Cargo workspace for Rust crates.  
**Consequences:** Single CI pipeline, shared configs, but larger clone size and more complex build graph.

## ADR-004: Zero-Trust Security Model

**Status:** Approved  
**Date:** 2025-12-10  
**Context:** Retail environments have untrusted networks. PCI-DSS requires strict payment data isolation.  
**Decision:** Zero-trust: mTLS everywhere, SPIFFE identities, per-service least privilege, HMAC request signing.  
**Consequences:** Higher operational complexity (certificate management), but meets PCI-DSS and prevents lateral movement.

## ADR-005: Micro-Frontends via Module Federation

**Status:** Approved  
**Date:** 2026-01-05  
**Context:** Multiple teams need to independently develop and deploy kiosk UI features.  
**Decision:** Use Webpack/Vite Module Federation with React 19 singleton for independent micro-frontends.  
**Consequences:** Independent deployments, shared component library, but increased bundle complexity.

## ADR-006: NATS JetStream as Event Bus

**Status:** Approved  
**Date:** 2026-01-15  
**Context:** Need reliable async communication between microservices with at-least-once delivery guarantees.  
**Decision:** NATS JetStream with transactional outbox pattern for exactly-once event publication.  
**Consequences:** Reliable event delivery, built-in persistence, but adds infrastructure dependency.

## ADR-007: gRPC for Inter-Service Communication

**Status:** Approved  
**Date:** 2026-02-01  
**Context:** Services need efficient, typed, streaming-capable communication.  
**Decision:** gRPC with Protocol Buffers for all inter-service calls; REST/gRPC-Gateway for external consumption.  
**Consequences:** Strong typing, efficient binary protocol, but needs proto management and code generation.

## ADR-008: SQLCipher for Kiosk Offline Storage

**Status:** Approved  
**Date:** 2026-02-10  
**Context:** Kiosks store sensitive data (payment tokens, employee credentials) on untrusted hardware.  
**Decision:** SQLCipher (AES-256-GCM encrypted SQLite) for the sync daemon's local database.  
**Consequences:** Data encrypted at rest, but slight performance overhead vs. plain SQLite.

## ADR-009: Raft for P2P Consensus (3+ Kiosks)

**Status:** Approved  
**Date:** 2026-03-01  
**Context:** When multiple kiosks are present, a coordinator is needed for cloud sync orchestration.  
**Decision:** Raft consensus for leader election when 3+ kiosks exist; no leader with 1-2 kiosks.  
**Consequences:** Reliable leader election under 3 seconds, but adds operational complexity.

## ADR-010: Rust for Edge Daemons

**Status:** Approved  
**Date:** 2026-03-10  
**Context:** Sync daemon and payment sidecar require memory safety, performance, and FFI with vendor SDKs.  
**Decision:** Rust for all edge daemons (sync-daemon, payment-sidecar, verifone-ffi).  
**Consequences:** Memory safety without GC, excellent C interop, but steeper learning curve.

## ADR-011: XState v5 for Kiosk Workflow

**Status:** Approved  
**Date:** 2026-04-01  
**Context:** Kiosk UI has complex state transitions with guards, actions, and async actors.  
**Decision:** Model kiosk workflow as a single XState v5 state machine with typed events and context.  
**Consequences:** Deterministic UI flow, easy tracing, but adds abstraction overhead.

## ADR-012: Multi-Store State Management

**Status:** Approved  
**Date:** 2026-04-15  
**Context:** Different state has different characteristics (ephemeral vs. persistent, local vs. server).  
**Decision:** Use XState for workflow, Zustand for session, Valtio for cart, TanStack Query for server cache.  
**Consequences:** Each concern uses the best tool, but developers must understand four state management libraries.
