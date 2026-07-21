# Astra-System

<p align="center">
  <img src="https://raw.githubusercontent.com/cat-milk/Anime-Girls-Holding-Programming-Books/master/Typescript/Beako_Reading_The_TypeScript_Programming_Language.png" width="420" alt="Anime girl reading the TypeScript programming language book" />
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
  <a href="./README.bn.md"><b>বাংলা</b></a> ·
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

[![CI](https://img.shields.io/badge/CI-passing-green.svg)](https://github.com/xdfkenny/Astra-System/actions)
[![License](https://img.shields.io/badge/license-Apache%202.0-blue.svg)](../../LICENSE)
[![Go](https://img.shields.io/badge/Go-1.25-00ADD8.svg)](https://go.dev/dl/)
[![Rust](https://img.shields.io/badge/Rust-1.82-dea584.svg)](https://www.rust-lang.org/tools/install)
[![TypeScript](https://img.shields.io/badge/TypeScript-5.7-3178C6.svg)](https://www.typescriptlang.org/download)

> প্রোডাকশন-গ্রেড, অফলাইন-প্রথম স্বয়ংক্রিয় সেলফ-চেকআউট প্ল্যাটফর্ম যা ২৪/৭ খুচরা পরিবেশের জন্য ইঞ্জিনিয়ার করা হয়েছে।

**Astra-System** একটি বহুভাষিক মনোরেপো যা তত্ত্বাবধানহীন এবং তত্ত্বাবধানযুক্ত সেলফ-চেকআউট কিয়স্ক চালিত করে। এটি **৪৮ ঘণ্টার অফলাইন সহনশীলতা** সহ জিরো-ডাউনটাইম দোকান পরিচালনা প্রদান করে, একটি জিরো-ট্রাস্ট নিরাপত্তা মডেল, এবং একটি পিয়ার-টু-পিয়ার মেশ সিঙ্ক লেয়ার যা প্রতিটি কিয়স্ককে দোকানে সুসংগত রাখে — এমনকি যখন ক্লাউড অনুপলব্ধ থাকে।

---

## সূচিপত্র

- [সারসংক্ষেপ](#সারসংক্ষেপ)
- [মূল বৈশিষ্ট্য](#মূল-বৈশিষ্ট্য)
- [আর্কিটেকচার](#আর্কিটেকচার)
- [প্রযুক্তি স্ট্যাক](#প্রযুক্তি-স্ট্যাক)
- [রিপোজিটরি লেআউট](#রিপোজিটরি-লেআউট)
- [কিভাবে শুরু করবেন](#কিভাবে-শুরু-করবেন)
- [উন্নয়ন কার্যপ্রবাহ](#উন্নয়ন-কার্যপ্রবাহ)
- [বিল্ড এবং রান](#বিল্ড-এবং-রান)
- [পরীক্ষা](#পরীক্ষা)
- [নথিপত্র](#নথিপত্র)
- [অবদান](#অবদান)
- [লাইসেন্স](#লাইসেন্স)

---

## সারসংক্ষেপ

Astra-System খুচরা বিক্রেতাদের এমন সেলফ-চেকআউট কিয়স্কের বহর মোতায়েন করতে সক্ষম করে যা ইন্টারনেট সংযোগ ছাড়াই **৪৮ ঘণ্টা পর্যন্ত স্বায়ত্তশাসিতভাবে পরিচালিত** হতে পারে। সহনশীলতা তিনটি স্তরে সাজানো:

1. **স্থানীয় ডেটা স্তর** — প্রতিটি কিয়স্কে একটি এনক্রিপ্টেড SQLite (SQLCipher) স্টোর যাতে সম্পূর্ণ মেনু ক্যাটালগ, ইনভেন্টরি, মুলতুবি লেনদেন এবং অফলাইন পেমেন্ট টোকেন রয়েছে।
2. **পিয়ার-টু-পিয়ার মেশ** — কিয়স্কগুলো স্থানীয় নেটওয়ার্কে একে অপরকে সনাক্ত করে (mDNS + libp2p/QUIC) এবং CRDTs ব্যবহার করে অবস্থা প্রতিলিপি করে, তিন বা ততোধিক উপস্থিত থাকলে Raft নেতা নির্বাচন করে।
3. **মৃদু অবনতি** — পেমেন্ট, ইনভেন্টরি এবং অর্ডার ক্যাপচার স্থানীয়ভাবে চালিয়ে যায় এবং সংযোগ ফিরে এলে ক্লাউডের সাথে সমন্বয় করে।

ক্লাউড টিয়ার (Go মাইক্রোসার্ভিস, PostgreSQL 16, Redis 7, NATS JetStream) কর্তৃক প্রমাণিত ইভেন্ট-সোর্সড স্টোর, সেটেলমেন্ট এবং ফ্লিট ব্যবস্থাপনা প্রদান করে।

### ডিজাইন লক্ষ্য

| লক্ষ্য               | লক্ষ্য                                                                 |
| ------------------- | ---------------------------------------------------------------------- |
| অফলাইন সহনশীলতা     | ক্লাউড সংযোগ ছাড়াই ৪৮ ঘণ্টার স্বায়ত্তশাসিত পরিচালনা                  |
| বিলম্ব              | < 200 ms মেনু লোড, < 500 ms P2P ইনভেন্টরি সিঙ্ক, < 3 s নেতা ফেইলওভার   |
| উপলব্ধতা            | 99.99% আপটাইম (ক্লাউড টিয়ার); স্থানীয়-শুধু মোডে 100% আপটাইম           |
| নিরাপত্তা           | জিরো ট্রাস্ট, সর্বত্র mTLS, PCI-DSS সম্মত পেমেন্ট পথ                   |
| স্কেল              | ১–১০,০০০ কিয়স্ক প্রতি টেনেন্ট; মাল্টি-রিজিয়ন ক্লাউড মোতায়েন           |

---

## মূল বৈশিষ্ট্য

- **অফলাইন-প্রথম ইঞ্জিন** — নির্ধারিত CRDT মিশ্রণ (PN-Counter, LWW-Register, OR-Set) কিয়স্ক জুড়ে কারণাগত ক্রমের জন্য Hybrid Logical Clocks সহ।
- **P2P মেশ এবং Raft সম্মতি** — libp2p QUIC ট্রান্সপোর্ট, Noise প্রোটোকল এনক্রিপশন এবং ৩ সেকেন্ডের কম নেতা ফেইলওভার।
- **ট্রানজাকশনাল আউটবক্স** — NATS JetStream এর মাধ্যমে ক্লাউড সার্ভিস থেকে এক্সাক্ট-ওয়ান্স ইভেন্ট প্রকাশন।
- **জিরো-ট্রাস্ট নিরাপত্তা** — mTLS, প্রতি-কিয়স্ক HMAC সাইনিং, SPIFFE পরিচয় এবং PCI-DSS সম্মত পেমেন্ট পথ (কার্ড ডেটা কখনই কিয়স্ক মেমরিস্পর্শ করে না)।
- **Verifone FFI ব্রিজ** — পেমেন্ট টার্মিনাল ইন্টিগ্রেশনের জন্য ভেন্ডর C SDK উপর একটি নিরাপদ Rust র‍্যাপার (`astra-verifone-ffi`)।
- **বায়োফিলিক কিয়স্ক UI** — React 19 মাইক্রো-ফ্রন্টএন্ড যা Module Federation, XState v5 ওয়ার্কফ্লো মেশিন এবং Zustand/TanStack Query স্টেট ব্যবস্থাপনা দিয়ে তৈরি।
- **উন্নত বুদ্ধিমত্তা** — Ghost Carts, উৎপাদ সনাক্তকরণ (ONNX), লেইন ইন্টেলিজেন্স (TFLite), WebAuthn/passkeys এবং ডিফারেনশিয়াল-প্রাইভেসি অ্যানালিটিক্স।
- **কেইস-প্রস্তুত CI** — ইন্টিগ্রেশন পরীক্ষার সময় সহনশীলতা, CRDT কনভার্জেন্স এবং পেমেন্ট কিউয়িং যাচাই করার জন্য নেটওয়ার্ক পার্টিশন ইনজেক্ট করা হয়।
- **বহুভাষিক কিয়স্ক UI** — গ্রাহকরা সেশনের শুরুতে ১৭+ সমর্থিত ভাষা থেকে তাদের পছন্দের ভাষা নির্বাচন করে (English, Spanish, Chinese, French, Japanese, Korean, Hindi, Arabic, Portuguese, Russian, Bengali, German, Urdu, Turkish, Traditional Chinese, Vietnamese, Thai এবং আরও)। সমস্ত UI টেক্সট, রসিদ এবং অডিয়ো প্রম্পট নির্বাচিত লোকেলে রেন্ডার হয়।

---

## আর্কিটেকচার

Astra-System **ক্লাউড টিয়ার** এবং **স্টোর এজ / কিয়স্ক ক্লাস্টার** এ বিভক্ত।

```text
┌─────────────────────────────────────────────────────────────────┐
│                         Cloud Tier                              │
│  API Gateway · Order Svc · Payment Svc · Inventory Svc ·       │
│  Cart Svc · Sync Svc · PostgreSQL 16 · Redis 7 · NATS JetStream │
└──────────────────────────────────┬──────────────────────────────┘
                                   │ TLS 1.3
┌──────────────────────────────────┴──────────────────────────────┐
│                    Store Edge / Kiosk Cluster                   │
│  Kiosk 1 ──┐   Kiosk 2 ──┐   Kiosk N ──┐                       │
│  React 19  │   React 19  │   React 19  │  (local mesh QUIC)    │
│  Rust P2P  │   Rust P2P  │   Rust P2P  │                       │
│  SQLite    │   SQLite    │   SQLite    │                       │
│  Verifone · Printer · Scanner · NFC/Scale                      │
└─────────────────────────────────────────────────────────────────┘
```

সম্পূর্ণ টোপোলজি, নিরাপত্তা মডেল, পেমেন্ট প্রবাহ, অবজারভেবিলিটি এবং ডিজাস্টার-রিকাভারি বিস্তারিতের জন্য, [`ARCHITECTURE.md`](./ARCHITECTURE.md) দেখুন।

### সার্ভিস ইনভেন্টরি

| সার্ভিস          | ভাষা      | দায়িত্ব                                           |
| ----------------- | ---------- | ------------------------------------------------ |
| `api-gateway`     | Go         | এজ রাউটিং, authN/authZ, রেট লিমিটিং               |
| `order-svc`       | Go         | অর্ডার লাইফসাইকল, কার্ট পারসিস্টেন্স, ফুলফিলমেন্ট |
| `payment-svc`     | Go         | পেমেন্ট অরকেস্ট্রেশন, টোকেন সেটেলমেন্ট            |
| `inventory-svc`   | Go         | স্টক লেভেল, সফ্ট হোল্ড, ক্যাটালগ সিঙ্ক             |
| `cart-svc`        | Go         | কার্ট CRDT মিশ্রণ, ghost-cart সমাধান               |
| `sync-svc`        | Go         | ক্লাউড-সাইড মেশ গেটওয়ে এবং ব্যাচ ইনজেশন          |
| `astra-syncd`     | Rust       | কিয়স্ক P2P ডেমন, CRDT সিঙ্ক, Verifone FFI ব্রিজ    |
| `kiosk-shell`     | TypeScript | React 19 গ্রাহক UI, পেরিফেরাল ইন্টিগ্রেশন          |
| `update-server`   | Go         | সাইন্ড OTA ম্যানিফেস্ট ডেলিভারি                    |

---

## প্রযুক্তি স্ট্যাক

- **ফ্রন্টএন্ড** — TypeScript, React 19, Vite, Module Federation, XState v5, Zustand, TanStack Query, Tailwind CSS (apps-এ v4, ডিজাইন সিস্টেমে v3)।
- **ব্যাকএন্ড** — Go (Fiber / gRPC), PostgreSQL 16, Redis 7, NATS JetStream।
- **এজ** — Rust (`astra-syncd`, `astra-verifone-ffi`), SQLite (SQLCipher), libp2p।
- **ML** — ONNX Runtime, TensorFlow Lite।
- **ইনফ্রা** — Kubernetes, Docker / Podman, Traefik, HashiCorp Vault, Nix flake।
- **অবজারভেবিলিটি** — Prometheus, Grafana, Loki, Jaeger, OpenTelemetry।

---

## রিপোজিটরি লেআউট

```text
astra-service/          Service and application code
  apps/                 TypeScript micro-frontends (kiosk-shell, kiosk-menu, …)
  packages/             Shared libraries and design system
  services/             Go microservices
  sync-daemon/          astra-syncd (Rust) P2P daemon
  daemons/              Sidecar daemons (payment-sidecar)
  tools/                Operational tooling (chaos, etc.)
services/               Standalone services (update-server, …)
database/               Schema migrations
proto/                  Protocol Buffer definitions and generated code
docs/                   Operational runbooks
infra/                  Infrastructure tooling and secrets helpers
.github/                CI workflows and community files
flake.nix               Reproducible Nix dev shell
docker-compose*.yml     Local and production compose manifests
```

---

## কিভাবে শুরু করবেন

### পূর্বশর্ত

- **Node.js 22** এবং **pnpm 9+**
- **Go 1.25**
- **Rust 1.82** (সিঙ্ক ডেমন তৈরি করতে `protoc` সহ)
- **Docker** এবং **Docker Compose**
- *(ঐচ্ছিক)* সম্পূর্ণ পুনরুৎপাদনযোগ্য টুলচেইনের জন্য **Nix**:

  ```bash
  nix develop
  ```

### দ্রুত শুরু

```bash
# 1. Install frontend dependencies
pnpm install

# 2. Bring up the local backend stack (PostgreSQL, Redis, NATS)
docker compose up -d

# 3. Run all TypeScript apps with hot reload
pnpm dev

# 4. Build the Rust sync daemon
cd astra-service/sync-daemon && cargo build --release
```

সার্ভিস চালানোর আগে `.env.example` কে `.env` এ কপি করুন এবং প্রয়োজন অনুযায়ী মান সামঞ্জস্য করুন।

---

### ইনস্টলার

macOS, Linux এবং Windows-এর জন্য পূর্ব-নির্মিত পরীক্ষামূলক বাইনারি [Releases পৃষ্ঠায়](https://github.com/xdfkenny/Astra-System/releases) পাওয়া যায়।

| প্ল্যাটফর্ম        | বাইনারি                         |
| --------------- | ------------------------------- |
| macOS (Intel)   | `astra-installer-darwin-amd64`  |
| macOS (Apple Silicon) | `astra-installer-darwin-arm64` |
| Linux (x86_64)  | `astra-installer-linux-amd64`   |
| Linux (ARM64)   | `astra-installer-linux-arm64`   |
| Windows (x86_64)| `astra-installer-windows-amd64.exe` |

```bash
# macOS / Linux — download and run the bootstrap script
curl -sL https://raw.githubusercontent.com/xdfkenny/Astra-System/main/installer/scripts/install.sh | bash

# Or download a binary directly from Releases, make it executable, and run:
./astra-installer-<platform>
```

---

## উন্নয়ন কার্যপ্রবাহ

```bash
# Lint, typecheck, and test (order matters)
pnpm lint
pnpm typecheck
pnpm test

# End-to-end tests (Playwright)
pnpm test:e2e

# Format
pnpm format && pnpm format:check

# Build all packages
pnpm build
```

Turborepo ফিল্টার দিয়ে একক প্যাকেজ চালান:

```bash
pnpm turbo run dev --filter=@astra/kiosk
pnpm turbo run test --filter=@astra/kiosk
```

### Go সার্ভিস

```bash
cd astra-service/services
go test -race ./...
go vet ./...
```

### Rust ডেমন

```bash
cd astra-service/sync-daemon
cargo test
cargo clippy -- -D warnings
cargo fmt --check
```

---

## বিল্ড এবং রান

### Protobuf জেনারেশন

```bash
cd proto
buf generate        # or: protoc as documented in proto/README.md
```

### স্থানীয় সম্পূর্ণ স্ট্যাক

```bash
docker compose up -d
pnpm dev            # kiosk-shell hot reload
```

প্রোডাকশন ম্যানিফেস্টের জন্য, `docker-compose.prod.yml` ব্যবহার করুন।

---

## পরীক্ষা

| স্তর        | টুলিং                                            |
| ------------ | -------------------------------------------------- |
| ইউনিট (TS)    | Vitest + happy-dom                                 |
| E2E (TS)     | Playwright `kiosk-shell` এর বিপরীতে               |
| ইউনিট (Go)    | `go test -race ./...`                              |
| ইউনিট (Rust)  | `cargo test`, `cargo clippy`                       |
| ইন্টিগ্রেশন  | Docker Compose স্ট্যাক (PostgreSQL, Redis, NATS)   |
| কেইস         | ইন্টিগ্রেশনের সময় নেটওয়ার্ক-পার্টিশন ইনজেকশন     |

> ইন্টিগ্রেশন এবং কেইস পরীক্ষার জন্য `postgres`, `redis` এবং `nats` কন্টেইনার চালানো Docker প্রয়োজন।

---

## নথিপত্র

সম্পূর্ণ নথিপত্র [`docs/`](./docs/)-তে পাওয়া যায়:

| বিভাগ | বিষয়বস্তু |
|---------|----------|
| **আর্কিটেকচার** | [সারসংক্ষেপ](./docs/architecture/overview.md), [সিস্টেম ডিজাইন](./docs/architecture/system-design.md), [অফলাইন-প্রথম কৌশল](./docs/architecture/offline-first.md), [নিরাপত্তা মডেল](./docs/architecture/security-model.md) |
| **ব্যাকএন্ড** | [মাইক্রোসার্ভিস](./docs/backend/microservices.md), [API গেটওয়ে](./docs/backend/api-gateway.md), [REST API](./docs/backend/rest-api.md), [gRPC API](./docs/backend/grpc-api.md), [পেমেন্ট অরকেস্ট্রেটর](./docs/backend/payment-orchestrator.md) |
| **ফ্রন্টএন্ড** | [মাইক্রো-ফ্রন্টএন্ড](./docs/frontend/micro-frontends.md), [কিয়স্ক অ্যাপস](./docs/frontend/kiosk-apps.md), [স্টেট ব্যবস্থাপনা](./docs/frontend/state-management.md) |
| **ডাটাবেস** | [স্কিমা](./docs/database/schema.md), [মাইগ্রেশন](./docs/database/migrations.md), [এন্টিটি](./docs/database/entities.md) |
| **অবকাঠামো** | [Docker](./docs/infrastructure/docker.md), [Kubernetes](./docs/infrastructure/kubernetes.md), [অবজারভেবিলিটি](./docs/infrastructure/monitoring.md), [CI/CD](./docs/infrastructure/ci-cd.md) |
| **নেটওয়ার্কিং** | [P2P মেশ](./docs/networking/p2p-mesh.md), [প্রোটোকল](./docs/networking/protocols.md) |
| **নিরাপত্তা** | [সারসংক্ষেপ](./docs/security/overview.md), [প্রমাণীকরণ](./docs/security/authentication.md), [এনক্রিপশন](./docs/security/encryption.md) |

মূল রেফারেন্স:
- [`ARCHITECTURE.md`](./ARCHITECTURE.md) — সিস্টেম ডিজাইন, নিরাপত্তা মডেল, পেমেন্ট প্রবাহ, অবজারভেবিলিটি এবং DR।
- [`UX_UI_AUDIT_REPORT.md`](./astra-service/UX_UI_AUDIT_REPORT.md) — "Living Weave" বায়োফিলিক কিয়স্ক UI ডিজাইন স্পেসিফিকেশন।
- [`docs/API-BACKEND-ASTRA.md`](./docs/API-BACKEND-ASTRA.md) — সম্পূর্ণ API এন্ডপয়েন্ট ইনভেন্টরি।
- [`docs/Readme Translations/`](./docs/Readme Translations/) — ১৭+ ভাষায় কমিউনিটি-অবদানিত README অনুবাদ।
- [`docs/runbooks/`](./docs/runbooks/) — অপারেশনাল রানবুক (দুর্ঘটনা প্রতিক্রিয়া, অফলাইন মোড, P2P রিকাভারি, পেমেন্ট ব্যর্থতা)।

---

## অবদান

1. সমস্ত কমিট বার্তার জন্য [Conventional Commits](https://www.conventionalcommits.org/) অনুসরণ করুন।
2. Lefthook প্রি-কমিট হুক ইনস্টল করতে `pnpm prepare` চালান।
3. পুল অনুরোধ খোলার আগে নিশ্চিত করুন যে `lint → typecheck → test` সব পাস করেছে।
4. পরিবর্তনগুলো পাথ-স্কোপ্ড রাখুন; CI পাথ-ফিল্টার্ড এবং শুধুমাত্র প্রাসঙ্গিক টুলচেইন চালায়।

---

## লাইসেন্স

[Apache License, Version 2.0](../../LICENSE)-এর অধীনে লাইসেন্সপ্রাপ্ত।

---

<p align="center">
  <sub>Astra-System · স্থিতিশীল, অফলাইন-প্রথম খুচরা বিক্রির জন্য তৈরি।</sub>
</p>
