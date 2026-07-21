# Astra-System

<p align="center">
  <img src="https://raw.githubusercontent.com/cat-milk/Anime-Girls-Holding-Programming-Books/master/Typescript/Beako_Reading_The_TypeScript_Programming_Language.png" width="420" alt="Anime girl reading the TypeScript programming language book" />
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
  <a href="./README.de.md"><b>Deutsch</b></a> ·
  <a href="./README.ur.md">اردو</a> ·
  <a href="./README.tr.md">Türkçe</a> ·
  <a href="./README.zh-TW.md">繁體中文</a> ·
  <a href="./README.vi.md">Tiếng Việt</a> ·
  <a href="./README.th.md">ไทย</a> ·
  <a href="./README.la.md">Latina</a> ·
  <a href="./README.tlh.md">tlhIngan Hol</a>
  </sub>
</p>

[![CI](https://img.shields.io/badge/CI-passing-green.svg)](https://github.com/xdfkenny/Astra-System/actions)
[![License](https://img.shields.io/badge/license-Apache%202.0-blue.svg)](../../LICENSE)
[![Go](https://img.shields.io/badge/Go-1.25-00ADD8.svg)](https://go.dev/dl/)
[![Rust](https://img.shields.io/badge/Rust-1.82-dea584.svg)](https://www.rust-lang.org/tools/install)
[![TypeScript](https://img.shields.io/badge/TypeScript-5.7-3178C6.svg)](https://www.typescriptlang.org/download)

> Produktionsreife, offline-first automatisierte Self-Checkout-Plattform, entwickelt für 24/7-Einzelhandelsumgebungen.

**Astra-System** ist ein mehrsprachiges Monorepo, das unbewachte und bewachte Self-Checkout-Kioske betreibt. Es ermöglicht eine ausfallsichere Ladenoperation mit **48 Stunden Offline-Resilienz**, einem Zero-Trust-Sicherheitsmodell und einer Peer-to-Peer-Mesh-Synchronisationsschicht, die jeden Kiosk im Geschäft konsistent hält — auch wenn die Cloud nicht erreichbar ist.

---

## Inhaltsverzeichnis

- [Übersicht](#übersicht)
- [Hauptfunktionen](#hauptfunktionen)
- [Architektur](#architektur)
- [Technologie-Stack](#technologie-stack)
- [Repository-Struktur](#repository-struktur)
- [Erste Schritte](#erste-schritte)
- [Entwicklungsablauf](#entwicklungsablauf)
- [Build & Ausführung](#build--ausführung)
- [Tests](#tests)
- [Dokumentation](#dokumentation)
- [Beitragen](#beitragen)
- [Lizenz](#lizenz)

---

## Übersicht

Astra-System ermöglicht es Einzelhändlern, Flotten von Self-Checkout-Kiosken einzusetzen, die **bis zu 48 Stunden autonom** ohne Internetverbindung arbeiten. Die Resilienz ist über drei Ebenen gestapelt:

1. **Lokale Datenebene** — ein verschlüsselter SQLite-Speicher (SQLCipher) auf jedem Kiosk mit dem vollständigen Menükatalog, Bestand, ausstehenden Transaktionen und Offline-Zahlungstoken.
2. **Peer-to-Peer-Mesh** — Kioske erkennen sich gegenseitig über das lokale Netzwerk (mDNS + libp2p/QUIC) und replizieren den Zustand mittels CRDTs, wobei bei drei oder mehr Kiosken ein Raft-Leader gewählt wird.
3. **Graceful Degradation** — Zahlungen, Bestand und Auftragsannahme werden lokal fortgesetzt und mit der Cloud abgeglichen, wenn die Konnektivität wiederhergestellt ist.

Die Cloud-Ebene (Go-Microservices, PostgreSQL 16, Redis 7, NATS JetStream) bietet den maßgeblichen ereignisgesteuerten Speicher, die Abwicklung und das Flottenmanagement.

### Design-Ziele

| Ziel | Zielwert |
| --- | --- |
| Offline-Resilienz | 48 Stunden autonomer Betrieb ohne Cloud-Konnektivität |
| Latenz | < 200 ms Menüladezeit, < 500 ms P2P-Bestandssync, < 3 s Leader-Failover |
| Verfügbarkeit | 99,99 % Verfügbarkeit (Cloud-Ebene); 100 % Verfügbarkeit im Nur-Lokal-Modus |
| Sicherheit | Zero Trust, mTLS überall, PCI-DSS-konformer Zahlungspfad |
| Skalierung | 1–10.000 Kiosks pro Mandant; Cloud-Deployment in mehreren Regionen |

---

## Hauptfunktionen

- **Offline-First-Engine** — deterministische CRDT-Fusion (PN-Counter, LWW-Register, OR-Set) mit Hybriden Logischen Uhren für kausale Reihenfolge zwischen Kiosken.
- **P2P-Mesh & Raft-Konsens** — libp2p-QUIC-Transport, Noise-Protocol-Verschlüsselung und Leader-Failover unter 3 Sekunden.
- **Transaktionaler Outbox** — exakt-einmalige Event-Veröffentlichung aus Cloud-Diensten via NATS JetStream.
- **Zero-Trust-Sicherheit** — mTLS, HMAC-Signierung pro Kiosk, SPIFFE-Identitäten und ein PCI-DSS-konformer Zahlungspfad (Kartendaten berühren niemals den Kiosk-Speicher).
- **Verifone-FFI-Brücke** — ein sicherer Rust-Wrapper über das Hersteller-C-SDK für die Integration von Zahlungsterminals.
- **Biophile Kiosk-UI** — ein React 19-Micro-Frontend mit Module Federation, XState-v5-Workflow-Maschine und Zustand/TanStack-Query-Zustandsverwaltung.
- **Erweiterte Intelligenz** — Ghost Carts,-produkterkennung (ONNX), Fahrbahnenanalytik (TFLite), WebAuthn/Passkeys und differentiell-private Analytik.
- **Chaos-fähige CI** — Netzwerkpartitionen werden während Integrationstests injiziert, um Resilienz, CRDT-Konvergenz und Zahlungswarteschlangen zu verifizieren.
- **Mehrsprachige Kiosk-UI** — Kunden wählen zu Beginn ihrer Sitzung ihre bevorzugte Sprache aus über 17 unterstützten Sprachen. Alle UI-Texte, Quittungen und Audioaufforderungen werden in der gewählten Sprache dargestellt.

---

## Architektur

Astra-System ist in eine **Cloud-Ebene** und einen **Store-Edge-/Kiosk-Cluster** aufgeteilt.

Für die vollständige Topologie, das Sicherheitsmodell, Zahlungsflüsse, Observability und Disaster-Recovery-Details siehe [`ARCHITECTURE.md`](../../ARCHITECTURE.md).

### Service-Übersicht

| Service | Sprache | Verantwortung |
| --- | --- | --- |
| `api-gateway` | Go | Edge-Routing, AuthN/AuthZ, Rate-Limiting |
| `order-svc` | Go | Auftragslebenszyklus, Warenkorb, Fulfillment |
| `payment-svc` | Go | Zahlungsorchestrierung, Token-Abwicklung |
| `inventory-svc` | Go | Bestandsniveaus, Soft-Holds, Katalog-Sync |
| `cart-svc` | Go | Warenkorb-CRDT-Fusion, Ghost-Cart-Auflösung |
| `sync-svc` | Go | Cloud-seitiges Mesh-Gateway und Batch-Ingestion |
| `astra-syncd` | Rust | Kiosk-P2P-Daemon, CRDT-Sync, Verifone-FFI-Brücke |
| `kiosk-shell` | TypeScript | React 19-Kunden-UI, Peripherie-Integration |
| `update-server` | Go | Signierte OTA-Manifest-Bereitstellung |

---

## Technologie-Stack

- **Frontend** — TypeScript, React 19, Vite, Module Federation, XState v5, Zustand, TanStack Query, Tailwind CSS.
- **Backend** — Go (Fiber / gRPC), PostgreSQL 16, Redis 7, NATS JetStream.
- **Edge** — Rust (`astra-syncd`, `astra-verifone-ffi`), SQLite (SQLCipher), libp2p.
- **ML** — ONNX Runtime, TensorFlow Lite.
- **Infra** — Kubernetes, Docker / Podman, Traefik, HashiCorp Vault, Nix flake.
- **Observability** — Prometheus, Grafana, Loki, Jaeger, OpenTelemetry.

---

## Repository-Struktur

```text
astra-service/          Service- und Anwendungscode
  apps/                 TypeScript-Micro-Frontends (kiosk-shell, kiosk-menu, …)
  packages/             Geteilte Bibliothek und Design System
  services/             Go-Microservices
  sync-daemon/          astra-syncd (Rust) P2P-Daemon
  daemons/              Sidecar-Daemons (payment-sidecar)
  tools/                Operationswerkzeuge (Chaos, etc.)
services/               Eigenständige Services (update-server, …)
database/               Schema-Migrationen
proto/                  Protocol-Buffer-Definitionen und generierter Code
docs/                   Operative Runbooks
infra/                  Infrastrukturwerkzeuge und Secrets-Helfer
.github/                CI-Workflows und Community-Dateien
flake.nix               Reproduzierbare Nix-Entwicklungsumgebung
docker-compose*.yml     Lokale und Produktions-Manifeste
```

---

## Erste Schritte

### Voraussetzungen

- **Node.js 22** und **pnpm 9+**
- **Go 1.25**
- **Rust 1.82** (mit `protoc` zum Kompilieren des Sync-Daemon)
- **Docker** und **Docker Compose**
- *(Optional)* **Nix** für eine vollständig reproduzierbare Toolchain:

  ```bash
  nix develop
  ```

### Schnellstart

```bash
# 1. Frontend-Abhängigkeiten installieren
pnpm install

# 2. Lokalen Backend-Stack starten (PostgreSQL, Redis, NATS)
docker compose up -d

# 3. Alle TypeScript-Apps mit Hot-Reload starten
pnpm dev

# 4. Rust-Sync-Daemon kompilieren
cd astra-service/sync-daemon && cargo build --release
```

Kopieren Sie `.env.example` nach `.env` und passen Sie die Werte bei Bedarf an, bevor Sie die Dienste starten.

---

### Installationsprogramm

Vorgefertigte Test-Binaries für macOS, Linux und Windows sind auf der [Releases-Seite](https://github.com/xdfkenny/Astra-System/releases) verfügbar.

| Plattform | Binary |
| --- | --- |
| macOS (Intel) | `astra-installer-darwin-amd64` |
| macOS (Apple Silicon) | `astra-installer-darwin-arm64` |
| Linux (x86_64) | `astra-installer-linux-amd64` |
| Linux (ARM64) | `astra-installer-linux-arm64` |
| Windows (x86_64) | `astra-installer-windows-amd64.exe` |

```bash
# macOS / Linux — Bootstrap-Skript herunterladen und ausführen
curl -sL https://raw.githubusercontent.com/xdfkenny/Astra-System/main/installer/scripts/install.sh | bash

# Oder ein Binary direkt aus Releases herunterladen, ausführbar machen und ausführen:
./astra-installer-<platform>
```

---

## Entwicklungsablauf

```bash
# Lint, Typecheck und Test (Reihenfolge ist wichtig)
pnpm lint
pnpm typecheck
pnpm test

# End-to-End-Tests (Playwright)
pnpm test:e2e

# Format
pnpm format && pnpm format:check

# Alle Pakete kompilieren
pnpm build
```

Einzelnes Paket über Turborepo-Filter ausführen:

```bash
pnpm turbo run dev --filter=@astra/kiosk
pnpm turbo run test --filter=@astra/kiosk
```

### Go-Dienste

```bash
cd astra-service/services
go test -race ./...
go vet ./...
```

### Rust-Daemons

```bash
cd astra-service/sync-daemon
cargo test
cargo clippy -- -D warnings
cargo fmt --check
```

---

## Build & Ausführung

### Protobuf-Generierung

```bash
cd proto
buf generate
```

### Lokaler Full-Stack

```bash
docker compose up -d
pnpm dev
```

Für Produktions-Manifeste verwenden Sie `docker-compose.prod.yml`.

---

## Tests

| Ebene | Werkzeuge |
| --- | --- |
| Unit (TS) | Vitest + happy-dom |
| E2E (TS) | Playwright gegen `kiosk-shell` |
| Unit (Go) | `go test -race ./...` |
| Unit (Rust) | `cargo test`, `cargo clippy` |
| Integration | Docker-Compose-Stack (PostgreSQL, Redis, NATS) |
| Chaos | Netzwerkpartition-Injection während der Integration |

> Integrations- und Chaos-Tests erfordern Docker mit laufenden `postgres`-, `redis`- und `nats`-Containern.

---

## Dokumentation

Vollständige Dokumentation ist in [`docs/`](../../docs/) verfügbar:

| Abschnitt | Inhalte |
| --- | --- |
| **Architektur** | [Übersicht](../../docs/architecture/overview.md), [Systemdesign](../../docs/architecture/system-design.md), [Offline-First-Strategie](../../docs/architecture/offline-first.md), [Sicherheitsmodell](../../docs/architecture/security-model.md) |
| **Backend** | [Microservices](../../docs/backend/microservices.md), [API Gateway](../../docs/backend/api-gateway.md), [REST API](../../docs/backend/rest-api.md), [gRPC API](../../docs/backend/grpc-api.md), [Zahlungsorchestrator](../../docs/backend/payment-orchestrator.md) |
| **Frontend** | [Micro-Frontends](../../docs/frontend/micro-frontends.md), [Kiosk-Apps](../../docs/frontend/kiosk-apps.md), [Zustandsverwaltung](../../docs/frontend/state-management.md) |
| **Datenbank** | [Schema](../../docs/database/schema.md), [Migrationen](../../docs/database/migrations.md), [Entitäten](../../docs/database/entities.md) |
| **Infrastruktur** | [Docker](../../docs/infrastructure/docker.md), [Kubernetes](../../docs/infrastructure/kubernetes.md), [Observability](../../docs/infrastructure/monitoring.md), [CI/CD](../../docs/infrastructure/ci-cd.md) |
| **Netzwerk** | [P2P-Mesh](../../docs/networking/p2p-mesh.md), [Protokolle](../../docs/networking/protocols.md) |
| **Sicherheit** | [Übersicht](../../docs/security/overview.md), [Authentifizierung](../../docs/security/authentication.md), [Verschlüsselung](../../docs/security/encryption.md) |

Wichtige Referenzen:
- [`ARCHITECTURE.md`](../../ARCHITECTURE.md) — Systemdesign, Sicherheitsmodell, Zahlungsflüsse, Observability und DR.
- [`UX_UI_AUDIT_REPORT.md`](../../astra-service/UX_UI_AUDIT_REPORT.md) — Die „Living Weave" biophile Kiosk-UI-Designspezifikation.
- [`docs/API-BACKEND-ASTRA.md`](../../docs/API-BACKEND-ASTRA.md) — Vollständiges API-Endpunkt-Inventar.
- [`docs/Readme Translations/`](../../docs/Readme%20Translations/) — Von der Community beigetragene README-Übersetzungen in 17+ Sprachen.
- [`docs/runbooks/`](../../docs/runbooks/) — Operative Runbooks (Vorfallsreaktion, Offline-Modus, P2P-Wiederherstellung, Zahlungsausfall).

---

## Beitragen

1. Befolgen Sie [Conventional Commits](https://www.conventionalcommits.org/) für alle Commit-Nachrichten.
2. Führen Sie `pnpm prepare` aus, um Lefthook-Pre-Commit-Hooks zu installieren.
3. Stellen Sie sicher, dass `lint → typecheck → test` bestehen, bevor Sie einen Pull Request erstellen.
4. Halten Sie Änderungen pfadbezogen; CI ist pfadgefiltert und führt nur die relevanten Toolchains aus.

---

## Lizenz

Lizenziert unter der [Apache License, Version 2.0](../../LICENSE).

---

<p align="center">
  <sub>Astra-System · Gebaut für widerstandsfähigen, offline-first Einzelhandel.</sub>
</p>
