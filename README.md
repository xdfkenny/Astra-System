# Astra-System

Monorepo for Astra self-checkout components and operational artifacts.

This repository contains the source, protobufs, and operational files for Astra components that support unattended and attended kiosk deployments.

## What is actually in this repository (top-level)

- .changeset                     — changelog fragments for releases
- .commitlintrc.json             — commitlint configuration
- .editorconfig                  — editor configuration
- .env.example                   — example environment variables
- .github                        — GitHub workflows and community files
- .gitignore
- .husky                         — Git hooks
- .prettierignore
- .prettierrc
- .spectral.json                 — API/linting config
- AGENTS.md                      — agent-related notes
- ARCHITECTURE.md                — architectural design and diagrams
- astra-service/                 — service and app code (contains a Rust sync daemon)
  - sync-daemon/                 — astra-syncd (Rust) with its own README
- database/                      — database migrations and schema (DB artifacts)
- docker-compose.yml             — local development compose file
- docker-compose.prod.yml        — production compose manifest
- docs/                          — runbooks and operational documentation
- flake.nix                      — Nix flake for reproducible dev shell
- infra/                         — infrastructure tooling and TLS/secret helpers
- lefthook.yml                   — lefthook configuration for repo hooks
- packages/                      — shared packages/libraries
- proto/                         — .proto schemas and go-generation layout (has its own README)
- services/                      — service implementations (Go modules and related code)

## Notable subprojects

- proto/
  - Contains Protocol Buffer definitions and generated-code layout.
  - See proto/README.md for instructions to regenerate code with buf or protoc and for the Go module details.

- astra-service/sync-daemon/
  - `astra-syncd` is a Rust-based peer-to-peer sync daemon. See astra-service/sync-daemon/README.md for build, configuration, and runtime instructions.

- docs/
  - Operational runbooks and run-time guides are stored under docs/. Refer to those files for incident response and operational procedures.

## Quick, conservative build/run notes (what you can run with what's present)

- Generate protobuf code (see proto/README.md):

  ```bash
  cd proto
  buf generate    # or use protoc as documented in proto/README.md
  ```

- Build the Rust sync daemon:

  ```bash
  cd astra-service/sync-daemon
  cargo build --release
  ```

- Start local services via Docker Compose (where applicable):

  ```bash
  docker compose up -d
  ```

This README intentionally focuses on what the repository currently contains. For detailed developer workflows, CI, and the full project vision, consult ARCHITECTURE.md and the READMEs under proto/ and astra-service/sync-daemon/.

## If you want changes

Tell me whether you want:
- a shorter README (project index only),
- more detailed developer setup steps (Node/Go/Rust toolchains, pnpm commands), or
- an annotated tree with links to important files — and I will update the README accordingly.
