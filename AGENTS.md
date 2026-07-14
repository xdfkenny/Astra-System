# Astra-System

Compact instruction file for OpenCode agents.

## Design reference

The full design spec ("Living Weave" biophilic kiosk UI) lives in **`promt.md`** — always read it before touching kiosk UI code. It specifies pixel-level colors, typography, spacing, animation curves, and component specs.

## Repository architecture

Multi-language monorepo rooted at `astra-service/`:

- **TypeScript**: pnpm workspaces + Turborepo. Apps in `apps/*`, shared packages in `packages/*`.
- **Go**: `go.work` workspace at `astra-service/go.work`. All service modules in `astra-service/services/*` + root `services/update-server` + `infra/secrets`.
- **Rust**: `sync-daemon` at `astra-service/sync-daemon` + `payment-sidecar` at `astra-service/daemons/payment-sidecar`.
- **Python ML**: `ml-lane-intel` at `astra-service/services/ml-lane-intel` (profiled behind `--profile ml` in Docker Compose).
- **Protobufs**: Root `proto/` directory, with generated Go/TS code.
- **Database**: `database/migrations/` for schema migrations.

## Kiosk micro-frontend architecture

The kiosk uses **Module Federation** (`@originjs/vite-plugin-federation`). The host shell consumes independently-deployable remotes:

| App | Package | Port | Purpose |
|-----|---------|------|---------|
| `kiosk-shell` | `@astra/kiosk-shell` | 5170 | Host shell, PWA, SW |
| `kiosk-menu` | `@astra/kiosk-menu` | 5171 | Menu browse + item detail |
| `kiosk-cart` | `@astra/kiosk-cart` | 5172 | Cart review |
| `kiosk-payment` | `@astra/kiosk-payment` | 5173 | Payment methods + auth |
| `kiosk-admin` | `@astra/kiosk-admin` | 5174 | Employee admin panel |
| `kiosk` | `@astra/kiosk` | 5180 | Unified (all-in-one) build |

**Important**: The unified `@astra/kiosk` app and the federated `@astra/kiosk-shell` app coexist. The unified build is a standalone deploy; the shell+remotes is for independent hotfix delivery.

Shared federation deps: `react`, `react-dom`, `zustand`, `@tanstack/react-query` (shell) — plus `valtio` (unified).

## Tailwind CSS version split

- **All apps** (`apps/*`) use **Tailwind CSS 4** with `@tailwindcss/vite` plugin.
- **`@astra/design-system`** package uses **Tailwind CSS 3** via PostCSS config (`postcss.config.js`, `tailwind.config.ts`).
- **`@astra/design-tokens`** package is pure TS + CSS variables — no Tailwind dependency.

Do NOT assume Tailwind 4 features work in the design-system package.

## State management

- **XState v5** for the kiosk workflow machine (`kioskMachine.ts`). Uses `fromPromise` actors for async operations. States: ATTRACT, MENU, ITEM_DETAIL, CART, PAYMENT, PROCESSING, RECEIPT, ADMIN.
- **Zustand** for ephemeral UI state (bottom sheet open, scroll position, search query).
- **TanStack Query** for server state (menu API, inventory) with stale-while-revalidate + optimistic updates.
- **Valtio** available as extra option (used by `@astra/kiosk-state`).

## Key commands

All commands run from `astra-service/` unless noted.

```bash
pnpm dev                          # Run all apps in parallel (turbo)
pnpm typecheck                    # tsc -b --noEmit per package
pnpm lint                         # turbo run lint (ESLint per package)
pnpm test                         # turbo run test (Vitest + happy-dom)
pnpm test:e2e                     # Playwright E2E tests
pnpm format && pnpm format:check  # Prettier across all TS/TSX/MD/JSON/YAML
pnpm build                        # turbo run build
pnpm clean                        # Remove all dist/ + node_modules
pnpm prepare                      # Install lefthook hooks
```

**Important**: `lint` → `typecheck` → `test` order matters. Typecheck and test both depend on `^build` (upstream dependencies build first).

### Run single package

```bash
pnpm turbo run dev --filter=@astra/kiosk
pnpm turbo run typecheck --filter=@astra/kiosk
pnpm turbo run test --filter=@astra/kiosk
pnpm turbo run test:e2e --filter=@astra/kiosk
```

### Go

```bash
cd astra-service/services
go test -race ./...
```

Or for a specific service:
```bash
cd astra-service/services/gateway
go test -race ./...
go vet ./...
```

### Rust

```bash
cd astra-service/sync-daemon
cargo test
cargo clippy -- -D warnings
cargo fmt --check

cd astra-service/daemons/payment-sidecar
cargo test
```

### Nix

```bash
nix flake check                  # Verify toolchain versions
nix develop                      # Enter dev shell (Node 22, Go 1.22, Rust 1.79)
```

## CI pipeline

Path-filtered: only relevant language toolchains run based on changed paths (see `.github/workflows/ci.yml` `dorny/paths-filter`). Order: lint → test-unit → test-integration → build-docker → sbom → release.

- `pnpm turbo run lint typecheck --concurrency=100%` runs lint + typecheck simultaneously
- Integration tests spin up `postgres`, `redis`, `nats` via Docker Compose
- E2E tests install Playwright browsers and run `test:e2e` on filtered packages
- Security audit (npm audit, cargo audit, govulncheck, Trivy) is non-blocking
- Docker images are built for linux/amd64+arm64, pushed to GHCR, signed with cosign, SBOM attested
- Chaos tests (`vars.ASTRA_RUN_CHAOS == 'true'`) are optional

## Testing quirks

- **Vitest** with `happy-dom` (not jsdom) — no full browser API. E2E uses Playwright.
- **Setup file**: `apps/*/src/test-utils/setup.ts` each app.
- **Go tests**: Use `-race` flag. No infrastructure required for unit tests.
- **Rust tests**: `sync-daemon` needs `protoc` for build (prost). CI uses `arduino/setup-protoc@v3`.
- **Integration tests**: Require Docker running with `postgres`, `redis`, `nats` containers.

## Service worker

Injected via `vite-plugin-pwa` with `injectManifest` strategy. Source in each app's `src/workers/service-worker.ts`. Uses Workbox Background Sync for offline queue resilience. Registered on `load` event after first paint.

## Lint / format tooling

- **ESLint** (flat config, `eslint.config.js`) for TS/TSX
- **Biome** (`biome.json`) for additional formatting (VCS-enabled, respects `.gitignore`)
- **Prettier** (root `.prettierrc` + `astra-service/.prettierrc.json`) for final formatting pass
- **gofmt** for Go (CI checks `test -z "$(gofmt -l .)"`)
- **cargo fmt** + **clippy** for Rust

## Lefthook (pre-commit)

Run `pnpm prepare` to install hooks. The current `lefthook.yml` is a commented-out template — hooks are not yet active. If you add hooks, uncomment the template entries.

## Important gotchas

- `@astra/design-system` has a `build` script that uses `del` + `rd` + `copy` (Windows commands in the package.json). This will NOT work on macOS/Linux — the package likely relies on its `dist/` being prebuilt or built via CI. If you need to build it locally, fix the script.
- `verbatimModuleSyntax: true` in tsconfig — use `import type` for type-only imports.
- `exactOptionalPropertyTypes: true` — be careful with optional fields.
- Module Federation remotes run on separate ports. All 4 remotes + shell must be running for the federated setup to work.
- `apps/kiosk` (unified) does NOT use federation — it's a standalone bundle.
- The `docker-compose.yml` uses `golang:1.25-alpine` but the flake specifies Go 1.22. CI uses Go 1.25. The Rust CI version (1.82) also differs from flake (1.79). When in doubt, match CI versions.
- ML service (`ml-lane-intel`) requires `--profile ml` flag on `docker compose`.
- Rust sync daemon requires `--profile sync` flag on `docker compose` and `protoc` for local builds.
- Commit messages must follow [Conventional Commits](https://www.conventionalcommits.org/).
