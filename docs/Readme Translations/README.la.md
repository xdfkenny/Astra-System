# Astra-System

<p align="center">
  <img src="https://raw.githubusercontent.com/cat-milk/Anime-Girls-Holding-Programming-Books/master/Typescript/Beako_Reading_The_TypeScript_Programming_Language.png" width="420" alt="Puella animata librum de lingua TypeScript legens" />
</p>

<p align="center">
  <a href="../../README.md"><b>English</b></a> ·
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
  <a href="./README.la.md"><b>Latina</b></a>
  </sub>
</p>

[![CI](https://img.shields.io/badge/CI-passing-green.svg)](https://github.com/xdfkenny/Astra-System/actions)
[![License](https://img.shields.io/badge/license-Apache%202.0-blue.svg)](../../LICENSE)
[![Go](https://img.shields.io/badge/Go-1.25-00ADD8.svg)](https://go.dev/dl/)
[![Rust](https://img.shields.io/badge/Rust-1.82-dea584.svg)](https://www.rust-lang.org/tools/install)
[![TypeScript](https://img.shields.io/badge/TypeScript-5.7-3178C6.svg)](https://www.typescriptlang.org/download)

> Proventus gradui productivo, primum a conductione remoti, automatica taberna sui ipsius emptoriae in ambitu rei venditae XXIV/VII designata.

**Astra-System** est mono-repositorium multilingue quod terminalia emptoria sui ipsius custodita et incustodita sustinet. Praebet operationem tabernae sine intermissione cum **resilientia conductionis remoti XLVIII horarum**, exemplari securitatis nullius fiduciae, atque strato synchronizationis retis aequalis ad aequalem (P2P) quod omne terminale in taberna consonum servat — etiam cum nubes (cloud) attingi non potest.

---

## Index

- [Conspectus](#conspectus)
- [Proprietates Praecipuae](#proprietates-praecipuae)
- [Architectura](#architectura)
- [Acervus Technologicus](#acervus-technologicus)
- [Dispositio Repositorii](#dispositio-repositorii)
- [Cito Inceptum](#cito-inceptum)
- [Ratio Evolutionis](#ratio-evolutionis)
- [Aedificare et Currere](#aedificare-et-currere)
- [Probationes](#probationes)
- [Documentatio](#documentatio)
- [Conferre](#conferre)
- [Licentia](#licentia)

---

## Conspectus

Astra-System venditoribus permittit ut greges terminalium emptoriorum sui ipsius disponant quae **sine nexu interretiali usque ad XLVIII horas** autonome operantur. Resilientia per tres gradus instruitur:

1. **Stratum datum locale** — in omni terminali est thesaurus SQLite (SQLCipher) encryptus cum catalogo tabulae pleno, inventario, transactionibus pendentibus, et signis solutionis conductionis remoti.
2. **Retis aequalis ad aequalem** — terminalia inter se in reti locali (mDNS + libp2p/QUIC) reperiunt et statum per CRDT replicant, ductorem Raft eligentes cum tria vel plura adsunt.
3. **Degradatio gratiosa** — solutio, inventarium, et captura ordinis localiter pergunt et cum nube reconciliantur cum connexio redit.

Gradus nubis (microservitia Go, PostgreSQL 16, Redis 7, NATS JetStream) thesaurum auctoritativum ex eventibus fontibus, compensationem, et administrationem classis praebet.

### Proposita Designanda

| Propositum          | Metum                                                                    |
| ------------------- | ------------------------------------------------------------------------ |
| Resilientia remoti  | XLVIII horae operationis autonomae sine nexu nubis                        |
| Latentia            | Tabula < 200 ms onerata, sync inventarii P2P < 500 ms, defectus ductus < 3 s |
| Disponibilitas       | 99.99% tempus sursum (gradus nubis); 100% in modo locali solo            |
| Securitas           | Nullius fiduciae, mTLS ubique, catena solutionis conformis PCI-DSS       |
| Amplitudo           | I–X,000 terminalia per conductionem; dispositio multiregionalis nubis     |

---

## Proprietates Praecipuae

- **Motor primum a conductione remoti** — innixus commixtioni CRDT determinanti (PN-Counter, LWW-Register, OR-Set), et horologiis logicis hybridis (HLC) ad ordinem causalem inter terminalia.
- **Retis P2P et consensus Raft** — transvectio libp2p QUIC, encryptio protocollorum Noise, defectus ductus infra 3 secundas.
- **Outbox Transactionalis** — per NATS JetStream eventus microservitiorum nubis "semel exacte" divulgantur.
- **Securitas nullius fiduciae** — mTLS, signatura HMAC per terminale, identitas SPIFFE, et catena solutionis conformis PCI-DSS (data chartae numquam in memoriam terminalis intrant).
- **Pons Verifone FFI** — involucrum securum in Rust fundatum (`astra-verifone-ffi`) ad SDK venditoris C pro terminalibus solutionis integrandis.
- **UI Terminalis Biomimetica** — micro-frontes React 19 in Federatione Modulorum fundati, cum automatis status XState v5 et administratione status Zustand/TanStack Query.
- **Intelligentia Provecta** — Ghost Cart, recognitio mercis (ONNX), intelligentia canalis (TFLite), WebAuthn/Passkey, et analyticas differentiae privatae.
- **CI Paratae ad Chaos** — partitiones retis in probationibus integrationis iniectae ad resilientiam, convergentiam CRDT, et facultatem queues solutionis verificandas.

---

## Architectura

Astra-System in **gradu nubis** et **margine tabernae / grege terminalium** dividitur.

```text
┌─────────────────────────────────────────────────────────────────┐
│                         Gradus Nubis                            │
│  API Gateway · Svc Ordo · Svc Solutio · Svc Inventarium ·      │
│  Svc Cart · Svc Sync · PostgreSQL 16 · Redis 7 · NATS JetStream │
└──────────────────────────────────┬──────────────────────────────┘
                                   │ TLS 1.3
┌──────────────────────────────────┴──────────────────────────────┐
│              Margo Tabernae / Grex Terminalium                   │
│  Term. 1 ─┐   Term. 2 ─┐   Term. N ─┐                            │
│  React 19 │   React 19 │   React 19 │  (retis QUIC locale)      │
│  Rust P2P │   Rust P2P │   Rust P2P │                           │
│  SQLite   │   SQLite   │   SQLite   │                           │
│  Verifone · Impressora · Scanner · NFC/Libra                    │
└─────────────────────────────────────────────────────────────────┘
```

Topologiam plenam, exemplar securitatis, fluxum solutionis, observabilitatem, et recuperationem calamitatis vide in [`ARCHITECTURE.md`](../../ARCHITECTURE.md).

### Index Servitiorum

| Servitium          | Lingua      | Officium                                               |
| ------------------ | ----------- | ------------------------------------------------------ |
| `api-gateway`      | Go          | Route marginis, authN/authZ, moderatio fluxus          |
| `order-svc`        | Go          | Vita ordinis, persistencia cart, impletio              |
| `payment-svc`      | Go          | Orchestratio solutionis, compensatio signorum          |
| `inventory-svc`    | Go          | Gradus inventarii, claustra mollia, sync catalogi      |
| `cart-svc`         | Go          | Commixtio CRDT cart, resolutio Ghost Cart              |
| `sync-svc`         | Go          | Porta retis nubis et ingestio globi                     |
| `astra-syncd`      | Rust        | Daemon P2P terminalis, sync CRDT, pons Verifone FFI    |
| `kiosk-shell`      | TypeScript  | UI clientis React 19, integratio periphericorum        |
| `update-server`    | Go          | Distributio indicis OTA signati                        |

---

## Acervus Technologicus

- **Frons** — TypeScript, React 19, Vite, Federatio Modulorum, XState v5, Zustand, TanStack Query, Tailwind CSS (v4 in app, v3 in systemate designi).
- **Posticum** — Go (Fiber / gRPC), PostgreSQL 16, Redis 7, NATS JetStream.
- **Margo** — Rust (`astra-syncd`, `astra-verifone-ffi`), SQLite (SQLCipher), libp2p.
- **Disciplina Machine** — ONNX Runtime, TensorFlow Lite.
- **Infrastructura** — Kubernetes, Docker / Podman, Traefik, HashiCorp Vault, Nix flake.
- **Observabilitas** — Prometheus, Grafana, Loki, Jaeger, OpenTelemetry.

---

## Dispositio Repositorii

```text
astra-service/           Codex servitiorum et applicationum
  apps/                 Micro-frontes TypeScript (kiosk-shell, kiosk-menu, etc.)
  packages/             Bibliothecae communes et systema designi
  services/             Microservitia Go
  sync-daemon/          astra-syncd (Rust) daemon P2P
  daemons/              Daemones lateris (payment-sidecar)
  tools/                Instrumenta operationalia (chaos, etc.)
database/               Migrationes schemae数据库
proto/                  Definitiones et codex generatus Protocol Buffer
docs/                   Manualia operationalia
infra/                  Instrumenta infrastructurae et adiutores clavis
.github/                Fluxus CI et fasciculi communitatis
flake.nix               Circumstantia evolutionis Nix reproducibilis
docker-compose*.yml     Indices Compose localis et productivi
```

---

## Cito Inceptum

### Exigentia Ambientis

- **Node.js 22** et **pnpm 9+**
- **Go 1.25**
- **Rust 1.82** (cum `protoc` ad daemon synchronizationis aedificandum)
- **Docker** et **Docker Compose**
- *(optional)* **Nix** ad torculum plene reproducibile:

  ```bash
  nix develop
  ```

### Celeriter Surge

```bash
# 1. Dependentiis frontis instala
pnpm install

# 2. Stacam posticam localem surge (PostgreSQL, Redis, NATS)
docker compose up -d

# 3. Omnes applicationes TypeScript cum reload calido curre
pnpm dev

# 4. Daemon synchronizationis Rust aedifica
cd astra-service/sync-daemon && cargo build --release
```

Ante servitia currenda, copia `.env.example` in `.env` et secundum opus adapta.

---

### Installator

Binaria pretemptata pro macOS, Linux, et Windows praeconstructa in [pagina Releases](https://github.com/xdfkenny/Astra-System/releases) reperiuntur.

| Suggestus        | Binarium                        |
| ---------------- | ------------------------------- |
| macOS (Intel)    | `astra-installer-darwin-amd64`  |
| macOS (Apple Silicon) | `astra-installer-darwin-arm64` |
| Linux (x86_64)   | `astra-installer-linux-amd64`   |
| Linux (ARM64)    | `astra-installer-linux-arm64`   |
| Windows (x86_64) | `astra-installer-windows-amd64.exe` |

```bash
# macOS / Linux — scriptum deduc et curre
curl -sL https://raw.githubusercontent.com/xdfkenny/Astra-System/main/installer/scripts/install.sh | bash

# Aut binarium de Release prehende, exsecrabile fac, et curre:
./astra-installer-<platform>
```

---

## Ratio Evolutionis

```bash
# Lint, typum proba, et testare (ordo interest)
pnpm lint
pnpm typecheck
pnpm test

# Testae end-to-end (Playwright)
pnpm test:e2e

# Formare
pnpm format && pnpm format:check

# Omnes fasciculos aedificare
pnpm build
```

Per filtra Turborepo unicum fasciculum curre:

```bash
pnpm turbo run dev --filter=@astra/kiosk
pnpm turbo run test --filter=@astra/kiosk
```

### Servitia Go

```bash
cd astra-service/services
go test -race ./...
go vet ./...
```

### Daemones Rust

```bash
cd astra-service/sync-daemon
cargo test
cargo clippy -- -D warnings
cargo fmt --check
```

---

## Aedificare et Currere

### Generatio Protobuf

```bash
cd proto
buf generate        # vel: protoc ut in proto/README.md
```

### Staca Localis Plena

```bash
docker compose up -d
pnpm dev            # kiosk-shell cum reload calido
```

Ad indicem productivum utere `docker-compose.prod.yml`.

---

## Probationes

| Gradus            | Instrumentum                                              |
| ----------------- | --------------------------------------------------------- |
| Unitas (TS)       | Vitest + happy-dom                                        |
| End-to-end (TS)   | Playwright contra `kiosk-shell`                           |
| Unitas (Go)       | `go test -race ./...`                                      |
| Unitas (Rust)     | `cargo test`, `cargo clippy`                              |
| Integratio        | Staca Docker Compose (PostgreSQL, Redis, NATS)            |
| Chaos             | Partitiones retis in gradu integrationis iniectae         |

> Probationes integrationis et chaos Docker currentem requirunt, cum vasis `postgres`, `redis`, `nats` surgentibus.

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

## Conferre

1. Omnes inscriptiones committendarum sequantur [Conventional Commits](https://www.conventionalcommits.org/).
2. Curre `pnpm prepare` ad hamos prae-committendum Lefthook installandos.
3. Ante Pull Request mittendum, cura ut `lint → typecheck → test` omnes transeant.
4. Mutationes per viam focus serva; CI filtris viae utitur ut solum torcula relevantia currat.

---

## Licentia

Sub [Licentia Apache 2.0](LICENSE) concessa.

---

<p align="center">
  <sub>Astra-System · nata ad venditionem cum resilientia conductionis remoti.</sub>
</p>
