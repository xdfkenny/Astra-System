# CI/CD Pipeline

## Overview

Two GitHub Actions workflows manage continuous integration and deployment: a main CI pipeline (981 lines) and an installer build pipeline (328 lines).

## CI Pipeline

**File:** `.github/workflows/ci.yml`

### Pipeline Stages

```
                  ┌──────────┐
                  │  changes │ (path filtering)
                  └─────┬────┘
                        │
        ┌───────────────┼───────────────────┐
        │               │                    │
   ┌────┴────┐   ┌─────┴──────┐   ┌────────┴────────┐
   │ lint-ts │   │ lint-go    │   │  lint-rust      │
   │ Biome   │   │ gofmt+govet│   │  clippy+fmt     │
   │ ESLint  │   │ golangci   │   └────────┬────────┘
   │ tsc     │   └─────┬──────┘            │
   └────┬────┘         │                    │
        │              │                    │
   ┌────┴──────────────┴────────────────────┴────┐
   │              test-unit (matrix)              │
   │  TS (vitest) │ Go (go test -race) │ Rust     │
   └──────────────────────┬───────────────────────┘
                          │
        ┌─────────────────┼─────────────────────┐
        │                 │                      │
   ┌────┴──────┐   ┌──────┴──────┐   ┌──────────┴──────────┐
   │test-integ │   │ test-python │   │    test-e2e         │
   │(Go smoke) │   │ ruff+mypy  │   │  (Playwright)       │
   └────┬──────┘   │ pytest      │   └──────────┬──────────┘
        │          └──────┬──────┘              │
        │                 │                      │
   ┌────┴─────────────────┴──────────────────────┴──────┐
   │              security-audit                         │
   │  cargo audit │ govulncheck │ Trivy │ Gitleaks      │
   └──────────────────────┬─────────────────────────────┘
                          │
                   ┌──────┴──────┐
                   │ iac-validate│
                   │ tf+helm+k8s │
                   └──────┬──────┘
                          │
                   ┌──────┴──────────────────────────┐
                   │         build-docker             │
                   │  16 images × 2 platforms (32 jobs)│
                   │  Multi-stage + distroless         │
                   └──────┬──────────────────────────┘
                          │
                   ┌──────┴──────┐
                   │ sbom-generate │
                   │ Syft SPDX    │
                   └──────┬──────┘
                          │
                   ┌──────┴──────────────────┐
                   │     merge-manifests      │
                   │  multi-arch manifests    │
                   └──────┬──────────────────┘
                          │
                   ┌──────┴──────┐
                   │ scan-images │
                   │  Trivy      │
                   └──────┬──────┘
                          │
                   ┌──────┴──────┐
                   │   release   │
                   │ changesets  │
                   └─────────────┘
```

### Key Details

- **Matrix builds:** 16 Docker images × 2 platforms (linux/amd64, linux/arm64) = 32 parallel build jobs
- **Path filtering:** Only relevant jobs run based on changed files
- **Cache:** Docker layers cached via GHA cache scoped per platform
- **Supply chain:** All images cosign-signed with Syft SBOM attestation
- **Vulnerability scanning:** Trivy SARIF output uploaded to GitHub

## Installer Pipeline

**File:** `.github/workflows/build-installer.yml`

### Stages

1. **Cross-compile Go installer** for windows/darwin/linux × amd64/arm64
2. **Build Windows Inno Setup installer** (`installer/setup.iss`)
3. **Create GitHub release** with all artifacts (binaries + installer)

## Development Commands (Turborepo)

| Command | Description |
|---------|-------------|
| `pnpm build` | Build all packages |
| `pnpm dev` | Start all dev servers |
| `pnpm lint` | Lint all code |
| `pnpm test` | Run all unit tests |
| `pnpm test:e2e` | Run Playwright E2E tests |
| `pnpm typecheck` | TypeScript type checking |
| `pnpm clean` | Clean all build artifacts |
| `pnpm changeset` | Create a changeset |
| `pnpm release` | Version and publish |

## Local Development

```bash
# Start infrastructure (PostgreSQL, Redis, NATS, Jaeger)
docker compose up -d postgres redis nats jaeger

# Start all services in dev mode
pnpm dev
```
