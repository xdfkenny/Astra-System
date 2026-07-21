# Development Setup

## Prerequisites

| Tool | Version | Purpose |
|------|---------|---------|
| Node.js | 22.x | TypeScript apps |
| pnpm | 9.12.0+ | Package manager |
| Go | 1.25+ | Microservices |
| Rust | 1.82+ | Sync daemon, sidecars |
| Python | 3.12-3.13 | ML lane intel |
| Docker | Latest | Containers |
| Docker Compose | Latest | Local stack |
| Nix | Latest | (Optional) reproducible shell |
| protoc + buf | Latest | Protobuf generation |

## Quick Start

```bash
# 1. Clone the repository
git clone https://github.com/MOTHER/Astra-System
cd Astra-System

# 2. Install frontend dependencies
pnpm install

# 3. Start infrastructure services
docker compose up -d

# 4. Start development servers
pnpm dev

# 5. (Optional) Nix development shell
nix develop
```

## Environment Configuration

Copy the example environment file:

```bash
cp .env.example .env
```

Key variables to configure:

| Variable | Default | Description |
|----------|---------|-------------|
| `POSTGRES_PASSWORD` | - | Database password |
| `REDIS_PASSWORD` | - | Redis password |
| `NATS_URL` | nats://localhost:4222 | NATS connection |
| `GATEWAY_JWT_SIGNING_KEY` | - | JWT signing key |
| `KIOSK_MESH_PSK` | - | P2P mesh pre-shared key |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | http://localhost:4317 | OpenTelemetry endpoint |

See [Environment Variables](../references/env-vars.md) for the complete list.

## IDE Setup

### VS Code Extensions

- Biome (linting)
- ESLint
- Prettier
- Go (gopls)
- rust-analyzer
- Tailwind CSS IntelliSense
- Docker
- YAML

### EditorConfig

The project includes `.editorconfig`:
- UTF-8, LF line endings
- 2-space indent (TypeScript, YAML)
- 4-space indent (Go, Rust)
- 100-character line width (Prettier)

## Verification

```bash
# Verify all tools work
node --version  # 22.x
pnpm --version  # 9.12.0+
go version      # 1.25+
rustc --version # 1.82+
docker compose version

# Verify the dev stack
pnpm lint
pnpm typecheck
pnpm test
```
