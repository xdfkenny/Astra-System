# Astra-System

<p align="center">
  <img src="https://raw.githubusercontent.com/cat-milk/Anime-Girls-Holding-Programming-Books/master/Typescript/Beako_Reading_The_TypeScript_Programming_Language.png" width="420" alt="Anime girl reading the TypeScript programming language book" />
</p>

<p align="center">
  <a href="./README.md"><b>English</b></a> ·
  <a href="./docs/Readme Translations/README.es.md">Español</a> ·
  <a href="./docs/Readme Translations/README.zh.md">中文</a> ·
  <a href="./docs/Readme Translations/README.fr.md">Français</a>
  <br>
  <sub>
   <a href="./docs/Readme Translations/README.ja.md">日本語</a> ·
  <a href="./docs/Readme Translations/README.ko.md">한국어</a> ·
  <a href="./docs/Readme Translations/README.hi.md">हिन्दी</a> ·
  <a href="./docs/Readme Translations/README.ar.md">العربية</a> ·
  <a href="./docs/Readme Translations/README.pt.md">Português</a> ·
  <a href="./docs/Readme Translations/README.ru.md">Русский</a> ·
  <a href="./docs/Readme Translations/README.bn.md">বাংলা</a> ·
  <a href="./docs/Readme Translations/README.de.md">Deutsch</a> ·
  <a href="./docs/Readme Translations/README.ur.md">اردو</a> ·
  <a href="./docs/Readme Translations/README.tr.md">Türkçe</a> ·
  <a href="./docs/Readme Translations/README.zh-TW.md">繁體中文</a> ·
  <a href="./docs/Readme Translations/README.vi.md">Tiếng Việt</a> ·
  <a href="./docs/Readme Translations/README.th.md">ไทย</a> ·
  <a href="./docs/Readme Translations/README.la.md">Latina</a>
  </sub>
</p>

[![CI](https://img.shields.io/badge/CI-pass-green.svg)](https://github.com/anomalyco/astra-system/actions)
[![License](https://img.shields.io/badge/license-Apache%202.0-blue.svg)](LICENSE)
[![Go](https://img.shields.io/badge/Go-1.25-00ADD8.svg)](https://go.dev)
[![Rust](https://img.shields.io/badge/Rust-1.82-dea584.svg)](https://www.rust-lang.org)
[![TypeScript](https://img.shields.io/badge/TypeScript-5.x-3178C6.svg)](https://www.typescriptlang.org)

> Production-grade, offline-first automated self-checkout platform engineered for 24/7 retail environments.

**Astra-System** is a multi-language monorepo that powers unattended and attended self-checkout kiosks. It delivers zero-downtime store operation with **48 hours of offline resilience**, a zero-trust security model, and a peer-to-peer mesh sync layer that keeps every kiosk in a store consistent — even when the cloud is unreachable.

---

## Table of Contents

- [Overview](#overview)
- [Key Features](#key-features)
- [Architecture](#architecture)
- [Technology Stack](#technology-stack)
- [Repository Layout](#repository-layout)
- [Getting Started](#getting-started)
- [Development Workflow](#development-workflow)
- [Build & Run](#build--run)
- [Testing](#testing)
- [Documentation](#documentation)
- [Contributing](#contributing)
- [License](#license)

---

## Overview

Astra-System enables retailers to deploy fleets of self-checkout kiosks that operate **autonomously for up to 48 hours** without internet connectivity. Resilience is layered across three tiers:

1. **Local data layer** — an encrypted SQLite (SQLCipher) store on every kiosk with the full menu catalog, inventory, pending transactions, and offline payment tokens.
2. **Peer-to-peer mesh** — kiosks discover one another over the local network (mDNS + libp2p/QUIC) and replicate state using CRDTs, electing a Raft leader when three or more are present.
3. **Graceful degradation** — payments, inventory, and order capture continue locally and reconcile with the cloud when connectivity returns.

The cloud tier (Go microservices, PostgreSQL 16, Redis 7, NATS JetStream) provides the authoritative event-sourced store, settlement, and fleet management.

### Design Goals

| Goal               | Target                                                                 |
| ------------------ | ---------------------------------------------------------------------- |
| Offline resilience | 48 hours of autonomous operation with no cloud connectivity            |
| Latency            | < 200 ms menu load, < 500 ms P2P inventory sync, < 3 s leader failover |
| Availability       | 99.99% uptime (cloud tier); 100% uptime during local-only mode         |
| Security           | Zero trust, mTLS everywhere, PCI-DSS compliant payment path            |
| Scale              | 1–10,000 kiosks per tenant; multi-region cloud deployment              |

---

## Key Features

- **Offline-first engine** — deterministic CRDT merge (PN-Counter, LWW-Register, OR-Set) with Hybrid Logical Clocks for causal ordering across kiosks.
- **P2P mesh & Raft consensus** — libp2p QUIC transport, Noise protocol encryption, and sub-3-second leader failover.
- **Transactional outbox** — exactly-once event publication from cloud services via NATS JetStream.
- **Zero-trust security** — mTLS, per-kiosk HMAC signing, SPIFFE identities, and a PCI-DSS compliant payment path (card data never touches kiosk memory).
- **Verifone FFI bridge** — a safe Rust wrapper (`astra-verifone-ffi`) over the vendor C SDK for payment terminal integration.
- **Biophilic kiosk UI** — a React 19 micro-frontend built with Module Federation, XState v5 workflow machine, and Zustand/TanStack Query state management.
- **Advanced intelligence** — Ghost Carts, produce recognition (ONNX), lane intelligence (TFLite), WebAuthn/passkeys, and differential-privacy analytics.
- **Chaos-ready CI** — network partitions are injected during integration tests to verify resilience, CRDT convergence, and payment queueing.
- **Multilingual kiosk UI** — customers select their preferred language at session start from 17+ supported languages (English, Spanish, Chinese, French, Japanese, Korean, Hindi, Arabic, Portuguese, Russian, Bengali, German, Urdu, Turkish, Traditional Chinese, Vietnamese, Thai, and more). All UI text, receipts, and audio prompts render in the selected locale.

---

## Architecture

Astra-System is split into a **Cloud Tier** and a **Store Edge / Kiosk Cluster**.

```text
┌─────────────────────────────────────────────────────────────────┐
│                         Cloud Tier                              │
│  API Gateway · Order Svc · Payment Svc · Inventory Svc ·       │
│  Cart Svc · Sync Svc · PostgreSQL 16 · Redis 7 · NATS JetStream │
└──────────────────────────────────┬──────────────────────────────┘
                                   │ TLS 1.3
┌──────────────────────────────────┴──────────────────────────────┐
│                    Store Edge / Kiosk Cluster                   │
│  Kiosk 1 ──┐   Kiosk 2 ──┐   Kiosk N ──┐                       │
│  React 19  │   React 19  │   React 19  │  (local mesh QUIC)    │
│  Rust P2P  │   Rust P2P  │   Rust P2P  │                       │
│  SQLite    │   SQLite    │   SQLite    │                       │
│  Verifone · Printer · Scanner · NFC/Scale                      │
└─────────────────────────────────────────────────────────────────┘
```

For the complete topology, security model, payment flows, observability, and disaster-recovery details, see [`ARCHITECTURE.md`](./ARCHITECTURE.md).

### Service Inventory

| Service           | Language   | Responsibility                                   |
| ----------------- | ---------- | ------------------------------------------------ |
| `api-gateway`     | Go         | Edge routing, authN/authZ, rate limiting         |
| `order-svc`       | Go         | Order lifecycle, cart persistence, fulfillment   |
| `payment-svc`     | Go         | Payment orchestration, token settlement          |
| `inventory-svc`   | Go         | Stock levels, soft holds, catalog sync           |
| `cart-svc`        | Go         | Cart CRDT merge, ghost-cart resolution           |
| `sync-svc`        | Go         | Cloud-side mesh gateway and batch ingestion      |
| `astra-syncd`     | Rust       | Kiosk P2P daemon, CRDT sync, Verifone FFI bridge |
| `kiosk-shell`     | TypeScript | React 19 customer UI, peripheral integration     |
| `update-server`   | Go         | Signed OTA manifest delivery                     |

---

## Technology Stack

- **Frontend** — TypeScript, React 19, Vite, Module Federation, XState v5, Zustand, TanStack Query, Tailwind CSS (v4 in apps, v3 in the design system).
- **Backend** — Go (Fiber / gRPC), PostgreSQL 16, Redis 7, NATS JetStream.
- **Edge** — Rust (`astra-syncd`, `astra-verifone-ffi`), SQLite (SQLCipher), libp2p.
- **ML** — ONNX Runtime, TensorFlow Lite.
- **Infra** — Kubernetes, Docker / Podman, Traefik, HashiCorp Vault, Nix flake.
- **Observability** — Prometheus, Grafana, Loki, Jaeger, OpenTelemetry.

---

## Repository Layout

```text
astra-service/          Service and application code
  apps/                 TypeScript micro-frontends (kiosk-shell, kiosk-menu, …)
  packages/             Shared libraries and design system
  services/             Go microservices
  sync-daemon/          astra-syncd (Rust) P2P daemon
  daemons/              Sidecar daemons (payment-sidecar)
  tools/                Operational tooling (chaos, etc.)
services/               Standalone services (update-server, …)
database/               Schema migrations
proto/                  Protocol Buffer definitions and generated code
docs/                   Operational runbooks
infra/                  Infrastructure tooling and secrets helpers
.github/                CI workflows and community files
flake.nix               Reproducible Nix dev shell
docker-compose*.yml     Local and production compose manifests
```

---

## Getting Started

### Prerequisites

- **Node.js 22** and **pnpm 9+**
- **Go 1.25**
- **Rust 1.82** (with `protoc` for building the sync daemon)
- **Docker** and **Docker Compose**
- *(Optional)* **Nix** for a fully reproducible toolchain:

  ```bash
  nix develop
  ```

### Quick Start

```bash
# 1. Install frontend dependencies
pnpm install

# 2. Bring up the local backend stack (PostgreSQL, Redis, NATS)
docker compose up -d

# 3. Run all TypeScript apps with hot reload
pnpm dev

# 4. Build the Rust sync daemon
cd astra-service/sync-daemon && cargo build --release
```

Copy `.env.example` to `.env` and adjust values as needed before running services.

---

## Development Workflow

```bash
# Lint, typecheck, and test (order matters)
pnpm lint
pnpm typecheck
pnpm test

# End-to-end tests (Playwright)
pnpm test:e2e

# Format
pnpm format && pnpm format:check

# Build all packages
pnpm build
```

Run a single package via Turborepo filters:

```bash
pnpm turbo run dev --filter=@astra/kiosk
pnpm turbo run test --filter=@astra/kiosk
```

### Go services

```bash
cd astra-service/services
go test -race ./...
go vet ./...
```

### Rust daemons

```bash
cd astra-service/sync-daemon
cargo test
cargo clippy -- -D warnings
cargo fmt --check
```

---

## Build & Run

### Protobuf generation

```bash
cd proto
buf generate        # or: protoc as documented in proto/README.md
```

### Local full stack

```bash
docker compose up -d
pnpm dev            # kiosk-shell hot reload
```

For production manifests, use `docker-compose.prod.yml`.

---

## Testing

| Layer        | Tooling                                            |
| ------------ | -------------------------------------------------- |
| Unit (TS)    | Vitest + happy-dom                                 |
| E2E (TS)     | Playwright against `kiosk-shell`                   |
| Unit (Go)    | `go test -race ./...`                              |
| Unit (Rust)  | `cargo test`, `cargo clippy`                       |
| Integration  | Docker Compose stack (PostgreSQL, Redis, NATS)     |
| Chaos        | Network-partition injection during integration     |

> Integration and chaos tests require Docker running with `postgres`, `redis`, and `nats` containers.

---

## Documentation

- [`ARCHITECTURE.md`](./ARCHITECTURE.md) — system design, security model, payment flows, observability, and DR.
- [`UX_UI_AUDIT_REPORT.md`](./astra-service/UX_UI_AUDIT_REPORT.md) — the "Living Weave" biophilic kiosk UI design specification.
- [`docs/Readme Translations/`](./docs/Readme Translations/) — community-contributed README translations in 17+ languages.
- `proto/README.md`, `astra-service/sync-daemon/README.md`, and `docs/` — subproject and operational runbooks.

---

## Contributing

1. Follow [Conventional Commits](https://www.conventionalcommits.org/) for all commit messages.
2. Run `pnpm prepare` to install Lefthook pre-commit hooks.
3. Ensure `lint → typecheck → test` all pass before opening a pull request.
4. Keep changes path-scoped; CI is path-filtered and only runs the relevant toolchains.

---

## License

Licensed under the [Apache License, Version 2.0](LICENSE).

---

<p align="center">
  <sub>Astra-System · Built for resilient, offline-first retail.</sub>
</p>
