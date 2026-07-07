# Astra-Service

Production-grade, offline-first automated self-checkout platform for 24/7 retail environments.

[![CI](https://github.com/astra-service/astra-system/actions/workflows/ci.yml/badge.svg)](https://github.com/astra-service/astra-system/actions/workflows/ci.yml)

## Overview

Astra-Service powers unattended and attended self-checkout kiosks in retail stores. It is designed to keep selling even when the internet is unavailable for up to 48 hours, using a local peer-to-peer mesh, CRDT-based state replication, and offline payment token queueing.

### Key Features

- **Offline-first:** 48 hours of autonomous kiosk operation.
- **P2P mesh:** libp2p + QUIC + Noise for secure in-store sync.
- **CRDT consensus:** PN-Counters, LWW-Registers, and OR-Sets with Hybrid Logical Clocks.
- **Zero-trust security:** mTLS, HMAC request signing, HashiCorp Vault, and OS keychain integration.
- **Payment bridge:** Safe Rust FFI bindings for Verifone payment terminals.
- **Observability:** OpenTelemetry, Prometheus, Grafana, Loki, and Jaeger across all languages.
- **Auto-updates:** Signed OTA manifests and rollback-on-failure kiosk updater.

## Repository Layout

```text
Astra-System/
├── astra-service/          # TypeScript / React / Go / Rust monorepo
│   ├── apps/               # Kiosk shell, admin dashboard
│   ├── packages/           # Shared libraries (go-common, shared-types, ...)
│   ├── services/           # Go microservices
│   └── sync-daemon/        # Rust P2P sync daemon
├── database/               # Migrations and schemas
├── go/                     # Additional Go modules
├── infra/                  # TLS, secrets, Docker security profiles
├── proto/                  # Protocol Buffers and generated code
├── docs/                   # Architecture and operational runbooks
├── docker-compose.yml      # Local development stack
├── flake.nix               # Reproducible Nix development shell
└── ARCHITECTURE.md         # Comprehensive system design
```

## Quick Start

### Prerequisites

Choose one of the following:

- **Nix (recommended):** `nix develop` provides Node 22, Go 1.22, Rust 1.79, PostgreSQL 16, Redis 7, NATS, and Docker.
- **Manual:** Node 22+, pnpm 9+, Go 1.22+, Rust 1.79+, Docker, PostgreSQL 16, Redis 7, NATS.

### 1. Enter the Development Shell

```bash
nix develop
```

### 2. Start the Local Infrastructure

```bash
docker compose up -d
```

This brings up PostgreSQL, Redis, NATS JetStream, Vault, Prometheus, Grafana, and all Go microservices.

### 3. Install Node Dependencies

```bash
cd astra-service
pnpm install
```

### 4. Run the Kiosk Shell

```bash
pnpm dev
```

The kiosk simulator is available at `http://localhost:5170`.

### 5. Run the Rust Sync Daemon

```bash
cd astra-service/sync-daemon
cargo run
```

## Validation Commands

### Nix

```bash
nix flake check
nix develop --command go version
nix develop --command rustc --version
```

### TypeScript

```bash
cd astra-service
pnpm typecheck
pnpm test
pnpm lint
pnpm format:check
```

### Go

```bash
cd astra-service/services
go test -race ./...
```

### Rust

```bash
cd astra-service/sync-daemon
cargo test
cargo clippy -- -D warnings
```

### Docker Compose

```bash
docker compose config
docker compose up -d
docker compose ps
```

## Architecture

See [ARCHITECTURE.md](./ARCHITECTURE.md) for the full design covering:

- System overview
- Offline-first strategy
- P2P mesh and Raft consensus
- CRDTs and Hybrid Logical Clocks
- Event sourcing and transactional outbox
- Security model (zero trust, mTLS, secrets)
- Payment flow (Verifone, offline tokens)
- Deployment and CI/CD
- Observability
- Deep improvements summary

## Operational Runbooks

- [Incident Response](./docs/runbooks/incident-response.md)
- [Payment Failure](./docs/runbooks/payment-failure-runbook.md)
- [P2P Partition Recovery](./docs/runbooks/p2p-partition-recovery.md)
- [Offline Mode Operations](./docs/runbooks/offline-mode-operations.md)

## Security

Astra-Service follows a zero-trust security model. See the [Security Model](./ARCHITECTURE.md#security-model) section of ARCHITECTURE.md for details on mTLS, secrets management, and PCI-DSS compliance.

## Contributing

1. Create a feature branch from `main`.
2. Install hooks: `pnpm prepare` (Lefthook).
3. Make your changes and add tests.
4. Run the validation commands above.
5. Open a pull request.

All commits must follow [Conventional Commits](https://www.conventionalcommits.org/) and pass CI.

## License

Proprietary — Astra-Service Engineering Team.
