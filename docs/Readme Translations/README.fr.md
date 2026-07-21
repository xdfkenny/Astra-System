# Astra-System

<p align="center">
  <img src="https://raw.githubusercontent.com/cat-milk/Anime-Girls-Holding-Programming-Books/master/Typescript/Beako_Reading_The_TypeScript_Programming_Language.png" width="420" alt="Anime girl reading the TypeScript programming language book" />
</p>

<p align="center">
  <a href="../../README.md">English</a> ·
  <a href="./README.es.md">Español</a> ·
  <a href="./README.zh.md">中文</a> ·
  <a href="./README.fr.md"><b>Français</b></a>
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
  <a href="./README.tlh.md">tlhIngan Hol</a>
  </sub>
</p>

[![CI](https://img.shields.io/badge/CI-pass-green.svg)](https://github.com/anomalyco/astra-system/actions)
[![License](https://img.shields.io/badge/license-Apache%202.0-blue.svg)](../../LICENSE)
[![Go](https://img.shields.io/badge/Go-1.25-00ADD8.svg)](https://go.dev)
[![Rust](https://img.shields.io/badge/Rust-1.82-dea584.svg)](https://www.rust-lang.org)
[![TypeScript](https://img.shields.io/badge/TypeScript-5.x-3178C6.svg)](https://www.typescriptlang.org)

> Plateforme de caisse automatique autonome, de qualité production et conçue pour fonctionner hors ligne en priorité, destinée aux environnements de vente au détail 24h/24 et 7j/7.

**Astra-System** est un monorepo multilingue qui alimente des bornes de libre-service avec ou sans personnel. Il garantit un fonctionnement en magasin sans interruption avec **48 heures de résilience hors ligne**, un modèle de sécurité zéro confiance et une couche de synchronisation en maillage pair-à-pair qui maintient chaque borne cohérente, même lorsque le cloud est inaccessible.

---

## Table des Matières

- [Vue d'Ensemble](#vue-densemble)
- [Fonctionnalités Clés](#fonctionnalités-clés)
- [Architecture](#architecture)
- [Pile Technologique](#pile-technologique)
- [Structure du Répertoire](#structure-du-répertoire)
- [Pour Commencer](#pour-commencer)
- [Flux de Travail de Développement](#flux-de-travail-de-développement)
- [Compilation et Exécution](#compilation-et-exécution)
- [Tests](#tests)
- [Documentation](#documentation)
- [Contribuer](#contribuer)
- [Licence](#licence)

---

## Vue d'Ensemble

Astra-System permet aux détaillants de déployer des flottes de bornes de libre-service qui fonctionnent **de manière autonome jusqu'à 48 heures** sans connexion Internet. La résilience est répartie sur trois niveaux :

1. **Couche de données locale** — une base de données SQLite chiffrée (SQLCipher) sur chaque borne avec le catalogue complet, l'inventaire, les transactions en attente et les jetons de paiement hors ligne.
2. **Maillage pair-à-pair** — les bornes se découvrent sur le réseau local (mDNS + libp2p/QUIC) et répliquent l'état à l'aide de CRDT, en élisant un leader Raft lorsqu'au moins trois sont présentes.
3. **Dégradation élégante** — les paiements, l'inventaire et la capture des commandes se poursuivent localement et se réconcilient avec le cloud lorsque la connectivité est rétablie.

La couche cloud (microservices Go, PostgreSQL 16, Redis 7, NATS JetStream) fournit le stockage d'événements comme source de vérité, le règlement et la gestion de flotte.

### Objectifs de Conception

| Objectif              | Cible                                                                 |
| --------------------- | --------------------------------------------------------------------- |
| Résilience hors ligne | 48 heures de fonctionnement autonome sans connectivité cloud          |
| Latence               | < 200 ms chargement menu, < 500 ms sync P2P, < 3 s failover leader   |
| Disponibilité         | 99,99 % uptime (cloud) ; 100 % uptime en mode local uniquement        |
| Sécurité              | Zéro confiance, mTLS partout, chemin de paiement conforme PCI-DSS     |
| Échelle               | 1–10 000 bornes par locataire ; déploiement cloud multirégion         |

---

## Fonctionnalités Clés

- **Moteur offline-first** — fusion CRDT déterministe (PN-Counter, LWW-Register, OR-Set) avec horloges logiques hybrides pour un ordre causal entre les bornes.
- **Maillage P2P et consensus Raft** — transport libp2p QUIC, chiffrement Noise Protocol et failover du leader en moins de 3 secondes.
- **Outbox transactionnelle** — publication d'événements exactement une fois depuis les services cloud via NATS JetStream.
- **Sécurité zéro confiance** — mTLS, signature HMAC par borne, identités SPIFFE et chemin de paiement conforme PCI-DSS (les données de carte ne touchent jamais la mémoire de la borne).
- **Pont Verifone FFI** — un wrapper Rust sécurisé (`astra-verifone-ffi`) sur le SDK C du fabricant pour l'intégration des terminaux de paiement.
- **UI de borne biophilique** — micro-frontend React 19 avec Module Federation, machine d'état XState v5 et gestion d'état Zustand/TanStack Query.
- **Intelligence avancée** — Ghost Carts, reconnaissance de produits (ONNX), intelligence de file d'attente (ONNX), WebAuthn/passkeys et analytique à confidentialité différentielle.
- **CI prête pour le chaos** — des partitions réseau sont injectées pendant les tests d'intégration pour vérifier la résilience, la convergence CRDT et la mise en file d'attente des paiements.

---

## Architecture

Astra-System est divisé en une **Couche Cloud** et un **Cluster de Magasin / Bornes**.

```text
┌─────────────────────────────────────────────────────────────────┐
│                         Couche Cloud                             │
│  API Gateway · Order Svc · Payment Svc · Inventory Svc ·       │
│  Cart Svc · Sync Svc · PostgreSQL 16 · Redis 7 · NATS JetStream │
└──────────────────────────────────┬──────────────────────────────┘
                                   │ TLS 1.3
┌──────────────────────────────────┴──────────────────────────────┐
│                  Cluster de Magasin / Bornes                     │
│  Borne 1 ──┐   Borne 2 ──┐   Borne N ──┐                       │
│  React 19  │   React 19  │   React 19  │  (maillage local QUIC) │
│  Rust P2P  │   Rust P2P  │   Rust P2P  │                       │
│  SQLite    │   SQLite    │   SQLite    │                       │
│  Verifone · Imprimante · Scanner · NFC/Balance                   │
└─────────────────────────────────────────────────────────────────┘
```

Pour la topologie complète, le modèle de sécurité, les flux de paiement, l'observabilité et les détails de reprise après sinistre, voir [`ARCHITECTURE.md`](../../ARCHITECTURE.md).

### Inventaire des Services

| Service          | Langage    | Responsabilité                                  |
| ---------------- | ---------- | ----------------------------------------------- |
| `api-gateway`    | Go         | Routage périphérique, authN/authZ, limitation   |
| `order-svc`      | Go         | Cycle de vie des commandes, paniers, exécution  |
| `payment-svc`    | Go         | Orchestration des paiements, règlement des jetons|
| `inventory-svc`  | Go         | Niveaux de stock, réservations, synchronisation  |
| `cart-svc`       | Go         | Fusion CRDT des paniers, résolution Ghost Carts |
| `sync-svc`       | Go         | Passerelle maillage cloud et ingestion par lots |
| `astra-syncd`    | Rust       | Démon P2P, sync CRDT, pont Verifone FFI         |
| `kiosk-shell`    | TypeScript | Interface client React 19, intégration périph.   |
| `update-server`  | Go         | Livraison signée de manifestes OTA              |

---

## Pile Technologique

- **Frontend** — TypeScript, React 19, Vite, Module Federation, XState v5, Zustand, TanStack Query, Tailwind CSS (v4 dans les apps, v3 dans le design system).
- **Backend** — Go (Fiber / gRPC), PostgreSQL 16, Redis 7, NATS JetStream.
- **Périphérie** — Rust (`astra-syncd`, `astra-verifone-ffi`), SQLite (SQLCipher), libp2p.
- **ML** — ONNX Runtime, TensorFlow Lite.
- **Infra** — Kubernetes, Docker / Podman, Traefik, HashiCorp Vault, Nix flake.
- **Observabilité** — Prometheus, Grafana, Loki, Jaeger, OpenTelemetry.

---

## Structure du Répertoire

```text
astra-service/          Code des services et applications
  apps/                 Micro-frontends TypeScript (kiosk-shell, kiosk-menu, …)
  packages/             Bibliothèques partagées et design system
  services/             Microservices Go
  sync-daemon/          astra-syncd (Rust) démon P2P
  daemons/              Démon sidecar (payment-sidecar)
  tools/                Outils opérationnels (chaos, etc.)
services/               Services autonomes (update-server, …)
database/               Migrations de schéma
proto/                  Définitions Protocol Buffer et code généré
docs/                   Runbooks opérationnels
infra/                  Outils d'infrastructure et secrets
.github/                Workflows CI et fichiers communautaires
flake.nix               Shell de développement Nix reproductible
docker-compose*.yml     Manifests compose local et de production
```

---

## Pour Commencer

### Prérequis

- **Node.js 22** et **pnpm 9+**
- **Go 1.25**
- **Rust 1.82** (avec `protoc` pour compiler le démon de sync)
- **Docker** et **Docker Compose**
- *(Optionnel)* **Nix** pour une chaîne d'outils entièrement reproductible :

  ```bash
  nix develop
  ```

### Démarrage Rapide

```bash
# 1. Installer les dépendances frontend
pnpm install

# 2. Démarrer la pile backend locale (PostgreSQL, Redis, NATS)
docker compose up -d

# 3. Lancer toutes les apps TypeScript avec rechargement à chaud
pnpm dev

# 4. Compiler le démon de sync en Rust
cd astra-service/sync-daemon && cargo build --release
```

Copiez `.env.example` vers `.env` et ajustez les valeurs si nécessaire avant d'exécuter les services.

---

### Installateur

Les binaires de test précompilés pour macOS, Linux et Windows sont disponibles sur la [page des Releases](https://github.com/xdfkenny/Astra-System/releases).

| Plateforme       | Binaire                          |
| ---------------- | -------------------------------- |
| macOS (Intel)    | `astra-installer-darwin-amd64`   |
| macOS (Apple Silicon) | `astra-installer-darwin-arm64` |
| Linux (x86_64)   | `astra-installer-linux-amd64`    |
| Linux (ARM64)    | `astra-installer-linux-arm64`    |
| Windows (x86_64) | `astra-installer-windows-amd64.exe` |

```bash
# macOS / Linux — télécharger et exécuter le script d'amorçage
curl -sL https://raw.githubusercontent.com/xdfkenny/Astra-System/main/installer/scripts/install.sh | bash

# Ou télécharger un binaire directement depuis Releases, le rendre exécutable et l'exécuter :
./astra-installer-<platform>
```

---

## Flux de Travail de Développement

```bash
# Lint, typecheck et test (l'ordre compte)
pnpm lint
pnpm typecheck
pnpm test

# Tests de bout en bout (Playwright)
pnpm test:e2e

# Format
pnpm format && pnpm format:check

# Compiler tous les paquets
pnpm build
```

Exécutez un seul paquet via les filtres Turborepo :

```bash
pnpm turbo run dev --filter=@astra/kiosk
pnpm turbo run test --filter=@astra/kiosk
```

### Services Go

```bash
cd astra-service/services
go test -race ./...
go vet ./...
```

### Démon Rust

```bash
cd astra-service/sync-daemon
cargo test
cargo clippy -- -D warnings
cargo fmt --check
```

---

## Compilation et Exécution

### Génération Protobuf

```bash
cd proto
buf generate        # ou : protoc comme documenté dans proto/README.md
```

### Pile locale complète

```bash
docker compose up -d
pnpm dev            # rechargement à chaud kiosk-shell
```

Pour les manifests de production, utilisez `docker-compose.prod.yml`.

---

## Tests

| Couche       | Outillage                                             |
| ------------ | ----------------------------------------------------- |
| Unitaire (TS)| Vitest + happy-dom                                    |
| E2E (TS)     | Playwright contre `kiosk-shell`                       |
| Unitaire (Go)| `go test -race ./...`                                 |
| Unitaire (Rust)| `cargo test`, `cargo clippy`                        |
| Intégration  | Pile Docker Compose (PostgreSQL, Redis, NATS)          |
| Chaos        | Injection de partitions réseau pendant l'intégration  |

> Les tests d'intégration et de chaos nécessitent Docker avec les conteneurs `postgres`, `redis` et `nats` en cours d'exécution.

---

## Documentation

- [`ARCHITECTURE.md`](../../ARCHITECTURE.md) — conception du système, modèle de sécurité, flux de paiement, observabilité et DR.
- [`UX_UI_AUDIT_REPORT.md`](../../astra-service/UX_UI_AUDIT_REPORT.md) — la spécification de conception UI de borne biophilique "Living Weave".
- `proto/README.md`, `astra-service/sync-daemon/README.md` et `docs/` — sous-projets et runbooks opérationnels.

---

## Contribuer

1. Suivez [Conventional Commits](https://www.conventionalcommits.org/) pour tous les messages de commit.
2. Exécutez `pnpm prepare` pour installer les hooks pre-commit Lefthook.
3. Assurez-vous que `lint → typecheck → test` passent avant d'ouvrir une pull request.
4. Gardez les modifications limitées aux chemins concernés ; la CI est filtrée par chemin et n'exécute que les chaînes d'outils pertinentes.

---

## Licence

Sous licence [Apache License, Version 2.0](../../LICENSE).

---

<p align="center">
  <sub>Astra-System · Conçu pour un commerce de détail résilient et hors ligne.</sub>
</p>
