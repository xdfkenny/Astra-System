# Astra-pat

<p align="center">
  <img src="https://raw.githubusercontent.com/cat-milk/Anime-Girls-Holding-Programming-Books/master/Typescript/Beako_Reading_The_TypeScript_Programming_Language.png" width="420" alt="Be' wa' DIvI' Hol TypeScript laDtaH" />
</p>

<p align="center">
  <a href="../../README.md">English</a> ·
  <a href="./README.es.md">Español</a> ·
  <a href="./README.zh.md">中文</a> ·
  <a href="./README.fr.md">Français</a>
  <br>
  <sub>
  <a href="./README.ja.md">日本語</a> ·
  <a href="./README.ko.md">한국어</a> ·
  <a href="./README.hi.md">हिन्दी</a> ·
  <a href="./README.ar.md">العربية</a> ·
  <a href="./README.pt.md">Português</a> ·
  <a href="./README.ru.md">Русский</a> ·
  <a href="./README.bn.md">বাংলা</a> ·
  <a href="./README.de.md">Deutsch</a> ·
  <a href="./README.ur.md">اردو</a> ·
  <a href="./README.tr.md">Türkçe</a> ·
  <a href="./README.zh-TW.md">繁體中文</a> ·
  <a href="./README.vi.md">Tiếng Việt</a> ·
  <a href="./README.th.md">ไทย</a> ·
  <a href="./README.la.md">Latina</a> ·
  <a href="./README.tlh.md">tlhIngan Hol</a> ·
  <a href="./README.tlh.md"><b>tlhIngan Hol</b></a>
  </sub>
</p>

[![CI](https://img.shields.io/badge/CI-passing-green.svg)](https://github.com/xdfkenny/Astra-System/actions)
[![License](https://img.shields.io/badge/license-Apache%202.0-blue.svg)](../../LICENSE)
[![Go](https://img.shields.io/badge/Go-1.25-00ADD8.svg)](https://go.dev/dl/)
[![Rust](https://img.shields.io/badge/Rust-1.82-dea584.svg)](https://www.rust-lang.org/tools/install)
[![TypeScript](https://img.shields.io/badge/TypeScript-5.7-3178C6.svg)](https://www.typescriptlang.org/download)

> SuvwI' po' pat. QaD law' tlhIngan maH. 24/7 yInHa' qem.

**Astra-pat** maH. wa' Hol pol pat. QIHbe'qu' pat. 48 rep QaD. ghobmey QIHHa'. DujDu' DajatlhlaHbe'chugh QoQ.

---

## mIch

- [mIwHur](#mIwhur)
- [Qu'mey](#qumey)
- [patwI'](#patwi)
- [QaH pat](#qah-pat)
- [pat nay'](#pat-nay)
- [Qapla'](#qapla)
- [chenmoh](#chenmoh)
- [qorDu'](#qordu)
- [veS qeD](#ves-qed)
- [ghItlhwI'](#ghithlwi)
- [ra'tlh](#rathl)

---

## mIwHur

Astra-pat DajatlhlaHbe'bogh tentIv ghajbogh qorDu' DujDu' DaHjaj Qap. **48 rep QaD** ghaj. ghobmey QIHHa' ghaj. DujDu' DajatlhlaHbe'chugh, yInHa' qem pat.

1. **nay' QaD** — SQLCipher pat. menu' Daj, nav, Huch, DajatlhlaHbe'bogh Huch mIw, Dujvam.
2. **Duj Dajatlh** — DujDu' mDNS lo'laH. libp2p/QUIC lo'laH. CRDT lo'laH. Raft loD cha' DujDu' HeDtaHvIS.
3. **QaD QIHHa'** — Huch, nav, mIw QIHHa'. QoQ DajatlhlaHmeH pat QoQ yInHa' qem.

QoD pat (Go, PostgreSQL 16, Redis 7, NATS JetStream) So', QoQ pat, Huch, je DujDu' mIw.

### Qu' QapmeH

| Qu'              | 'ay'                                                                 |
| ---------------- | -------------------------------------------------------------------- |
| QaD QIHHa'       | 48 rep QoQ DajatlhlaHbe'chugh yIn                                  |
| QaQ law'          | < 200 ms menu' laD, < 500 ms Duj Dajatlh, < 3 s loD choH            |
| tagha'            | 99.99% QoD pat; 100% Duj DuQoQ DajatlhlaHbe'chugh                   |
| QaD               | mTLS Hoch, PCI-DSS Huch QIHHa', ghobmey QIHHa'                       |
| rap               | 1–10,000 Duj qorDu' wa'; QoD pat DoD law'                           |

---

## Qu'mey

- **QaD QIHHa' Qu'** — CRDT lo'laH. PN-Counter, LWW-Register, OR-Set. HLC lo'laH. DujDu' rap.
- **Duj Dajatlh je Raft** — libp2p QUIC lo'laH. Noise lo'laH. 3 lup loD choH.
- **wa'logh QIn** — Hoch mIw wa'logh QIn. NATS JetStream lo'laH.
- **ghobmey QIHHa'** — mTLS. HMAC Hoch Duj. SPIFFE lo'laH. PCI-DSS Huch QIHHa'. Huch Duj pa' Qotlhbe'.
- **Verifone FFI** — Rust po' pat. Verifone C SDK po' pat.
- **Duj QaD QIHHa'** — React 19. XState v5. Zustand. TanStack Query. Module Federation.
- **QaD SuvwI'** — Ghost Carts, 'o'wI' (ONNX), Duj QaD (ONNX), WebAuthn/passkeys, differential-privacy janHut.
- **veS potlh** — veS qeD. veS Qap vIS. CRDT rap, Huch Doj, QIHHa'.

---

## patwI'

Astra-pat cha': **QoD pat** je **Duj pat**.

```text
┌─────────────────────────────────────────────────────────────────┐
│                         QoD pat                                  │
│  API Gateway · Order Svc · Payment Svc · Inventory Svc ·       │
│  Cart Svc · Sync Svc · PostgreSQL 16 · Redis 7 · NATS JetStream │
└──────────────────────────────────┬──────────────────────────────┘
                                   │ TLS 1.3
┌──────────────────────────────────┴──────────────────────────────┐
│                   qorDu' Duj pat                                 │
│  Duj 1 ──┐   Duj 2 ──┐   Duj N ──┐                              │
│  React 19│   React 19│   React 19│  (Duj Dajatlh QUIC)          │
│  Rust P2P│   Rust P2P│   Rust P2P│                              │
│  SQLite  │   SQLite  │   SQLite  │                              │
│  Verifone · wej'uch · jIH · NFC/wIj                              │
└─────────────────────────────────────────────────────────────────┘
```

QIch ra'ma', ghobmey QIHHa', Huch mIv, je janHut: tu' [`ARCHITECTURE.md`](../../ARCHITECTURE.md).

### pat SuvwI'

| pat              | Hol       | Qu'                                                |
| ---------------- | --------- | -------------------------------------------------- |
| `api-gateway`    | Go        | QIch ra', authN/authZ, rate limiting               |
| `order-svc`      | Go        | QIn, Duj, mIw QIHHa'                               |
| `payment-svc`    | Go        | Huch, Huch token, So'                              |
| `inventory-svc`  | Go        | nav, HoS, catalog Dajatlh                          |
| `cart-svc`       | Go        | CRDT DujDuj, Ghost Cart rap                        |
| `sync-svc`       | Go        | Duj Dajatlh QoD pat, DoD QIn                      |
| `astra-syncd`    | Rust      | Duj Dajatlh, CRDT Dajatlh, Verifone FFI            |
| `kiosk-shell`    | TypeScript| React 19 QaD QIHHa', jan Duj                       |
| `update-server`  | Go        | OTA QoDna' loD                                      |

---

## QaH pat

- **QaD QIHHa'** — TypeScript, React 19, Vite, Module Federation, XState v5, Zustand, TanStack Query, Tailwind CSS.
- **qa'pat** — Go (Fiber / gRPC), PostgreSQL 16, Redis 7, NATS JetStream.
- **Duj** — Rust (`astra-syncd`, `astra-verifone-ffi`), SQLite (SQLCipher), libp2p.
- **ML** — ONNX Runtime, TensorFlow Lite.
- **nengwI'** — Kubernetes, Docker / Podman, Traefik, HashiCorp Vault, Nix flake.
- **janHut** — Prometheus, Grafana, Loki, Jaeger, OpenTelemetry.

---

## pat nay'

```text
astra-service/          pat je QaD QIHHa' code
  apps/                 TypeScript micro-frontends (kiosk-shell, kiosk-menu, …)
  packages/             QaH libraries je design system
  services/             Go microservices
  sync-daemon/          astra-syncd (Rust) Duj Dajatlh
  daemons/              Sidecar daemons (payment-sidecar)
  tools/                Operational tooling (chaos, etc.)
services/               Standalone services (update-server, …)
database/               Schema migrations
proto/                  Protocol Buffer QIn je generated code
docs/                   Operational runbooks
infra/                  Infrastructure tooling je secrets
.github/                CI workflows je community files
flake.nix               Reproducible Nix dev shell
docker-compose*.yml     Local je production compose manifests
```

---

## Qapla'

### Qang

- **Node.js 22** je **pnpm 9+**
- **Go 1.25**
- **Rust 1.82** (protoc lo'laH)
- **Docker** je **Docker Compose**
- *(lo'laHbe'chugh)* **Nix**:

  ```bash
  nix develop
  ```

### Qap Qapla'

```bash
# 1. Hoch frontend Daput
pnpm install

# 2. QoD pat qol (PostgreSQL, Redis, NATS)
docker compose up -d

# 3. Hoch TypeScript apps Qap je hot reload
pnpm dev

# 4. Rust Duj Dajatlh chenmoH
cd astra-service/sync-daemon && cargo build --release
```

`.env.example` ra' `.env` DaghItlh. Dajatlhlu'meH, Daput.

---

### Installer

macOS, Linux, je Windows vaD binary test pre-built 'oH [Releases page](https://github.com/xdfkenny/Astra-System/releases) 'e' tu'laH.

| platform        | binary                          |
| --------------- | ------------------------------- |
| macOS (Intel)   | `astra-installer-darwin-amd64`  |
| macOS (Apple Silicon) | `astra-installer-darwin-arm64` |
| Linux (x86_64)  | `astra-installer-linux-amd64`   |
| Linux (ARM64)   | `astra-installer-linux-arm64`   |
| Windows (x86_64)| `astra-installer-windows-amd64.exe` |

```bash
# macOS / Linux — bootstrap script chu'wI' DachenmoH
curl -sL https://raw.githubusercontent.com/xdfkenny/Astra-System/main/installer/scripts/install.sh | bash

# Releasevo' binary DachenmoH, runmoHmeH chen, vaj run
./astra-installer-<platform>
```

---

## chenmoh

```bash
# Lint, typecheck, test (mIw Hur)
pnpm lint
pnpm typecheck
pnpm test

# End-to-end qeD (Playwright)
pnpm test:e2e

# Format
pnpm format && pnpm format:check

# Hoch pak Qap
pnpm build
```

Turborepo filter lo'laH:

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

## qorDu'

### Protobuf QIn chenmoH

```bash
cd proto
buf generate        # protoc lo'laH proto/README.md DaghItlh
```

### Hoch pat Qap

```bash
docker compose up -d
pnpm dev            # kiosk-shell hot reload
```

`docker-compose.prod.yml` lo'laH.

---

## veS qeD

| 'ay'        | QaH                                                  |
| ----------- | ---------------------------------------------------- |
| Unit (TS)   | Vitest + happy-dom                                   |
| E2E (TS)    | Playwright — `kiosk-shell`                           |
| Unit (Go)   | `go test -race ./...`                                |
| Unit (Rust) | `cargo test`, `cargo clippy`                         |
| Integration | Docker Compose pat (PostgreSQL, Redis, NATS)         |
| veS         | veS Qap vIS integration qeD                              |

> Integration je veS qeD: Docker Qap. `postgres`, `redis`, je `nats` containers.

---

## Documentation

Full documentation is available in [`docs/`](../../docs/):

| Section | Contents |
|---------|----------|
| **Architecture** | [Overview](../../docs/architecture/overview.md), [System Design](../../docs/architecture/system-design.md), [Offline-First Strategy](../../docs/architecture/offline-first.md), [Security Model](../../docs/architecture/security-model.md) |
| **Backend** | [Microservices](../../docs/backend/microservices.md), [API Gateway](../../docs/backend/api-gateway.md), [REST API](../../docs/backend/rest-api.md), [gRPC API](../../docs/backend/grpc-api.md), [Payment Orchestrator](../../docs/backend/payment-orchestrator.md) |
| **Frontend** | [Micro-Frontends](../../docs/frontend/micro-frontends.md), [Kiosk Apps](../../docs/frontend/kiosk-apps.md), [State Management](../../docs/frontend/state-management.md) |
| **Database** | [Schema](../../docs/database/schema.md), [Migrations](../../docs/database/migrations.md), [Entities](../../docs/database/entities.md) |
| **Infrastructure** | [Docker](../../docs/infrastructure/docker.md), [Kubernetes](../../docs/infrastructure/kubernetes.md), [Observability](../../docs/infrastructure/monitoring.md), [CI/CD](../../docs/infrastructure/ci-cd.md) |
| **Networking** | [P2P Mesh](../../docs/networking/p2p-mesh.md), [Protocols](../../docs/networking/protocols.md) |
| **Security** | [Overview](../../docs/security/overview.md), [Authentication](../../docs/security/authentication.md), [Encryption](../../docs/security/encryption.md) |

Key references:
- [`ARCHITECTURE.md`](../../ARCHITECTURE.md) — system design, security model, payment flows, observability, and DR.
- [`UX_UI_AUDIT_REPORT.md`](../../astra-service/UX_UI_AUDIT_REPORT.md) — the "Living Weave" biophilic kiosk UI design specification.
- [`docs/API-BACKEND-ASTRA.md`](../../docs/API-BACKEND-ASTRA.md) — complete API endpoint inventory.
- [`docs/runbooks/`](../../docs/runbooks/) — operational runbooks (incident response, offline mode, P2P recovery, payment failure).

---

## ra'tlh

1. [Conventional Commits](https://www.conventionalcommits.org/) lo'.
2. `pnpm prepare` lo'laH — Lefthook pre-commit hooks.
3. `lint → typecheck → test` Hoch Qap. vaj Duj QoD Qap.
4. ra' ghom Hoch.

---

## ra'tlh

[Apache License, Version 2.0](../../LICENSE).

---

<p align="center">
  <sub>Astra-pat · QaD qorDu', QIHHa'be'. Qapla'!</sub>
</p>

<p align="center">
  <sub>🖖 <b>tlhIngan maH!</b> taHjaj wo'rIv!</sub>
</p>
