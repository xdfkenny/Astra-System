# Docker Setup

## Overview

Containerization strategy using **multi-stage builds** with **distroless** runtime images, **seccomp** security profiles, and **AppArmor** for kiosk containers.

## Docker Compose Stacks

### Development (`docker-compose.yml`)

**426 lines** defining the full local dev stack with hot-reload:

| Service | Image | Port | Notes |
|---------|-------|------|-------|
| `postgres` | 16-alpine | 5432 | PG user `astra`, DB `astra_service` |
| `redis` | 7-alpine | 6379 | Appendonly, 256MB max, allkeys-lru |
| `nats` | 2-alpine | 4222 | JetStream enabled, 512MB mem, 2GB file |
| `jaeger` | all-in-one 1.60 | 16686 | OTLP gRPC+HTTP enabled |
| `otel-collector` | contrib 0.111 | 4317 | traces→Jaeger, metrics→Prometheus |
| `gateway` | Go 1.25 air | 8080 | Hot reload via air |
| `menu-service` | Go 1.25 air | 8085 | Hot reload |
| `cart-service` | Go 1.25 air | 8081 | Hot reload |
| `order-service` | Go 1.25 air | 8083 | Hot reload |
| `inventory-service` | Go 1.25 air | 8082 | Hot reload |
| `payment-orchestrator` | Go 1.25 air | 8086 | Hot reload |
| `sync-service` | Go 1.25 air | 8087 | Hot reload |
| `ml-lane-intel` | Python 3.13 | 8088 | Uvicorn with --reload |
| `kiosk` | Node 22 | 5180 | Vite HMR dev server |
| `syncd` | Rust 1.82 | 4499 | cargo-watch |

### Production (`docker-compose.prod.yml`)

**482 lines** with hardened security:

- Read-only root filesystem
- AppArmor profiles (kiosk, syncd)
- seccomp JSON profile
- Dropped capabilities
- Resource limits
- Health checks
- Replica scaling
- Non-root users

## Dockerfiles (16 total)

Located in `infra/docker/`:

| Dockerfile | Base Image | Target |
|------------|------------|--------|
| `Dockerfile.gateway` | golang:1.25 → distroless | API Gateway |
| `Dockerfile.menu-service` | golang:1.25 → distroless | Menu Service |
| `Dockerfile.cart-service` | golang:1.25 → distroless | Cart Service |
| `Dockerfile.order-service` | golang:1.25 → distroless | Order Service |
| `Dockerfile.inventory-service` | golang:1.25 → distroless | Inventory Service |
| `Dockerfile.payment-orchestrator` | golang:1.25 → distroless | Payment Orchestrator |
| `Dockerfile.sync-service` | golang:1.25 → distroless | Sync Service |
| `Dockerfile.webauthn-service` | golang:1.25 → distroless | WebAuthn Service |
| `Dockerfile.admin-graphql` | golang:1.25 → distroless | Admin GraphQL |
| `Dockerfile.update-server` | golang:1.25 → distroless | Update Server |
| `Dockerfile.kiosk` | node:22 → nginx:alpine | Kiosk SPA |
| `Dockerfile.kiosk-unified` | node:22 → nginx:alpine | Unified kiosk image |
| `Dockerfile.admin` | node:22 → nginx:alpine | Admin dashboard |
| `Dockerfile.ml-lane-intel` | python:3.13-slim | ML service |
| `Dockerfile.syncd` | rust:1.82 → distroless | Sync Daemon |

## Build Characteristics

- **Multi-platform:** linux/amd64 + linux/arm64 via Buildx
- **GHA cache:** Scoped per platform
- **Distroless runtime:** Minimize attack surface
- **Cosign signing:** All images signed after build
- **Syft SBOM:** SPDX format, attested via cosign
- **Trivy scanning:** Vulnerability scan before push

## Security Profiles

- **seccomp.json** - Restricted syscall whitelist
- **apparmor-kiosk** - AppArmor profile for kiosk container
- **apparmor-syncd** - AppArmor profile for sync daemon
