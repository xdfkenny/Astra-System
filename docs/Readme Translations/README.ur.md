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
  <a href="./README.de.md">Deutsch</a> ·
  <a href="./README.ur.md"><b>اردو</b></a> ·
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

> پروڈکشن گریڈ، آف لائن فرسٹ خودکار سیلف چیک آؤٹ پلیٹ فارم جو 24/7 خوردہ ماحول کے لیے انجینئر کیا گیا ہے۔

**Astra-System** ایک کثیر لسانی مونو ریپو ہے جو بغیر نگرانی اور نگرانی والے سیلف چیک آؤٹ کیوسک کو چلاتا ہے۔ یہ **48 گھنٹے کے آف لائن ریزیلئنس** کے ساتھ زیرو ڈاؤن ٹائم اسٹور آپریشن فراہم کرتا ہے، ایک زیرو ٹرسٹ سیکیورٹی ماڈل، اور ایک پیئر ٹو پیئر میش سنک لیئر جو اسٹور میں ہر کیوسک کو مسلسل رکھتا ہے — جب تک کہ کلاؤڈ نہ پہنچ سکے۔

---

## فہرست مضامین

- [تفصیل](#overview)
- [اہم خصوصیات](#key-features)
- [فن تعمیر](#architecture)
- [ٹیکنالوجی اسٹیک](#technology-stack)
- [ریپو کی ترتیب](#repository-layout)
- [شروع کریں](#getting-started)
- [ڈیوپلوپمنٹ ورک فلو](#development-workflow)
- [بلڈ اور رن](#build--run)
- [ٹیسٹنگ](#testing)
- [دستاویزات](#documentation)
- [شراکت داری](#contributing)
- [لائسنس](#license)

---

## تفصیل

Astra-System ریٹیلرز کو **48 گھنٹے تک آٹonomous** چلنے والے سیلف چیک آؤٹ کیوسکس کے بیڑے deploy کرنے کی اجازت دیتا ہے بغیر انٹرنیٹ کنکٹیوٹی۔ ریزیلئنس تین پرتوں میں تقسیم کی گئی ہے:

1. **لوکل ڈیٹا لیئر** — ہر کیوسک پر ایک خفیہ SQLite (SQLCipher) اسٹور جس میں مکمل مینو کیتالاگ، انوینٹری، زیر التوا لین دین، اور آف لائن پیمنٹ ٹوکنز ہیں۔
2. **پیئر ٹو پیئر میش** — کیوسکس لوکل نیٹ ورک (mDNS + libp2p/QUIC) کے ذریعے ایک دوسرے کو دریافت کرتے ہیں اور CRDTs کا استعمال کرتے ہوئے حالت کی نقل کرتے ہیں، جب تین یا اس سے زیادہ موجود ہوں تو Raft لیڈر منتخب کرتے ہیں۔
3. **خوشگوار گراوٹ** — ادائیگیاں، انوینٹری، اور آرڈر کیپچر مقامی طور پر جاری رہتے ہیں اور کنکٹیوٹی واپس آنے پر کلاؤڈ سے مطابقت پذیر ہوتے ہیں۔

کلاؤڈ ٹئیر (Goمائیکرو سروسز، PostgreSQL 16، Redis 7، NATS JetStream) مختاری ایونٹ سورسڈ اسٹور، سیٹلمنٹ، اور فلیٹ مینجمنٹ فراہم کرتی ہے۔

### ڈیزائن اہداف

| مقصد | ہدف |
| ------------------ | ---------------------------------------------------------------------- |
| آف لائن ریزیلئنس | 48 گھنٹے کا آٹonomous آپریشن بغیر کلاؤڈ کنکٹیوٹی |
| لیٹنسی | مینو لوڈ < 200 ملی سیکنڈ، P2P انوینٹری سنک < 500 ملی سیکنڈ، لیڈر فیلوور < 3 سیکنڈ |
| دستیابی | 99.99% آپ ٹائم (کلاؤڈ ٹئیر)؛ 100% آپ ٹائم صرف لوکل موڈ کے دوران |
| سیکیورٹی | زیرو ٹرسٹ، ہر جگہ mTLS، PCI-DSS تطابقی پیمنٹ پاتھ |
| اسکیل | 1–10,000 کیوسک فی ٹیننٹ؛ کثیر علاقائی کلاؤڈ ڈیپلومنٹ |

---

## اہم خصوصیات

- **آف لائن فرسٹ انجن** — م deterministik CRDT مرج (PN-Counter، LWW-Register، OR-Set) ہائبرج لاجیکل کلاکس کے ساتھ کیوسکس میں تسبیبی ترتیب کے لیے۔
- **P2P میش اور Raft консенсус** — libp2p QUIC ٹرانسپورٹ، Noise پروٹوکول خفیہ کاری، اور 3 سیکنڈ سے کم لیڈر فیلوور۔
- **ٹرانزیکشنل آٹ باکس** — NATS JetStream کے ذریعے کلاؤڈ سروسز سے بالکل ایک بار ایونٹ Publish۔
- **زیرو ٹرسٹ سیکیورٹی** — mTLS، فی کیوسک HMAC سائننگ، SPIFFE شناختیں، اور PCI-DSS تطابقی پیمنٹ پاتھ (کارڈ ڈیٹا کبھی کیوسک میموری کو نہیں چھوتا)۔
- **Verifone FFI بریج** — وندر C SDK کے اوپر ایک محفوظ Rust واپر (`astra-verifone-ffi`) پیمنٹ ٹرمینل انٹیگریشن کے لیے۔
- **بائوفیلک کیوسک UI** — React 19مائیکرو فرنٹ اینڈ جو Module Federation، XState v5 ورک فلو مشین، اور Zustand/TanStack Query اسٹیٹ مینجمنٹ کے ساتھ بنایا گیا ہے۔
- **ایڈوانسڈ انٹیلیجنس** — Ghost Carts، پیداوار کی شناخت (ONNX)، لین انٹیلیجنس (TFLite)، WebAuthn/passkeys، اور ڈفرنشل پرائیویسی اینالٹکس۔
- **کیوس آ CI** — انٹیگریشن ٹیسٹس کے دوران نیٹ ورک پارٹیشنز Inject کیے جاتے ہیں تاکہ ریزیلئنس، CRDT convergence، اور پیمنٹ queueing کی تصدیق کی جائے۔
- **کثیر لسانی کیوسک UI** — صارفین سیشن کی شروعات میں 17+ معاون زبانوں میں سے اپنی پسندیدہ زبان منتخب کرتے ہیں (انگریزی، اسپینش، چینی، فرانسیسی، جاپانی، کوریائی، ہندی، عربی، پرتگالی، روسی، بنگالی، جرمن، اردو، ٹرکی، روایتی چینی، ویتنامی، تھائی، اور مزید)۔ تمام UI ٹیکسٹ، رسیدز، اور آڈیو پرامپٹس منتخب مقامی زبان میں Render ہوتے ہیں۔

---

## فن تعمیر

Astra-System کو **کلاؤڈ ٹئیر** اور **اسٹور ایج / کیوسک کلسٹر** میں تقسیم کیا گیا ہے۔

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

مکمل ٹوپولوجی، سیکیورٹی ماڈل، پیمنٹ فلو، مشاہدتیت، اور ڈیزاسٹر ریکوری کی تفصیلات کے لیے [`ARCHITECTURE.md`](../../ARCHITECTURE.md) دیکھیں۔

### سروس انوینٹری

| سروس | زبان | ذمہ داری |
| ----------------- | ---------- | ------------------------------------------------ |
| `api-gateway` | Go | ایج روٹنگ، authN/authZ، ریٹ لمٹنگ |
| `order-svc` | Go | آرڈر لائف سائیکل، کارٹ پرسسٹنس، فل فلمنٹ |
| `payment-svc` | Go | پیمنٹ آرکسٹریشن، ٹوکن سیٹلمنٹ |
| `inventory-svc` | Go | اسٹاک لیولز، سافٹ ہولڈز، کیتالاگ سنک |
| `cart-svc` | Go | کارٹ CRDT مرج، گھوسٹ کارٹ ریزولوشن |
| `sync-svc` | Go | کلاؤڈ سائیڈ میش گیٹوے اور بیچ انگیشن |
| `astra-syncd` | Rust | کیوسک P2P ڈیمون، CRDT سنک، Verifone FFI بریج |
| `kiosk-shell` | TypeScript | React 19 صارف UI، پیریفرل انٹیگریشن |
| `update-server` | Go | سائنڈ OTA مینیفیسٹ ڈلیوری |

---

## ٹیکنالوجی اسٹیک

- **فرنٹ اینڈ** — TypeScript، React 19، Vite، Module Federation، XState v5، Zustand، TanStack Query، Tailwind CSS (apps میں v4، ڈیزائن سسٹم میں v3)۔
- **بیک اینڈ** — Go (Fiber / gRPC)، PostgreSQL 16، Redis 7، NATS JetStream۔
- **ایج** — Rust (`astra-syncd`، `astra-verifone-ffi`)، SQLite (SQLCipher)، libp2p۔
- **ایم ایل** — ONNX Runtime، TensorFlow Lite۔
- **انفراسٹرکچر** — Kubernetes، Docker / Podman، Traefik، HashiCorp Vault، Nix flake۔
- **مشاہدتیت** — Prometheus، Grafana، Loki، Jaeger، OpenTelemetry۔

---

## ریپو کی ترتیب

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

## شروع کریں

### ضروریات

- **Node.js 22** اور **pnpm 9+**
- **Go 1.25**
- **Rust 1.82** (سنک ڈیمون بنانے کے لیے `protoc` کے ساتھ)
- **Docker** اور **Docker Compose**
- *(اختیاری)* **Nix** مکمل طور پر تکرار پذیر ٹول چین کے لیے:

  ```bash
  nix develop
  ```

### تیز شروع

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

سروسز چلانے سے پہلے `.env.example` کو `.env` میں کاپی کریں اور قدریں ضرورت کے مطابق تبدیل کریں۔

---

### انسٹالر

macOS، Linux، اور Windows کے لیے پہلے سے بنائے گئے ٹیسٹ binaries [ریلیزس صفحہ](https://github.com/xdfkenny/Astra-System/releases) پر دستیاب ہیں۔

| پلیٹ فارم |ائنری |
| --------------- | ------------------------------- |
| macOS (Intel) | `astra-installer-darwin-amd64` |
| macOS (Apple Silicon) | `astra-installer-darwin-arm64` |
| Linux (x86_64) | `astra-installer-linux-amd64` |
| Linux (ARM64) | `astra-installer-linux-arm64` |
| Windows (x86_64) | `astra-installer-windows-amd64.exe` |

```bash
# macOS / Linux — download and run the bootstrap script
curl -sL https://raw.githubusercontent.com/xdfkenny/Astra-System/main/installer/scripts/install.sh | bash

# Or download a binary directly from Releases, make it executable, and run:
./astra-installer-<platform>
```

---

## ڈیوپلوپمنٹ ورک فلو

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

Turborepo فلٹرز کے ذریعے ایک سنگل پیکج چلائیں:

```bash
pnpm turbo run dev --filter=@astra/kiosk
pnpm turbo run test --filter=@astra/kiosk
```

### Go سروسز

```bash
cd astra-service/services
go test -race ./...
go vet ./...
```

### Rust ڈیمونز

```bash
cd astra-service/sync-daemon
cargo test
cargo clippy -- -D warnings
cargo fmt --check
```

---

## بلڈ اور رن

### پروٹو بفر جنریشن

```bash
cd proto
buf generate        # or: protoc as documented in proto/README.md
```

### مقامی مکمل سٹیک

```bash
docker compose up -d
pnpm dev            # kiosk-shell hot reload
```

پروڈکشن مینیفیسٹس کے لیے `docker-compose.prod.yml` استعمال کریں۔

---

## ٹیسٹنگ

| پرٹ | ٹولز |
| ------------ | -------------------------------------------------- |
| یونٹ (TS) | Vitest + happy-dom |
| E2E (TS) | Playwright `kiosk-shell` کے خلاف |
| یونٹ (Go) | `go test -race ./...` |
| یونٹ (Rust) | `cargo test`، `cargo clippy` |
| انٹیگریشن | Docker Compose stack (PostgreSQL, Redis, NATS) |
| کیوس | انٹیگریشن کے دوران نیٹ ورک پارٹیشن ایجیشن |

> انٹیگریشن اور کیوس ٹیسٹس کو `postgres`، `redis`، اور `nats` کنٹینرز کے ساتھ چلنے والے Docker کی ضرورت ہے۔

---

## دستاویزات

مکمل دستاویزات [`docs/`](../../docs/) میں دستیاب ہیں:

| سیکشن | مواد |
|---------|----------|
| **فن تعمیر** | [تفصیل](../../docs/architecture/overview.md)، [سسٹم ڈیزائن](../../docs/architecture/system-design.md)، [آف لائن فرسٹ حکمت عملی](../../docs/architecture/offline-first.md)، [سیکیورٹی ماڈل](../../docs/architecture/security-model.md) |
| **بیک اینڈ** | [مائیکرو سروسز](../../docs/backend/microservices.md)، [API گیٹوے](../../docs/backend/api-gateway.md)، [REST API](../../docs/backend/rest-api.md)، [gRPC API](../../docs/backend/grpc-api.md)، [پیمنٹ آرکسٹریٹر](../../docs/backend/payment-orchestrator.md) |
| **فرنٹ اینڈ** | [مائیکرو فرنٹ اینڈز](../../docs/frontend/micro-frontends.md)، [کیوسک ایپس](../../docs/frontend/kiosk-apps.md)، [اسٹیٹ مینجمنٹ](../../docs/frontend/state-management.md) |
| **ڈیٹابیس** | [اسکیما](../../docs/database/schema.md)، [مائیگریشنز](../../docs/database/migrations.md)، [انٹیٹیز](../../docs/database/entities.md) |
| **انفراسٹرکچر** | [Docker](../../docs/infrastructure/docker.md)، [Kubernetes](../../docs/infrastructure/kubernetes.md)، [مشاہدتیت](../../docs/infrastructure/monitoring.md)، [CI/CD](../../docs/infrastructure/ci-cd.md) |
| **نیٹ ورکنگ** | [P2P میش](../../docs/networking/p2p-mesh.md)، [پروٹوکولز](../../docs/networking/protocols.md) |
| **سیکیورٹی** | [تفصیل](../../docs/security/overview.md)، [توثیق](../../docs/security/authentication.md)، [خفیہ کاری](../../docs/security/encryption.md) |

اہم حوالجات:
- [`ARCHITECTURE.md`](../../ARCHITECTURE.md) — سسٹم ڈیزائن، سیکیورٹی ماڈل، پیمنٹ فلو، مشاہدتیت، اور ڈیزاسٹر ریکوری۔
- [`UX_UI_AUDIT_REPORT.md`](../../astra-service/UX_UI_AUDIT_REPORT.md) — "لائیوگ ویو" بائوفیلک کیوسک UI ڈیزائن سپیسیفیکیشن۔
- [`docs/API-BACKEND-ASTRA.md`](../../docs/API-BACKEND-ASTRA.md) — مکمل API اینڈ پوائنٹ انوینٹری۔
- [`docs/Readme Translations/`](../../docs/Readme Translations/) — 17+ زبانوں میں کمیونٹی کے شراکت داروں کی README ترجمے۔
- [`docs/runbooks/`](../../docs/runbooks/) — آپریشنل رن بکس (حادثہ ریسپانس، آف لائن موڈ، P2P ریکوری، پیمنٹ ناکامی)۔

---

## شراکت داری

1. تمام коммит پیغامات کے لیے [Conventional Commits](https://www.conventionalcommits.org/) کی پیروی کریں۔
2. Lefthook pre-commit hooks انٹال کرنے کے لیے `pnpm prepare` چلائیں۔
3. پل ریکوئسٹ کھولنے سے پہلے یقینی بنائیں کہ `lint ← typecheck ← test` سب گزر جاتے ہیں۔
4. تبدیلیاں path-scoped رکھیں؛ CI path-filtered ہے اور صرف متعلقہ ٹول چینز چلاتا ہے۔

---

## لائسنس

[Apache لائسنس، ورژن 2.0](../../LICENSE) کے تحت جاری۔

---

<p align="center">
  <sub>Astra-System · آف لائن-پہلے خریداری کے لیے تیار۔</sub>
</p>
