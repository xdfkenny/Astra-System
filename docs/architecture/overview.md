# System Overview

## Purpose

Astra-System is a **production-grade, offline-first, automated self-checkout platform** engineered for 24/7 unattended retail environments. It powers self-checkout kiosks with 48 hours of offline resilience, a zero-trust security model, and peer-to-peer mesh synchronization.

## Core Design Tenets

1. **Offline-First by Default** - Every feature works without network connectivity. Cloud is a sync target, not a dependency.
2. **Zero-Trust Security** - mTLS between all services, SPIFFE identities, no implicit trust within the mesh.
3. **Eventual Consistency with CRDTs** - Conflict-free Replicated Data Types enable safe concurrent edits across kiosks.
4. **Edge-to-Cloud Continuum** - Seamless transition between offline, P2P mesh, and cloud-connected modes.
5. **PCI-DSS Compliance** - Card data never enters kiosk memory; payment flows through dedicated Verifone terminals.

## System Boundaries

```
┌─────────────────────────────────────────────────────────┐
│                    Cloud Infrastructure                   │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌─────────┐ │
│  │ Postgres │  │  Redis   │  │  NATS    │  │  Vault  │ │
│  │   (HA)   │  │  (Cache) │  │ (Event)  │  │(Secrets)│ │
│  └────┬─────┘  └────┬─────┘  └────┬─────┘  └────┬────┘ │
│       │              │             │              │       │
│  ┌────┴──────────────┴─────────────┴──────────────┴────┐ │
│  │               Go Microservices (13)                  │ │
│  │  Gateway │ Menu │ Cart │ Order │ Inventory │ Payment │ │
│  │  Sync │ WebAuthn │ Admin │ ML │ Legacy-POS          │ │
│  └────────────────────────┬────────────────────────────┘ │
│                           │                               │
│  ┌────────────────────────┴────────────────────────────┐ │
│  │             Cloud Sync Gateway (gRPC)                │ │
│  └────────────────────────┬────────────────────────────┘ │
└───────────────────────────┼─────────────────────────────┘
                            │ Internet (TLS)
┌───────────────────────────┼─────────────────────────────┐
│                     Store Network                        │
│  ┌────────────────────────┴────────────────────────────┐ │
│  │            P2P Mesh (libp2p/QUIC)                   │ │
│  │  ┌─────────┐  ┌─────────┐  ┌─────────┐             │ │
│  │  │ Kiosk 1 │──│ Kiosk 2 │──│ Kiosk 3 │  (Raft)     │ │
│  │  └────┬────┘  └────┬────┘  └────┬────┘             │ │
│  │       │             │             │                   │ │
│  │  ┌────┴─────────────┴─────────────┴────┐            │ │
│  │  │      Local Sync Daemon (syncd)      │            │ │
│  │  │  SQLCipher │ CRDT Merge │ Raft      │            │ │
│  │  └────────────────────────────────────-┘            │ │
│  └─────────────────────────────────────────────────────┘ │
│                                                          │
│  ┌──────────────────────────────────────────────────────┐│
│  │              Kiosk (React 19 + Vite)                 ││
│  │  ┌──────────┐ ┌──────┐ ┌────────┐ ┌───────────┐    ││
│  │  │  Shell    │ │ Menu │ │  Cart  │ │  Payment   │    ││
│  │  │  (Host)   │ │(MFE) │ │ (MFE)  │ │  (MFE)     │    ││
│  │  └──────────┘ └──────┘ └────────┘ └───────────┘    ││
│  │  ┌──────────┐ ┌──────┐ ┌────────┐                  ││
│  │  │  XState  │ │Valtio│ │ Zustand│                  ││
│  │  │ (Machine)│ │(Cart)│ │(Session)│                  ││
│  │  └──────────┘ └──────┘ └────────┘                  ││
│  │  ┌──────────────────────────────────────────────┐   ││
│  │  │  Payment Sidecar (Rust)  │ Verifone Terminal │   ││
│  │  └──────────────────────────────────────────────┘   ││
│  └──────────────────────────────────────────────────────┘│
└─────────────────────────────────────────────────────────┘
```

## Tech Stack Summary

| Layer | Technology | Purpose |
|-------|------------|---------|
| Frontend | React 19, Vite, Tailwind CSS | Kiosk UIs, admin dashboard |
| State | XState v5, Zustand, Valtio, TanStack Query | Workflow machines, session, cart, server cache |
| Frontend Composition | Module Federation | Micro-frontend architecture |
| Backend | Go 1.25, Fiber, gRPC, gRPC-Gateway | 13 microservices |
| Database | PostgreSQL 16, SQLCipher, Drizzle ORM | Primary DB, encrypted offline storage |
| Event Bus | NATS JetStream | Async messaging, outbox relay |
| Cache | Redis 7 | Session caching, rate limiting |
| P2P Sync | libp2p, QUIC, Noise, Raft, CRDTs | Mesh synchronization |
| ML | Python 3.13, YOLOv8n, ONNX | Lane queue intelligence |
| Payments | Verifone (Rust FFI bridge) | Terminal integration |
| Infrastructure | Docker, K8s, Helm, Terraform | Containerization, orchestration, IaC |
| Observability | OpenTelemetry, Prometheus, Grafana, Loki, Jaeger | Traces, metrics, logs |
| CI/CD | GitHub Actions, Turborepo, Changesets | Build, test, release pipeline |
| Security | mTLS, SPIFFE, WebAuthn/FIDO2, Cosign, Trivy | Zero-trust, supply chain security |

## Key Design Decisions

| Decision | Rationale |
|----------|-----------|
| Multi-language monorepo (TS + Go + Rust + Python) | Right tool for each job: TS for UI, Go for services, Rust for edge/P2P, Python for ML |
| CRDTs over consensus for cart data | Allows concurrent edits across offline kiosks without coordination |
| Raft only for mesh leadership (3+ kiosks) | Leader election needed for coordinated sync; sub-3s failover |
| gRPC for inter-service + REST for external | Performance of gRPC internally, REST for browser/3rd-party consumption |
| Module Federation over iframes | Shared React context, atomic updates, better DX |
| SQLCipher for offline storage | Encryption at rest on untrusted kiosk hardware |
| Verifone payment sidecar (Rust) | Card data never in main process memory; isolated payment path |

## Version

Current version: **0.1.0** (Apache 2.0 licensed)
