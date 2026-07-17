# Astra-Service Container Infrastructure

This directory contains the hardened container images and security profiles for the Astra-Service platform.

## Dockerfiles

All production Dockerfiles are multi-stage builds defined in this directory and are designed to be built from the repository root.

| Dockerfile | Service | Runtime | Notes |
|------------|---------|---------|-------|
| `Dockerfile.gateway` | API Gateway (Go/Fiber) | `gcr.io/distroless/static-debian12:nonroot` | Entrypoint `/usr/local/bin/gateway`, port 8080 |
| `Dockerfile.menu-service` | Menu Service (Go) | `gcr.io/distroless/static-debian12:nonroot` | Port 8085 |
| `Dockerfile.cart-service` | Cart Service (Go) | `gcr.io/distroless/static-debian12:nonroot` | Port 8081 |
| `Dockerfile.order-service` | Order Service (Go) | `gcr.io/distroless/static-debian12:nonroot` | Port 8083 |
| `Dockerfile.inventory-service` | Inventory Service (Go) | `gcr.io/distroless/static-debian12:nonroot` | Port 8082 |
| `Dockerfile.payment-orchestrator` | Payment Orchestrator (Go) | `gcr.io/distroless/static-debian12:nonroot` | Port 8086 |
| `Dockerfile.sync-service` | Sync Service (Go) | `gcr.io/distroless/static-debian12:nonroot` | Port 8087 |
| `Dockerfile.update-server` | Update Server (Go) | `gcr.io/distroless/static-debian12:nonroot` | Port 8080 |
| `Dockerfile.admin-graphql` | Admin GraphQL (Go) | `gcr.io/distroless/static-debian12:nonroot` | GraphQL admin API |
| `Dockerfile.webauthn-service` | WebAuthn Service (Go) | `gcr.io/distroless/static-debian12:nonroot` | FIDO2/WebAuthn authentication |
| `Dockerfile.ml-lane-intel` | ML Lane Intelligence (Python/FastAPI) | `python:3.13-slim` | Non-root `astra` user, port 8088 |
| `Dockerfile.kiosk` | Unified Kiosk (React + Vite) | `nginx:1.27-alpine` | Node 22 build stage, nginx serves static bundle |
| `Dockerfile.syncd` | P2P Sync Daemon (Rust) | `gcr.io/distroless/cc-debian12:nonroot` | Static Rust build, port 4499 |

All images run as a non-root user, drop all capabilities, and use read-only root filesystems where applicable.

## Security profiles

- `seccomp.json` — restrictive seccomp allowlist suitable for Go, Rust, Python, and nginx workloads.
- `apparmor.kiosk` — AppArmor profile for the nginx-based kiosk container.
- `apparmor.syncd` — AppArmor profile for the Rust sync daemon.
- `apparmor/astra-go-service` — AppArmor profile for Go microservices.
- `apparmor/astra-rust-syncd` — legacy AppArmor profile for the sync daemon.

## Local development

Start the local development stack with hot reload:

```bash
docker compose -f docker-compose.yml up -d
```

This brings up PostgreSQL 16, Redis 7, NATS JetStream, Jaeger, and the OpenTelemetry collector. The Go services use `air` for file-watching rebuilds, the kiosk uses Vite HMR, and the Rust sync daemon uses `cargo-watch`. Optional services are behind Compose profiles:

```bash
# ML lane intelligence + sync daemon
docker compose -f docker-compose.yml --profile ml --profile sync up -d
```

## Production deployment

The production compose file uses the hardened Dockerfiles with read-only root filesystems, seccomp/AppArmor references, replica counts, and resource limits:

```bash
docker compose -f docker-compose.prod.yml up -d
```

> `deploy.replicas` and `deploy.resources` are intended for orchestrators that support the Compose specification deployment block (e.g., Docker Swarm). On a plain Docker Compose installation, replicas are honored only in Swarm mode.

## Kubernetes

Manifests are provided under `infra/k8s/`. Apply them in order:

```bash
kubectl apply -f infra/k8s/namespace.yaml
kubectl apply -f infra/k8s/configmap.yaml
# Populate infra/k8s/secret-template.yaml with real values before applying.
kubectl apply -f infra/k8s/secret-template.yaml
kubectl apply -f infra/k8s/
```

The manifests include a `NetworkPolicy` default-deny ingress with explicit allow rules for the gateway, service mesh, and data stores.

## TLS / mTLS

Run `infra/tls/generate-certs.sh` to generate a local PKI using `cfssl` (with an `openssl` fallback). It produces server certificates for every service, a service-to-service client certificate, and a kiosk device certificate.

```bash
./infra/tls/generate-certs.sh
```

All generated artifacts are written to `infra/tls/out/` and must not be committed.
