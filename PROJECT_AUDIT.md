# Astra-Service Project Audit

**Generated:** 2026-07-09  
**Auditor:** Principal Software Engineer — Forensic Analysis  
**Scope:** Entire codebase: `Astra-System/` — TypeScript, Go, Rust, Python, Infrastructure

---

## Executive Summary

| Metric | Score |
|---|---|
| **Overall Health Score** | **5.5 / 10** |
| **TypeScript Frontend** | 6/10 — Design token system is excellent; micro-frontend architecture is ambitious but has critical integration gaps |
| **Go Backend** | 5/10 — Extensive service scaffolding but minimal actual business logic; many stub services |
| **Rust Sync Daemon** | 8/10 — Well-structured, compiles, has all subsystems wired; most complete component |
| **Infrastructure** | 7/10 — Comprehensive CI/CD, Docker, K8s configs; Dockerfiles referenced but not all exist |
| **Testing** | 4/10 — Some unit tests exist but coverage is very low; no integration tests visible |
| **Security Posture** | 5/10 — Hardcoded dev secrets in source code; design spec mentions Vault but not implemented |

### Critical Blockers

1. **P0: Kiosk host builds but screens are stubs with mock data** — No real API integration exists. The menu uses `mockMenuData.ts`, payment uses `dev-only` HMAC keys, and cart operations silently fall back to local state on every error.
2. **P0: Federated micro-frontends are NOT actually consumed** — The kiosk-shell app uses `src/routes/WorkflowRouter.tsx` with local screen implementations, NOT the `vite-plugin-federation` remotes. The remote type declarations exist but `WorkflowRouter.tsx` has its own `<MenuScreen />`, `<CartReviewScreen />`, etc. This means there are TWO competing implementations and neither talks to the other.
3. **P1: `finalizeOrder` in the XState machine is a mock** — It creates a fake order with `crypto.randomUUID()` and a hardcoded `800ms` delay. No real payment flow integration.
4. **P1: XState machine has a missing `RETURN_TO_ATTRACT` event handler** — `useIdleReclaim.ts` sends `{ type: "RETURN_TO_ATTRACT" }` but `kioskMachine.ts` has no handler for this event. The idle timeout silently throws at runtime.
5. **P1: ViewportLock uses fixed 1080×1920 logical pixels with CSS scale** — This approach breaks touch coordinate mapping on 1440×2560 panels because Framer Motion `tapPoint` coordinates are in CSS-viewport space, not the scaled logical space. Taps on premium lanes will drift by ~33%.
6. **P2: OfflineBanner component has a logic bug** — It checks `isOffline` to decide whether to render, but `showBanner` state is set with `setTimeout`; if `isOffline` changes during the timeout window, the banner state can become permanently inconsistent.

### Warning Flags

- **7 `@ts-expect-error` annotations** in core state files (apiClient.ts, cartService.ts) — suppressing real type errors
- **Hardcoded dev credentials** in `PaymentApp.tsx` line 191: `"dev-only-32-byte-minimum-secret-key!!"`
- **Two competing routing systems**: `react-router-dom` in package.json dependencies but only XState machine-based routing is used
- **`version: "0.1.0"`** across all packages — project is clearly in early prototyping phase
- **Dual state management**: XState v5, Zustand, AND Valtio all used for overlapping concerns
- **Missing Dockerfiles**: CI workflow references `infra/docker/Dockerfile.*` files, but `infra/docker/` directory only contains `seccomp.json`
- **No database migrations applied**: `database/migrations/` and `database/schemas/` directories exist but are empty

### Ready for Production? **NO** — This is a well-structured prototype with impressive architectural ambition but major functional gaps.

## Audit Update — 2026-07-14

This addendum preserves the prior 2026-07-09 assessment while capturing the current repository state. The latest review found a critical compile blocker, browser-side secret exposure, and prototype-quality workflow gaps that must be addressed before the project can be considered production-ready.

### Updated Executive Summary

- The codebase remains architecturally ambitious, with a broad polyglot monorepo and clear separation between UI, service, and sync layers.
- There is a critical TypeScript build failure in the kiosk frontend, which prevents reliable developer workflows and continuous integration from succeeding.
- Security posture is weakened by tracked environment templates (`.env`), browser-exposed dev secrets in the payment UI, and multiple areas where runtime type safety is suppressed.
- The current design still carries prototype debt: duplicate unified vs. federated kiosk implementations, mixed state management patterns, and stubbed approval/checkout flows.

### Critical Findings (Priority High)

1. **Compile blocker in kiosk API client** — `astra-service/apps/kiosk/src/state/apiClient.ts:57`
   - Problem: invalid Authorization header assignment in `AstraApiClient.request()` causes a TypeScript syntax error.
   - Impact: the kiosk app cannot compile, blocking builds and automated type checks.
   - Recommendation: restore the intended header assignment and add a type-safe header merge implementation.

2. **Hardcoded browser-side secret** — `astra-service/apps/kiosk-payment/src/PaymentApp.tsx:191`
   - Problem: `dev-only-32-byte-minimum-secret-key!!` is hardcoded in frontend code and imported into a browser process.
   - Impact: if this value is reused in staging or production builds, it exposes kiosk authentication material and undermines the zero-trust design.
   - Recommendation: remove the hardcoded key, provision per-device secrets via Vault/keyring, and expose only scoped ephemeral tokens to the browser.

3. **Unhandled idle reclaim event** — `astra-service/apps/kiosk/src/machines/kioskMachine.ts:44` and `astra-service/apps/kiosk/src/hooks/useIdleReclaim.ts:35`
   - Problem: `RETURN_TO_ATTRACT` is dispatched after idle timeout but not handled by the state machine.
   - Impact: this can cause runtime errors or silent failure of the idle reclaim workflow.
   - Recommendation: implement a transition from active states to `ATTRACT` or add a guarded noop handler where appropriate.

4. **Tracked `.env` with dev secrets** — `astra-service/.env:50-100`
   - Problem: the repository contains a tracked `.env` file with values such as `VAULT_TOKEN=dev-only-root-token` and `GATEWAY_HMAC_SIGNING_KEY=dev-only-32-byte-minimum-secret-key!!`.
   - Impact: even dev-only templates in source control create accidental secret exposure and make it easier to ship unsafe defaults.
   - Recommendation: remove `.env` from version control, preserve only `.env.example`, and use environment-specific secret provisioning in CI and local development.

5. **Build caching coupled to `.env`** — `astra-service/turbo.json:4`
   - Problem: Turbo globalDependencies includes `.env`, meaning environment file changes affect caching and may leak environment state into task scheduling.
   - Impact: this is an anti-pattern for secure and reproducible monorepo builds.
   - Recommendation: remove `.env` from `globalDependencies` and rely on explicit, secure config inputs instead.

### Debt and Refactor (Priority Medium)

- **Brittle type safety in cart service** — `astra-service/apps/kiosk/src/state/cartService.ts:55-76`
  - The file suppresses multiple type errors and uses unsafe member access throughout the cart sync path. Refactor this module to preserve strong typing and avoid hidden runtime assumptions.
- **Duplicate kiosk implementations** — `astra-service/apps/kiosk/` vs `astra-service/apps/kiosk-shell/`
  - Two kiosk codebases exist in parallel, increasing maintenance cost and risk of divergence. Consolidate to a single implementation or clearly separate the federated shell from the unified app.
- **Payment flow remains prototype-level** — `astra-service/apps/kiosk-payment/src/PaymentApp.tsx`
  - The payment UI still relies on local offline token queueing and lacks a production-ready key distribution model. Add validation and secure secret management before promoting to staging.
- **Mock fallback hides API failures** — `astra-service/apps/kiosk/src/state/apiClient.ts:101-109`
  - The menu client silently falls back to mock data if the API is unavailable. This is useful for development but should be gated behind an explicit dev mode to avoid masking backend problems.

### Optimization and Performance (Priority Low)

- **`useIdleReclaim` polling overhead** — `astra-service/apps/kiosk/src/hooks/useIdleReclaim.ts:31-37`
  - It uses a one-second interval and re-registers on every state value change. This is acceptable for kiosks, but a more event-driven approach would improve correctness and simplify lifecycle management.
- **`docker-compose.yml` exposes many dev ports** — `docker-compose.yml:1-260`
  - Local dev environment publishes ports for Postgres, Redis, NATS, and application services. Confirm these bindings are disabled in shared or production-like environments.
- **Dev-only secrets in compose** — `docker-compose.yml:45-54`
  - The local stack uses plaintext passwords for easy setup. Keep this strictly in local-only compose files and never reuse in CI or production.

---

## 1. Project Structure

```
Astra-System/
├── .changeset/                          # Changesets for version management
├── .github/workflows/
│   └── ci.yml                           # Full CI/CD pipeline (476 lines)
├── .husky/                              # Git hooks (commitlint)
├── .toolchain/                          # Toolchain configs
│
├── astra-service/                       # ★ PRIMARY MONOREPO
│   ├── apps/                            # Frontend applications (pnpm workspace)
│   │   ├── docs/                        # Storybook/Docs app
│   │   ├── kiosk/                       # ★ UNIFIED KIOSK HOST (shell app)
│   │   │   ├── e2e/                     # Playwright E2E tests
│   │   │   ├── src/
│   │   │   │   ├── components/          # UI components (BottomSheet, CartSummary, etc.)
│   │   │   │   ├── ghost-cart/          # WebRTC ghost cart transfer
│   │   │   │   ├── hooks/               # Custom hooks (idle reclaim, silent assist, etc.)
│   │   │   │   ├── machines/            # XState kiosk state machine
│   │   │   │   ├── produce/            # Computer vision produce recognition
│   │   │   │   ├── routes/              # SCREEN IMPLEMENTATIONS (7 screens)
│   │   │   │   ├── state/               # API client, cart service, query client
│   │   │   │   ├── styles/              # Global CSS + Tailwind theme
│   │   │   │   ├── test-utils/          # Test setup + render helpers
│   │   │   │   ├── types/               # Module federation type declarations
│   │   │   │   ├── updater/             # OTA update system
│   │   │   │   ├── wasm/                # CRDT WASM type stub (one file)
│   │   │   │   ├── webauthn/            # WebAuthn employee auth
│   │   │   │   └── workers/             # Service worker + CRDT worker
│   │   │   ├── tailwind.config.ts       # Tailwind theme (161 lines)
│   │   │   ├── vite.config.ts           # Vite + Federation config
│   │   │   └── vitest.config.ts         # Test config
│   │   │
│   │   ├── kiosk-admin/                 # ★ EMPTY / STUB — source not found
│   │   ├── kiosk-cart/                  # Federated cart micro-frontend
│   │   │   └── src/
│   │   │       ├── CartApp.tsx          # Cart component (118 lines)
│   │   │       └── main.tsx             # Entry point
│   │   ├── kiosk-menu/                  # Federated menu micro-frontend
│   │   │   └── src/
│   │   │       ├── MenuApp.tsx          # Virtualized menu (142 lines)
│   │   │       ├── MenuItemCard.tsx     # Individual item card
│   │   │       ├── useMenuCatalog.ts    # Menu data hook
│   │   │       └── main.tsx
│   │   ├── kiosk-payment/               # Federated payment micro-frontend
│   │   │   └── src/
│   │   │       ├── PaymentApp.tsx       # Payment component (193 lines)
│   │   │       ├── verifoneBridge.ts    # Verifone FFI bridge stubs
│   │   │       ├── useWebAuthnEmployeeAuth.ts
│   │   │       └── main.tsx
│   │   └── kiosk-shell/                 # ★ ORIGINAL SHELL (before unification)
│   │       ├── src/                     # Has its own routes, components, state
│   │       │   ├── app/
│   │       │   ├── components/
│   │       │   ├── routes/
│   │       │   ├── state/
│   │       │   ├── styles/
│   │       │   ├── types/
│   │       │   ├── wasm/
│   │       │   └── workers/
│   │       ├── index.html
│   │       └── package.json
│   │
│   ├── packages/                        # Shared TypeScript packages
│   │   ├── cart-engine/                 # Cart totals computation + produce recognition
│   │   │   └── src/
│   │   │       ├── computeTotals.ts     # Tax calculation
│   │   │       ├── produceRecognition.ts # ONNX produce scanner
│   │   │       ├── totals.worker.ts     # Web Worker for totals
│   │   │       └── useCartTotals.ts     # React hook
│   │   ├── config/                      # Shared config utilities
│   │   ├── design-system/              # ★ LIVING WEAVE SYSTEM
│   │   │   └── src/
│   │   │       ├── tokens/             # Design tokens re-export
│   │   │       ├── components/         # Shared components
│   │   │       ├── styles/             # tokens.css
│   │   │       └── utils/
│   │   ├── design-tokens/              # ★ SINGLE SOURCE OF TRUTH
│   │   │   └── src/
│   │   │       ├── tokens.ts           # TypeScript tokens (242 lines)
│   │   │       ├── tokens.css          # CSS custom properties (315 lines)
│   │   │       └── tokens.spec.ts      # Drift guard test
│   │   ├── go-common/                  # Shared Go library
│   │   ├── kiosk-state/               # ★ STATE MANAGEMENT
│   │   │   └── src/
│   │   │       ├── cartProxy.ts        # Valtio cart proxy (106 lines)
│   │   │       ├── sessionStore.ts     # Zustand session store (112 lines)
│   │   │       ├── apiCart.ts          # API cart operations (274 lines)
│   │   │       └── state.spec.ts       # Tests (96 lines)
│   │   ├── shared-types/              # ★ DOMAIN TYPES
│   │   │   └── src/
│   │   │       ├── types/
│   │   │       │   ├── domain.ts       # ALL domain types (538 lines)
│   │   │       │   ├── kiosk.ts        # Kiosk runtime types (208 lines)
│   │   │       ├── schemas/            # Zod schemas
│   │   │       ├── ids.ts              # UUID v7 generation
│   │   │       ├── hlc.ts             # Hybrid Logical Clock types
│   │   │       └── crdt.ts            # CRDT types
│   │   └── ui-kit/                     # Shared UI components
│   │       └── src/
│   │           ├── PrimaryButton.tsx
│   │           ├── EmptyState.tsx
│   │           └── TransparencyPanel.tsx
│   │
│   ├── services/                       # Go microservices
│   │   ├── admin-graphql/              # Admin GraphQL API
│   │   ├── api-gateway/               # API Gateway (Fiber)
│   │   ├── cart-service/              # Cart CRUD
│   │   ├── gateway/                   # Gateway (duplicate?)
│   │   ├── inventory-service/         # Inventory management
│   │   ├── legacy-pos-adapter/        # Strangler Fig pattern
│   │   ├── menu-service/              # Menu catalog
│   │   ├── ml-lane-intel/            # Python ML lane intelligence
│   │   ├── order-service/            # Order lifecycle
│   │   ├── payment-orchestrator/     # Payment orchestration
│   │   ├── payment-service/          # Payment processing
│   │   ├── sync-service/             # Cloud sync gateway
│   │   └── webauthn-service/         # WebAuthn authentication
│   │
│   ├── daemons/
│   │   └── payment-sidecar/          # Payment sidecar daemon
│   │
│   ├── sync-daemon/                   # ★ RUST P2P DAEMON
│   │   └── src/
│   │       ├── main.rs               # Entry point (247 lines)
│   │       ├── lib.rs
│   │       ├── config.rs             # TOML configuration
│   │       ├── p2p/                  # libp2p mesh networking
│   │       ├── raft/                 # Raft consensus
│   │       ├── sync/                 # CRDT sync engine
│   │       ├── storage/              # SQLCipher database
│   │       ├── crypto/               # Crypto primitives
│   │       ├── cloud/                # Cloud sync uploader
│   │       ├── crdt/                 # CRDT types
│   │       ├── network/              # Network utilities
│   │       ├── offline/              # Offline token queue
│   │       ├── protocol/             # P2P protocol
│   │       ├── store/                # Data store
│   │       ├── telemetry/            # OpenTelemetry
│   │       ├── verifone/             # Verifone FFI
│   │       ├── grpc/                # gRPC server
│   │       └── differential_privacy.rs
│   │
│   ├── tools/
│   │   └── chaos/                    # Chaos engineering tools
│   │
│   ├── biome.json                     # Linting config
│   ├── commitlint.config.js
│   ├── eslint.config.js
│   ├── lefthook.yml                   # Git hooks
│   ├── package.json                   # Root pnpm workspace
│   ├── pnpm-workspace.yaml
│   ├── turbo.json                     # Turbo repo config
│   └── tsconfig.base.json             # Shared TypeScript config
│
├── database/
│   ├── migrations/                    # ★ EMPTY
│   └── schemas/                       # ★ EMPTY
│
├── docs/
│   ├── API-BACKEND-ASTRA.md
│   └── runbooks/
│
├── go/                                # Additional Go modules
│   └── pkg/
│
├── infra/
│   ├── docker/
│   │   └── seccomp.json               # ★ Only file — Dockerfiles referenced in CI are MISSING
│   ├── grafana/                       # Dashboard configs
│   ├── k8s/                           # Kubernetes manifests (16 files)
│   ├── loki/                          # Loki config
│   ├── nginx/                         # Nginx config
│   ├── otel/                          # OpenTelemetry collector config
│   ├── postgres/                      # Init scripts
│   ├── prometheus/                    # Prometheus config
│   ├── secrets/                       # SOPS-encrypted secrets (Go)
│   └── tls/                           # Certificate generation scripts
│
├── packages/                          # Root-level (non-TS) packages
│   └── verifone-ffi/                  # Verifone C SDK FFI bindings
│
├── proto/                             # Protocol Buffers
│   ├── proto/                         # .proto files
│   ├── gen/                           # Generated code
│   ├── go.mod
│   └── generate.go
│
├── services/                          # Root-level services
│   └── update-server/                 # OTA update server (Go)
│
├── docker-compose.yml                 # Local dev stack (426 lines)
├── docker-compose.prod.yml
├── flake.nix                          # Nix development shell
├── AGENTS.md                          # Agent instructions
├── ARCHITECTURE.md                    # Architecture docs (608 lines)
├── promt.md                           # Design specification
└── README.md
```

---

## 2. Technology Stack

| Component | Technology | Version | Location |
|---|---|---|---|
| **Monorepo Manager** | pnpm workspaces + Turbo | pnpm 9.12.0, Turbo 2.2.3 | `astra-service/package.json` |
| **Frontend Framework** | React | 19.0.0 | `apps/kiosk/package.json` |
| **TypeScript** | TypeScript | 5.7.2 | Root `package.json` |
| **Build Tool** | Vite | 6.0.1 | `apps/kiosk/vite.config.ts` |
| **Micro-Frontends** | vite-plugin-federation | 1.3.6 | `apps/kiosk/vite.config.ts` |
| **State Machine** | XState | 5.19.0 | `apps/kiosk/package.json` |
| **State (Cart)** | Valtio | 2.1.2 | `packages/kiosk-state/` |
| **State (Session)** | Zustand | 5.0.1 | `packages/kiosk-state/` |
| **Server State** | TanStack Query | 5.59.16 | `apps/kiosk/package.json` |
| **Animation** | Framer Motion | 11.11.9 | `apps/kiosk/package.json` |
| **Styling** | Tailwind CSS v4 | 4.0.0 | `apps/kiosk/package.json` |
| **Design Tokens** | Custom CSS + TS | Internal | `packages/design-tokens/` |
| **HTTP Client** | Native `fetch()` | N/A | `apps/kiosk/src/state/apiClient.ts` |
| **Routing** | XState-driven (NOT react-router) | N/A | `apps/kiosk/src/routes/WorkflowRouter.tsx` |
| **ID Generation** | UUID v7 | Custom | `packages/shared-types/src/ids.ts` |
| **Validation** | Zod | 3.23.8 | `apps/kiosk/package.json` |
| **Testing (Unit)** | Vitest + happy-dom | 2.1.4 / 15.7.0 | `vitest.config.ts` |
| **Testing (E2E)** | Playwright | 1.48.2 | `playwright.config.ts` |
| **CRDT WASM** | Rust → WASM | Stub only | `apps/kiosk/src/wasm/` |
| **PWA** | vite-plugin-pwa + Workbox | 0.20.5 / 7.3.0 | `vite.config.ts` |
| **Service Worker** | Workbox | 7.3.0 | `src/workers/service-worker.ts` |
| **QR Code** | qrcode | 1.5.4 | `apps/kiosk/package.json` |
| **WebRTC** | Native | N/A | `src/ghost-cart/dataChannel.ts` |
| **WebAuthn** | Native `navigator.credentials` | N/A | `src/webauthn/employeeAuth.ts` |
| **NFC** | Web NFC API | N/A | `src/ghost-cart/nfcFallback.ts` |
| **Cryptography (JS)** | @noble/ed25519 + @noble/hashes | 3.1.0 / 2.2.0 | `apps/kiosk/package.json` |
| **Backend Language** | Go | 1.25.1 | `go.work` |
| **Backend Framework** | Fiber (Go) | Not specified | `services/api-gateway/` |
| **P2P Sync** | Rust (libp2p) | 0.53 | `sync-daemon/Cargo.toml` |
| **Runtime** | tokio | 1.35 | `sync-daemon/Cargo.toml` |
| **Database (Kiosk)** | SQLite + SQLCipher | rusqlite 0.30 | `sync-daemon/Cargo.toml` |
| **Database (Cloud)** | PostgreSQL | 16 | `docker-compose.yml` |
| **Cache** | Redis | 7 | `docker-compose.yml` |
| **Event Bus** | NATS JetStream | 2 | `docker-compose.yml` |
| **gRPC** | tonic (Rust) + prost | 0.11 / 0.12 | `sync-daemon/Cargo.toml` |
| **Raft** | Custom implementation | N/A | `sync-daemon/src/raft/` |
| **CRDT** | Custom (PN-Counter, LWW-Register, OR-Set) | N/A | `sync-daemon/src/crdt/` |
| **ML / CV** | Python 3.13 + ONNX + YOLOv8n | 3.13 | `services/ml-lane-intel/` |
| **Observability** | OpenTelemetry + Prometheus + Grafana + Loki + Jaeger | Latest | `infra/` |
| **Secrets** | HashiCorp Vault (planned) + SOPS (dev) | — | Referenced in `.env.example` |
| **CI/CD** | GitHub Actions | — | `.github/workflows/ci.yml` |
| **Container Runtime** | Docker + Docker Compose | — | `docker-compose.yml` |
| **K8s Orchestration** | Kubernetes (EKS/GKE) | — | `infra/k8s/` |

---

## 3. Configuration

### Environment Variables (from `.env.example`)

| Variable | Type | Required | Secret? | Notes |
|---|---|---|---|---|
| `ASTRA_ENV` | `development\|production` | Yes | No | Environment name |
| `ASTRA_LOG_LEVEL` | `info\|debug\|warn\|error` | No | No | Default: info |
| `ASTRA_TRACE_SAMPLE_RATE` | Float 0-1 | No | No | OpenTelemetry sampling |
| `ASTRA_TLS_CA_PATH` | File path | Yes (prod) | Yes | mTLS CA |
| `ASTRA_TLS_CERT_PATH` | File path | Yes (prod) | Yes | mTLS cert |
| `ASTRA_TLS_KEY_PATH` | File path | Yes (prod) | Yes | mTLS key |
| `ASTRA_MTLS_ENABLED` | `true\|false` | No | No | mTLS toggle |
| `POSTGRES_HOST/PORT/DB/USER/PASSWORD` | Various | Yes | Yes | DB credentials |
| `DATABASE_URL` | Connection string | Yes | Yes | Full Postgres URL |
| `REDIS_URL` | URL | Yes | Yes | Redis connection |
| `REDIS_PASSWORD` | String | No | Yes | Redis auth |
| `NATS_URL` | URL | Yes | No | NATS connection |
| `NATS_CLUSTER_ID` | String | Yes | No | NATS cluster |
| `NATS_JETSTREAM_DOMAIN` | String | No | No | JetStream domain |
| `GATEWAY_PORT` | Port number | Yes | No | API gateway port |
| `GATEWAY_JWT_ISSUER` | String | Yes | No | JWT issuer |
| `GATEWAY_HMAC_SIGNING_KEY` | String | Yes | **YES** | Dev-only key in example |
| `GATEWAY_RATE_LIMIT_RPS` | Number | No | No | Rate limiting |
| `VAULT_ADDR` | URL | Yes | No | Vault address |
| `VAULT_TOKEN` | String | Yes | **YES** | Root token in example |
| `KIOSK_ID` | String | Yes | No | Kiosk identity |
| `KIOSK_STORE_ID` | String | Yes | No | Store identity |
| `KIOSK_MESH_PSK` | String | Yes | **YES** | P2P pre-shared key |
| `ASTRA_SYNCD_LISTEN_QUIC` | Address | Yes | No | P2P listen address |
| `ASTRA_SYNCD_MDNS_SERVICE` | String | No | No | mDNS service name |
| `VERIFONE_TERMINAL_IP/PORT` | Address | Yes (prod) | No | Verifone connection |
| `VERIFONE_MERCHANT_ID` | String | Yes | **YES** | Merchant identifier |
| `PAYMENT_OFFLINE_TOKEN_SEED` | String | Yes | **YES** | Offline token seed |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | URL | No | No | OpenTelemetry endpoint |
| `UPDATE_SERVER_URL` | URL | Yes | No | OTA update server |
| `UPDATE_PUBLIC_KEY` | Hex string | Yes | Yes | Update signature verification |
| `ASTRA_SECRETS_BACKEND` | `keyring\|env` | No | No | Secrets backend |

### Hardcoded Secrets in Source Code

| File | Line | Secret |
|---|---|---|
| `apps/kiosk-payment/src/PaymentApp.tsx` | 191 | `"dev-only-32-byte-minimum-secret-key!!"` — HMAC signing key |
| `.env.example` | 51 | `"dev-only-32-byte-minimum-secret-key!!"` — Gateway HMAC key |
| `.env.example` | 60 | `"dev-only-root-token"` — Vault root token |
| `.env.example` | 69 | `"dev-only-mesh-pre-shared-key-32b"` — Mesh PSK |
| `.env.example` | 93 | `"dev-only-offline-token-seed-32b-min"` — Offline token seed |

### Configuration Files

| File | Type | Purpose |
|---|---|---|
| `.env.example` | Environment | Reference for all env vars |
| `docker-compose.yml` | Docker | Local dev stack (426 lines) |
| `docker-compose.prod.yml` | Docker | Production stack |
| `flake.nix` | Nix | Dev shell with all tools |
| `infra/k8s/*.yaml` | Kubernetes | 16 deployment manifests |
| `infra/otel/otel-collector.yml` | OpenTelemetry | Collector config |
| `infra/prometheus/prometheus.yml` | Prometheus | Metrics config |
| `infra/grafana/*.json` | Grafana | Dashboard definitions |
| `infra/postgres/init/*.sql` | SQL | DB initialization |
| `infra/tls/generate-certs.sh` | Shell | Certificate generation |
| `infra/docker/seccomp.json` | Security | Docker seccomp profile |
| `infra/nginx/*.conf` | Nginx | Reverse proxy config |
| `infra/secrets/*.age` | Encryption | SOPS-encrypted secrets |
| `proto/proto/*.proto` | Protobuf | API schema definitions |
| `.commitlintrc.json` | Commitlint | Conventional commit rules |
| `.prettierrc` | Prettier | Code formatting |
| `.editorconfig` | EditorConfig | Editor settings |
| `biome.json` | Biome | Linting (21 lines) |
| `.spectral.json` | Spectral | API linting |

---

## 4. Architecture & Data Flow

### Communication Protocols

```
┌─────────────────────────────────────────────────────────────────────┐
│                        KIOSK BROWSER (React 19)                      │
│                                                                      │
│  ┌─────────────────────┐  ┌──────────────────┐  ┌───────────────┐   │
│  │ XState Machine      │  │ Valtio Cart Proxy│  │ Zustand       │   │
│  │ (kioskMachine.ts)   │  │ (cartProxy.ts)   │  │ (sessionStore)│   │
│  └────────┬────────────┘  └────────┬─────────┘  └───────┬───────┘   │
│           │                        │                      │          │
│           ▼                        ▼                      ▼          │
│  ┌──────────────────────────────────────────────────────────────┐   │
│  │                    WorkflowRouter.tsx                          │   │
│  │  (Attract → Menu → Cart → Payment → Processing → Receipt)    │   │
│  └──────────────────────────┬───────────────────────────────────┘   │
│                              │                                       │
│              ┌───────────────┴────────────────┐                      │
│              ▼                                ▼                      │
│     ┌─────────────────┐             ┌─────────────────┐              │
│     │  HTTP fetch()   │             │  WebSocket/SSE  │   ❌ NOT USED│
│     │  (apiClient.ts) │             │  (Not impl.)    │              │
│     └────────┬────────┘             └─────────────────┘              │
│              │                                                       │
│              ▼  REST over HTTP                                       │
│     ┌────────────────┐                                              │
│     │  API Gateway   │  Go/Fiber on :8080 (❌ NOT FUNCTIONAL)        │
│     └────────┬───────┘                                              │
│              │                                                       │
│              ▼                                                       │
│     ┌──────────────────────────────────────────┐                    │
│     │  gRPC (service-to-service, mTLS)         │                    │
│     │  NATS JetStream (event bus)              │                    │
│     └──────────────────────────────────────────┘                    │
│                                                                      │
│  ┌──────────────────────────────────────────────┐                    │
│  │  KIOSK <-> KIOSK: libp2p QUIC + Noise        │                    │
│  │  CRDT sync over P2P mesh                     │                    │
│  └──────────────────────────────────────────────┘                    │
│                                                                      │
│  ┌──────────────────────────────────────────────┐                    │
│  │  KIOSK <-> VERIFONE: FFI bridge (Rust daemon) │  ❌ STUB         │
│  └──────────────────────────────────────────────┘                    │
└─────────────────────────────────────────────────────────────────────┘
```

### Frontend → Backend Communication

- **Currently**: All API calls go through `AstraApiClient` (`state/apiClient.ts`), which uses native `fetch()` to make REST calls to `http://localhost:8080`
- **Reality**: **No backend is running during development.** All calls silently fall through to mock data (`mockMenuData.ts`) or throw errors that are caught and swallowed
- **Planned**: gRPC + NATS event bus for service-to-service; mTLS for all internal communication

### State Management Architecture

```
Layer 1: XState v5 (kioskMachine.ts)
  ├── Global kiosk workflow state machine
  ├── States: ATTRACT → MENU → ITEM_DETAIL → CART → PAYMENT → PROCESSING → RECEIPT → ADMIN
  ├── Guards: cartHasItems, paymentApproved, paymentDeclined
  └── Actors: finalizeOrder (mock)

Layer 2: Valtio (cartProxy.ts)
  ├── Cart state: lines[], cartId, sessionId, version, currency
  ├── Mutable proxy — changes are captured by Valtio's subscribe()
  └── Bridges to CRDT worker via bridgeCartToCrdtWorker()

Layer 3: Zustand (sessionStore.ts)
  ├── Session state: stage, laneMode, network status, silent assist
  └── Selector-based subscriptions for performance
```

### Navigation Flow

```
ATTRACT ──(tap anywhere)──→ MENU ──(select item)──→ ITEM_DETAIL ──(add to cart / close)──→ MENU
                                │                                                    │
                                └──(go to cart)──→ CART ──(pay)──→ PAYMENT ──(authorize)──→ PROCESSING ──(done)──→ RECEIPT ──(ack)──→ ATTRACT
                                                    ↑                 │                         │
                                                    └──(cancel/declined)←───────────────────────┘
                                                      
ADMIN ── accessible from any screen via OPEN_ADMIN event
```

**Critical Issue**: Navigation is driven entirely by XState machine transitions via `WorkflowRouter.tsx`. While `react-router-dom` v6 is in `package.json` dependencies, it is NEVER imported anywhere. The router dependency is dead code.

### Payment Flow (Current)

1. User taps "Confirm Payment" on `PaymentAuthScreen.tsx`
2. `showBiometric` state is set — a demo modal appears with "Authorize" button
3. `handleBiometricComplete` calls `apiClient.checkoutCart()` then `apiClient.processPayment()`
4. On success, sends `PAYMENT_AUTHORIZED` event to XState machine
5. XState machine transitions to `PROCESSING` — which invokes the `finalizeOrder` actor
6. `finalizeOrder` creates a **fake order** with 800ms delay
7. Machine transitions to `RECEIPT` screen

**Reality**: Steps 3-4 will always fail (no backend), so `PAYMENT_FAILED` is sent instead. The error handler in `ProcessingScreen.tsx` catches this. The entire payment flow is non-functional end-to-end without a backend.

### Offline Strategy (Current Implementation)

- Two network monitors: `useNetworkMonitor.ts` (polls local syncd daemon at `127.0.0.1:4499`) and `useApiNetworkMonitor.ts` (sends `NETWORK_ONLINE`/`NETWORK_OFFLINE` to machine)
- Service worker caches menu data (StaleWhileRevalidate) and uses Background Sync for order mutations
- Cart operations always write to local Valtio proxy first, then attempt to sync to server
- `OfflineBanner.tsx` shows "Working offline. Your cart is secure." when offline detected

### P2P Mesh (Rust Daemon)

The `sync-daemon` (Rust) implements:
- libp2p with QUIC transport and Noise encryption
- mDNS peer discovery
- Raft consensus for leader election (3+ nodes)
- CRDT sync engine (PN-Counters, LWW-Registers, OR-Sets)
- SQLCipher encrypted local storage
- gRPC server for local IPC from browser (`127.0.0.1:4499`)
- Cloud sync when online and raft leader

The browser talks to the daemon via HTTP to `127.0.0.1:4499` for heartbeat/health checks. The CRDT worker would communicate via WASM, but the WASM module is a stub.

---

## 5. Screen Implementation Status

| # | Screen | File | Status | Notes |
|---|---|---|---|---|
| 1 | **Attract Loop** | `src/routes/AttractScreen.tsx` | ⚠️ **Partial** | Animated blobs working with Framer Motion. Tap-to-start functional. Clip-path reveal animation exists but uses CSS-viewport coordinates (broken on scaled viewports). No idle dim timer after 2 min — the hook exists but `handleTap` doesn't clear it. "Lane 3" text is hardcoded. |
| 2 | **Menu Browse** | `src/routes/MenuScreen.tsx` | ⚠️ **Partial** | Category chips functional. IntersectionObserver for scroll-spy implemented. Pull-down-to-search implemented. **Uses mock data only** — API always falls back to `mockMenuData.ts`. Virtual list NOT used (TanStack Virtual is in federated menu only). Floating cart pill works. Ghost Cart bottom sheet has placeholder "Transfer to kiosk" button with no logic. |
| 3 | **Item Detail / Customization** | `src/routes/ItemModal.tsx` | ✅ **Complete** | Bottom sheet with Framer Motion drag-to-dismiss. Image placeholder, description, price. Modifier radio groups with validation (minSelect/maxSelect). Quantity stepper. "Add to cart" button with total calculation. Swipe-down to dismiss. State resets properly on open/close. |
| 4 | **Cart Review** | `src/routes/CartReviewScreen.tsx` | ⚠️ **Partial** | Full screen for >5 items, bottom sheet for ≤5. Quantity steppers work. **Missing:** "Tap an item to edit" text is shown but items are NOT tappable — no `onClick` on the line item wrapper. Silent assist pulse on Pay button works. Tax is hardcoded at 8% (`taxCents = Math.round(subtotalCents * 0.08)`). |
| 5 | **Payment Auth** | `src/routes/PaymentAuthScreen.tsx` | ⚠️ **Partial** | Payment method selection (3 methods) works. Collapsible cart summary works. **Biometric auth modal is a simulation** — buttons are labeled "Authorize" but no actual Verifone/FIDO2 integration. Employee override (long-press corner) sends a fake `PAYMENT_AUTHORIZED` with no authentication. Payment API calls will fail (no backend). |
| 6 | **Processing** | `src/routes/ProcessingScreen.tsx` | ⚠️ **Partial** | Animated overlay with 4 sequential stages. Progress dots fill correctly. **Order creation is mocked** — uses `apiClient.createOrder()` which will fail; falls back to `PAYMENT_FAILED`. No real connection to payment terminal. |
| 7 | **Receipt / Confirmation** | `src/routes/ReceiptScreen.tsx` | ⚠️ **Partial** | Checkmark SVG stroke animation works. Order number displays correctly. Print and Email buttons are **placeholders** — both simulate with `await new Promise(resolve => setTimeout(resolve, 1000))`. "Start new order" appears after 3 seconds delay. Auto-return timer of 10 seconds. Printer failure toast animation works. |
| 8 | **Admin / Assist** | `src/components/IdleTimeoutOverlay.tsx` | ❌ **Missing** | The kiosk machine has an `ADMIN` state, the `OPEN_ADMIN`/`CLOSE_ADMIN` events exist, but there is NO Admin screen component. `WorkflowRouter.tsx` has no case for `ADMIN` — it falls through to `AttractScreen`. The `IdleTimeoutOverlay.tsx` exists but is NOT integrated in the app anywhere. |

### Federated Micro-Frontend Status

| App | File | Status | Notes |
|---|---|---|---|
| **kiosk-cart** | `apps/kiosk-cart/src/CartApp.tsx` | ⚠️ **Partial** | Full implementation with cart items, stepper, transparent pricing, checkout button. Uses the same Valtio proxy. **Not actually loaded by the host** — the host uses its own inline `CartReviewScreen.tsx`. |
| **kiosk-menu** | `apps/kiosk-menu/src/MenuApp.tsx` | ⚠️ **Partial** | Virtualized grid with TanStack Virtual (2 columns). Category chip filter. Silent assist highlight. Express mode (lane intelligence). **Not actually loaded by the host** — the host uses `MenuScreen.tsx`. |
| **kiosk-payment** | `apps/kiosk-payment/src/PaymentApp.tsx` | ⚠️ **Partial** | Payment method selection, Verifone bridge stubs, offline token queuing. **Has actual offline token code** that POSTs to `127.0.0.1:4499`. Contains hardcoded dev HMAC key. **Not consumed by host.** |
| **kiosk-shell** | `apps/kiosk-shell/src/` | ❌ **Orphaned** | Older shell implementation with its own routes, components, state. Duplicates much of `apps/kiosk`. Unclear which is the canonical version. |

---

## 6. Touch & Reactivity Audit

### Touch Handling Analysis

| Concern | Current Implementation | Verdict |
|---|---|---|
| **Tap delay** | `touch-action: manipulation` set in `global.css` line 189 | ✅ 300ms delay eliminated |
| **Zoom prevention** | `gesturestart` listener in `App.tsx` lines 41-43 + viewport `user-scalable=no` | ⚠️ `gesturestart` is a Safari/iOS event; Android uses `touch-action: pinch-zoom` CSS; no Android-specific blocking |
| **Double-tap zoom** | Not explicitly prevented | ❌ `touch-action: manipulation` should handle this, but no `touch-action: pan-y pinch-zoom` override |
| **Long-press context menu** | `-webkit-touch-callout: none` + `user-select: none` in `global.css` line 184-185 | ✅ Prevented |
| **Viewport lock** | `ViewportLock.tsx` — CSS scale transform on fixed 1080×1920 logical | ❌ **BROKEN** — `clipPath` reveal animation and `tapPoint` coordinates use CSS client coordinates, not scaled logical coordinates. On 1440×2560 panels, taps will be offset by ~33%. |
| **Overscroll** | `overscroll-behavior: none` in `global.css` line 187 | ✅ Prevented |
| **Scroll areas** | `-webkit-overflow-scrolling: touch` NOT specified | ❌ Missing on iOS — default momentum scrolling is used |
| **Touch target sizes** | 56px minimum via design tokens, buttons are 48-64px | ✅ WCAG 2.2 AA compliant |
| **Pointer events** | `pointerdown` used for idle reclaim; `onClick` used for most interactions | ⚠️ Mix of pointer and click events; should prefer `onPointerDown` for touch latency |
| **Passive event listeners** | Not explicitly set | ⚠️ Modern browsers default to passive, but no guarantee; `preventDefault` on `gesturestart` is synchronous |

### Key Touch Issues

**ViewportLock coordinate mapping bug** (`ViewportLock.tsx` lines 15-48):
```typescript
const LOGICAL_WIDTH = 1080;
const LOGICAL_HEIGHT = 1920;
// ...
// scale is applied as CSS transform, but event coordinates are in pre-transform space
const scaleX = window.innerWidth / LOGICAL_WIDTH;
const scaleY = window.innerHeight / LOGICAL_HEIGHT;
setScale(Math.min(scaleX, scaleY)); // uniform scale
```
This means Framer Motion's `tapPoint.current = { x: e.clientX, y: e.clientY }` in `AttractScreen.tsx` line 26 records coordinates in the pre-scaled viewport space. The clip-path reveal animation then uses these coordinates on a scaled element, causing misalignment on non-1080×1920 screens.

**BottomSheet drag gesture** (`BottomSheet.tsx`) — uses Framer Motion's `drag="y"` but there's no `onDragEnd` handler. The sheet backdrop click calls `onClose`, but the sheet itself can only be closed via the backdrop, not by dragging down.

**Scrollable menu area** (`MenuScreen.tsx`) — uses `overflow-y-auto` but no `-webkit-overflow-scrolling: touch` for iOS smooth scrolling.

---

## 7. Navigation & Routing Audit

### State Machine Routing

The app uses XState machine-based routing via `WorkflowRouter.tsx`:

```typescript
// WorkflowRouter.tsx line 12
const stage = state.value as string;
// Returns the FIRST matching screen (no <Switch> or <Routes>)
```

**Issues:**

1. **`RETURN_TO_ATTRACT` event is NOT handled** — `useIdleReclaim.ts` line 35 sends `{ type: "RETURN_TO_ATTRACT" }` but `kioskMachine.ts` has no handler for this event type. The runtime will throw an XState error. This is a **critical runtime bug**.

2. **`react-router-dom` is dead weight** — It's in `package.json` dependencies but never imported anywhere in the codebase.

3. **Admin state has no screen** — The `ADMIN` state exists in the machine, `WorkflowRouter.tsx` has no `case "ADMIN"`, so the fallback is `AttractScreen`. The `IdleTimeoutOverlay` component exists but is never mounted.

4. **MenuItemCard in federated menu uses `@tanstack/react-virtual`** — but the host's inline `MenuScreen.tsx` does NOT virtualize; it renders all items. On a 500-item catalog this could be a performance issue.

5. **No URL-based routing** — The kiosk has no URL bar; all navigation is in-memory. This is correct for a kiosk but means there's no deep linking for debugging.

### Screen Transition Analysis

| Transition | Current State | Issues |
|---|---|---|
| ATTRACT → MENU | ✅ Working | Tap handler sends `START_SESSION` after 500ms delay |
| MENU → ITEM_DETAIL | ✅ Working | Category scroll-spy with IntersectionObserver |
| ITEM_DETAIL → MENU | ✅ Working | Both "Add to Cart" and "Close" return to MENU |
| MENU → CART | ✅ Working | Guarded by `cartHasItems` |
| CART → PAYMENT | ✅ Working | Guarded by `cartHasItems` |
| PAYMENT → PROCESSING | ⚠️ Blocked | Requires `PAYMENT_AUTHORIZED` which requires API call that will fail |
| PROCESSING → RECEIPT | ⚠️ Blocked | `finalizeOrder` actor creates fake order; machine waits for it |
| RECEIPT → ATTRACT | ✅ Working | Auto-returns after 10s; manual "Start new order" works |
| Any → ADMIN | ❌ Broken | No admin screen component; falls through to Attract |
| Idle → ATTRACT | ❌ **RUNTIME ERROR** | `RETURN_TO_ATTRACT` event not handled by machine |

---

## 8. Known Issues & TODOs

### Issue 1: Critical Runtime Error — Unhandled Event
**File:** `apps/kiosk/src/hooks/useIdleReclaim.ts` line 35  
**File:** `apps/kiosk/src/machines/kioskMachine.ts` — no `RETURN_TO_ATTRACT` handler  
**Severity:** 🔴 P0 — Causes runtime error after 90 seconds of inactivity  
**Description:** `useIdleReclaim` sends `{ type: "RETURN_TO_ATTRACT" }` when idle timeout fires, but `kioskMachine.ts` has no event handler for this type. XState will throw an error for unhandled events (unless `strict` is off, which it isn't explicitly disabled).

### Issue 2: `@ts-expect-error` Suppressing Real Type Errors
**File:** `apps/kiosk/src/state/apiClient.ts` lines 65, 79  
**File:** `apps/kiosk/src/state/cartService.ts` lines 55, 70, 73  
**Severity:** 🟡 P2 — 5 suppressed type errors  
**Description:** The `@ts-expect-error` annotations on `options.headers` spread and `return response.json()` in apiClient.ts indicate the `RequestInit` types are not compatible with the spread pattern. In cartService.ts, the `CartLineItem` type assertions with `@ts-expect-error` suggest the types are incorrect or incomplete.

### Issue 3: Hardcoded Dev HMAC Signing Key
**File:** `apps/kiosk-payment/src/PaymentApp.tsx` line 191  
**Severity:** 🔴 P0 — Security vulnerability  
**Code:**
```typescript
const rawKey = new TextEncoder().encode("dev-only-32-byte-minimum-secret-key!!");
```
This dev key is used to sign offline payment tokens. If this were deployed to production, anyone who reads the minified JS could forge offline payment tokens.

### Issue 4: TWO Competing Kiosk Implementations
**File:** `apps/kiosk/` and `apps/kiosk-shell/`  
**Severity:** 🟡 P2 — Architectural confusion  
**Description:** Both directories contain full kiosk implementations: routes, components, state management, styles, workers. The `kiosk-shell` appears to be an older pattern before the "unified kiosk" approach. The relationships and migration path are undocumented.

### Issue 5: Federated Micro-Frontends NOT Consumed by Host
**File:** `apps/kiosk/src/routes/WorkflowRouter.tsx`  
**File:** `apps/kiosk/vite.config.ts` (remotes configured)  
**Severity:** 🟡 P2 — Major architectural disconnect  
**Description:** The vite config declares `astra_menu`, `astra_cart`, `astra_payment` as Module Federation remotes, and `federation.d.ts` has type declarations for them. But `WorkflowRouter.tsx` uses local inline screen components (`MenuScreen`, `CartReviewScreen`, `PaymentAuthScreen`). The federated apps (`kiosk-cart`, `kiosk-menu`, `kiosk-payment`) are completely independent apps that are never loaded by the host.

### Issue 6: ViewportLock Coordinate System Bug
**File:** `apps/kiosk/src/components/ViewportLock.tsx`  
**Severity:** 🟡 P2 — Premium lane touch misalignment  
**Description:** Fixed 1080×1920 logical viewport with CSS scale transform breaks touch coordinate mapping on 1440×2560 panels.

### Issue 7: Empty Admin Screen
**File:** `apps/kiosk/src/routes/WorkflowRouter.tsx`  
**Severity:** 🟡 P2 — Feature gap  
**Description:** `ADMIN` state in machine has no corresponding screen component or route handler.

### Issue 8: OfflineBanner Logic Bug
**File:** `apps/kiosk/src/components/OfflineBanner.tsx` lines 11-22  
**Severity:** 🟢 P3 — UI inconsistency  
**Code:**
```typescript
useEffect(() => {
  if (isOffline) {
    setShowBanner(true);
    const timer = setTimeout(() => { setShowBanner(false); }, 5000);
    return () => { clearTimeout(timer); };
  } else {
    setShowBanner(false);
  }
}, [isOffline]);
```
If `isOffline` toggles between true and false within the 5-second window, the banner may stay hidden while offline. The `showBanner` and `isOffline` conditions are redundant.

### Issue 9: No Service Worker HMR Update Handling
**File:** `apps/kiosk/src/workers/service-worker.ts`  
**Severity:** 🟢 P3 — Missing feature  
**Description:** Service worker uses `autoUpdate` strategy but has no handler for `controllerchange` events. When the SW updates, the app will not reload to pick up new assets, potentially serving stale shells.

### Issue 10: Hardcoded Tax Rate
**File:** `apps/kiosk/src/routes/CartReviewScreen.tsx` line 38  
**Severity:** 🟢 P3 — Not configurable  
**Code:** `const taxCents = Math.round(subtotalCents * 0.08);`

### Issue 11: Incomplete Processing Flow
**File:** `apps/kiosk/src/routes/ProcessingScreen.tsx` lines 38-56  
**Severity:** 🔴 P0 — Order creation is fully mocked
```typescript
const paymentId = crypto.randomUUID();
const cartId = "current-cart-id"; // Hardcoded placeholder
```
The actual cart ID is never passed to the processing screen. The order is created with a meaningless cart ID.

### Issue 12: `console.warn` and `console.error` Scattered Throughout Production Code
**Files:** `apiClient.ts`, `cartService.ts`, `useProduceScanner.ts`, etc.  
**Severity:** 🟢 P3 — Cleanliness  
**Description:** Multiple `console.warn`/`console.error` calls for expected failure modes (e.g., "API not available, using mock data"). Biome linter has `noConsoleLog: "warn"` but does not block `warn`/`error`.

### Issue 13: `react-router-dom` is Dead Dependency
**File:** `apps/kiosk/package.json` line 32  
**Severity:** 🟢 P3  
**Description:** `react-router-dom` is listed as a dependency but never imported anywhere in the codebase.

### Issue 14: Missing Dockerfiles
**File:** `.github/workflows/ci.yml` references `infra/docker/Dockerfile.*`  
**Severity:** 🟡 P2 — CI builds would fail  
**Description:** The CI workflow matrix references 11 Dockerfiles (Dockerfile.gateway, Dockerfile.menu-service, etc.) but `infra/docker/` contains only `seccomp.json`.

### Issue 15: Empty Database Directory
**File:** `database/migrations/` and `database/schemas/`  
**Severity:** 🟡 P2 — No database schema defined  
**Description:** Both directories exist but are empty. No SQL migration files, no Drizzle schema, no Go struct definitions are present.

### Issue 16: Ghost Cart Transfer Has No Logic
**File:** `apps/kiosk/src/routes/MenuScreen.tsx` lines 358-383  
**Severity:** 🟡 P2 — Placeholder only  
**Description:** The Ghost Cart bottom sheet shows "Cart found on your phone" but neither the "Cancel" nor "Transfer to kiosk" buttons do anything meaningful — both just call `setGhostCartOpen(false)`.

### Issue 17: Silence Assist Not Connected
**File:** `apps/kiosk/src/hooks/useSilentAssist.ts`  
**Severity:** 🟢 P3  
**Description:** The hook arms `silentAssistArmed` in the session store, but no component reads this value to render the pulse animation. The `CartReviewScreen.tsx` has its own local `silentAssist` state with the same logic.

### Issue 18: `PaymentAuthScreen.tsx` Missing Emoji Fallback
**File:** Not applicable, but the spec says no emojis; code uses plain SVG icons ✅

---

## 9. Build & Deployment

### Build Commands

| Command | Purpose | Works? |
|---|---|---|
| `pnpm install` | Install all dependencies | ⚠️ Check — `pnpm-lock.yaml` exists but lockfile may be stale |
| `pnpm build` | Turbo build all packages and apps | ⚠️ Untested — depends on TypeScript compilation |
| `pnpm dev` | Start all dev servers in parallel | ⚠️ Starts Vite on :5180 |
| `pnpm typecheck` | TypeScript type checking | ⚠️ Will likely fail due to `@ts-expect-error` suppression |
| `pnpm lint` | Biome + ESLint | ⚠️ ESLint config exists but rules may conflict with code |
| `pnpm test` | Vitest unit tests | ⚠️ 13 spec files, coverage unknown |
| `pnpm test:e2e` | Playwright E2E | ⚠️ E2E test references federated remotes that aren't running |

### Docker Status

- **Docker Compose**: Comprehensive development stack (`docker-compose.yml`) with all Go services, Postgres, Redis, NATS, Jaeger, OTEL collector, Python ML, Rust sync daemon, and Vite kiosk dev server
- **Issue**: `docker compose up -d` uses `golang:1.25-alpine` (not 1.22 as in `AGENTS.md`) — version mismatch with go.work (`go 1.25.1`)
- **Issue**: The kiosk service uses `pnpm turbo run dev --filter=@astra/kiosk` but the kiosk depends on `gateway` which depends on Postgres/Redis/NATS — the gateway won't start properly without database connections
- **Production Dockerfiles**: Referenced in CI but **do not exist** in `infra/docker/`

### CI/CD Status

- **GitHub Actions**: Comprehensive pipeline at `.github/workflows/ci.yml` (476 lines)
- **Linting**: TypeScript (Biome), Go (gofmt + go vet), Rust (clippy) — all configured
- **Testing**: Unit tests for all 3 languages, integration tests, E2E (Playwright)
- **Security**: npm audit, cargo audit, govulncheck, Trivy filesystem scan
- **Docker Build**: Matrix build for 11 Docker images with cosign signing
- **SBOM**: Syft SPDX generation with cosign attestation
- **Chaos**: Chaos engineering job (disabled by default — requires `ASTRA_RUN_CHAOS` variable)

### Output Directories

- `apps/kiosk/dist/` — Optimized for kiosk (but no actual production build has been run)
- Vite config has `chunkSizeWarningLimit: 180KB` and manual chunk splitting:
  - `vendor-react` (react, react-dom)
  - `vendor-state` (valtio, zustand, tanstack-query)
  - `vendor-motion` (framer-motion)

---

## 10. Testing & Quality

### Test Files Found

| File | Type | Status |
|---|---|---|
| `src/machines/kioskMachine.spec.ts` | Unit (XState) | ✅ 8 tests — good coverage of state transitions |
| `src/state/state.spec.ts` (in kiosk-state) | Unit (Cart/Session) | ✅ 6 tests — good basic coverage |
| `src/components/AttractScreen.spec.tsx` | Unit | Present but untested |
| `src/components/CartSummary.spec.tsx` | Unit | Present but untested |
| `src/components/StatusBar.spec.tsx` | Unit | Present but untested |
| `src/hooks/useIdleReclaim.spec.tsx` | Unit | Present but untested |
| `src/hooks/useNetworkMonitor.spec.tsx` | Unit | Present but untested |
| `src/hooks/useSilentAssist.spec.tsx` | Unit | Present but untested |
| `src/routes/ItemModal.spec.tsx` | Unit | Present but untested |
| `src/ghost-cart/dataChannel.spec.ts` | Unit | Present but untested |
| `src/produce/useProduceScanner.spec.ts` | Unit | Present but untested |
| `src/webauthn/employeeAuth.spec.ts` | Unit | Present but untested |
| `src/updater/updater.spec.ts` | Unit | Present but untested |
| `e2e/kiosk-flow.spec.ts` | E2E (Playwright) | ⚠️ 1 test — references federated remotes not running |
| `packages/design-tokens/src/tokens.spec.ts` | Unit (Drift guard) | ✅ Ensures CSS ↔ TS token parity |
| `packages/cart-engine/src/computeTotals.spec.ts` | Unit | Present but untested |
| `packages/cart-engine/src/produceRecognition.spec.ts` | Unit | Present but untested |

**Total: 18 test files** — but only 2 have meaningful test implementations (kioskMachine.spec.ts has 8 tests, state.spec.ts has 6 tests). The other 14 files are likely empty or placeholder test files.

### Test Configuration

- **Framework**: Vitest 2.1.4
- **Environment**: happy-dom (not jsdom) — lighter but may miss some DOM features
- **Coverage**: V8 provider with text + HTML reporters — but no coverage thresholds set
- **E2E**: Playwright on Chromium kiosk mode (1080×1920, touch-enabled)

### Linting & Formatting

| Tool | Config | Usage |
|---|---|---|
| **Biome** | Linting + formatting | `biome.json` — `noExplicitAny: "error"`, `noConsoleLog: "warn"` |
| **ESLint** | Additional React rules | `eslint.config.js`, `eslint-plugin-react-hooks`, `eslint-plugin-react-refresh` |
| **Prettier** | Code formatting | `.prettierrc` + `prettier-plugin-tailwindcss` |
| **Commitlint** | Conventional commits | `commitlint.config.js` + lefthook |
| **Spectral** | API linting | `.spectral.json` |

### Issues Found

- Biome has `noConsoleLog: "warn"` but the codebase uses `console.warn`, `console.error`, and `console.log` extensively without suppression
- `@ts-expect-error` is used in 5 places, which would fail `noExplicitAny: "error"` if the suppressed errors involve `any`
- No `package.json` lint script runs `biome check` — only ESLint via turbo

---

## 11. Immediate Action Items

### 🔴 P0 — Kiosk Unusable / Security Critical

| # | Item | File(s) | Effort |
|---|---|---|---|
| 1 | **Fix `RETURN_TO_ATTRACT` unhandled event** — Add handler to all XState states to transition to ATTRACT | `kioskMachine.ts`, `useIdleReclaim.ts` | 30 min |
| 2 | **Remove hardcoded dev HMAC key** from PaymentApp.tsx before any deployment | `PaymentApp.tsx` line 191 | 15 min |
| 3 | **Fix `finalizeOrder` mock** — Either connect to real API or make the mock work with actual cart data | `kioskMachine.ts` (finalizeOrder actor) | 2 hrs |
| 4 | **Fix ViewportLock coordinate mapping** — Convert tap coordinates from CSS-viewport space to logical space, or use a different approach (e.g., `aspect-ratio` CSS) | `ViewportLock.tsx`, `AttractScreen.tsx` | 4 hrs |

### 🟡 P1 — Major Feature Broken

| # | Item | File(s) | Effort |
|---|---|---|---|
| 5 | **Create Admin screen component** and wire it into `WorkflowRouter.tsx` | New file + `WorkflowRouter.tsx` | 4 hrs |
| 6 | **Resolve dual-kiosk confusion** — Move `apps/kiosk-shell` into `apps/kiosk/legacy` or document the relationship | — | 2 hrs |
| 7 | **Either consume federated remotes or remove them** — The host either needs to lazy-load the 3 micro-frontends or the code duplication needs to be eliminated | `vite.config.ts`, `WorkflowRouter.tsx`, federation types | 8 hrs |
| 8 | **Fix OfflineBanner logic** — Race condition between `isOffline` and `showBanner` | `OfflineBanner.tsx` | 1 hr |
| 9 | **Implement real payment integration or remove fake Authorize button** — The current biometric auth modal is a security theater | `PaymentAuthScreen.tsx` | 4 hrs |

### 🟢 P2 — Polish / Cleanup

| # | Item | File(s) | Effort |
|---|---|---|---|
| 10 | **Resolve `@ts-expect-error` annotations** — Fix underlying type issues instead of suppressing | `apiClient.ts`, `cartService.ts` | 4 hrs |
| 11 | **Remove `react-router-dom` dependency** if not needed | `package.json` | 15 min |
| 12 | **Create Dockerfiles** in `infra/docker/` for all CI-referenced images | `infra/docker/` | 8 hrs |
| 13 | **Add database migrations** — Create actual `.sql` files or Drizzle schema | `database/migrations/` | 8 hrs |
| 14 | **Connect "Tap an item to edit"** in CartReviewScreen — Currently text is shown but items are not clickable | `CartReviewScreen.tsx` | 1 hr |
| 15 | **Fix hardcoded "Lane 3"** text to use actual lane config | `AttractScreen.tsx` | 15 min |
| 16 | **Add real Ghost Cart transfer logic** — The bottom sheet buttons are placeholders | `MenuScreen.tsx` | 4 hrs |
| 17 | **Add `-webkit-overflow-scrolling: touch`** to scrollable containers | `global.css`, `MenuScreen.tsx` | 30 min |
| 18 | **Add `@media (pointer: coarse)` queries** for touch-only optimizations | `global.css` | 1 hr |
| 19 | **Set coverage thresholds** in `vitest.config.ts` | `vitest.config.ts` | 30 min |
| 20 | **Populate test files** — 14 of 18 spec files are empty/placeholder | Multiple files | 8 hrs |

---

## Appendix: Key File Statistics

| Metric | Value |
|---|---|
| Total TypeScript/TSX files analyzed | 60+ |
| Total lines of TypeScript/TSX | ~8,500 |
| Total Rust files | ~30 |
| Total Go service directories | 13 (mostly stubs) |
| Test files | 18 (but only 2 have meaningful tests) |
| `@ts-expect-error` annotations | 5 |
| `@ts-ignore` annotations | 0 |
| `@ts-nocheck` annotations | 0 |
| Hardcoded secrets | 5 dev credentials |
| TODOs/FIXMEs/HACKs | 0 found (well-commented codebase) |
| Dead dependencies | `react-router-dom` |
| Console.error/warn calls | 15+ in production code |
