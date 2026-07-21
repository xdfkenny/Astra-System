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
  <a href="./README.ar.md"><b>العربية</b></a> ·
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

[![CI](https://img.shields.io/badge/CI-passing-green.svg)](https://github.com/xdfkenny/Astra-System/actions)
[![License](https://img.shields.io/badge/license-Apache%202.0-blue.svg)](../../LICENSE)
[![Go](https://img.shields.io/badge/Go-1.25-00ADD8.svg)](https://go.dev/dl/)
[![Rust](https://img.shields.io/badge/Rust-1.82-dea584.svg)](https://www.rust-lang.org/tools/install)
[![TypeScript](https://img.shields.io/badge/TypeScript-5.7-3178C6.svg)](https://www.typescriptlang.org/download)

> منصة دفع ذاتي آلي بمستوى الإنتاج مصممة للعمل دون اتصال أولاً في بيئات البيع بالتجزئة على مدار الساعة.

**Astra-System** هو مستودع واحد متعدد اللغات يدعم أكشاك الدفع الذاتي المراقبة وغير المراقبة. يوفر تشغيل متجر بدون توقف مع **48 ساعة من المرونة في وضع عدم الاتصال**، ونموذج أمان شامل، وطبقة مزامنة شبكة نظير إلى نظير تبقي كل كشك في المتجر متسقة — حتى عندما يكون السحابة غير متاحة.

---

## جدول المحتويات

- [نظرة عامة](#overview)
- [الميزات الرئيسية](#key-features)
- [البنية التحتية](#architecture)
- [تقنيات المستخدم](#technology-stack)
- [هيكل المستودع](#repository-layout)
- [البدء](#getting-started)
- [سير العمل أثناء التطوير](#development-workflow)
- [البناء والتشغيل](#build--run)
- [الاختبار](#testing)
- [التوثيق](#documentation)
- [المساهمة](#contributing)
- [الترخيص](#license)

---

## نظرة عامة

تمكّن Astra-System م retailers من نشر أسطول من أكشاك الدفع الذاتي التي تعمل **بشكل مستقل لمدة تصل إلى 48 ساعة** دون اتصال بالإنترنت. يتم توزيع المرونة عبر ثلاث طبقات:

1. **طبقة البيانات المحلية** — مستودع SQLite مشفّر (SQLCipher) على كل كشك يحتوي على كامل كتالوج القائمة، والمخزون، والمعاملات المعلقة، ورموز الدفع دون اتصال.
2. **شبكة النظير إلى النظير** — تكتشف الأكشاك بعضها البعض عبر الشبكة المحلية (mDNS + libp2p/QUIC) وتُكرّر الحالة باستخدام CRDTs، مع انتخاب قائد Raft عندما يكون هناك ثلاثة أو أكثر.
3. **التدهور الرشيق** — تستمر المدفوعات والمخزون والتقاط الطلبات محلياً وتتصالح مع السحابة عند عودة الاتصال.

توفر طبقة السحابة (خدمات Go المصغرة، PostgreSQL 16، Redis 7، NATS JetStream) المستودع الأصلي المبني على الأحداث، والتسوية، وإدارة الأسطول.

### أهداف التصميم

| الهدف | الهدف |
| ------------------ | ---------------------------------------------------------------------- |
| المرونة في وضع عدم الاتصال | 48 ساعة من التشغيل المستقل دون اتصال بالسحابة |
| زمن الاستجابة | تحميل قائمة أقل من 200 مللي ثانية، مزامنة P2P للمخزون أقل من 500 مللي ثانية، تrossover القائد أقل من 3 ثوانٍ |
| التوفر | 99.99% وقت تشغيل (طبقة السحابة)؛ 100% وقت التشغيل أثناء الوضع المحلي فقط |
| الأمان | ثقة شاملة، mTLS في كل مكان، مسار دفع متوافق مع PCI-DSS |
| النطاق | 1 إلى 10,000 كشك لكل مستأجر؛ نشر سحabi متعدد المناطق |

---

## الميزات الرئيسية

- **محرك يعمل دون اتصال أولاً** — دمج CRDT حاسم (PN-Counter، LWW-Register، OR-Set) مع ساعات منطقية هجينة للترتيب السببي عبر الأكشاك.
- **شبكة P2P واتفاق Raft** — نقل libp2p QUIC، تشفير بروتوكول Noise، وtrossover قائد أقل من 3 ثوانٍ.
- **صندوق خارجي معاملاتي** — نشر أحداث واحد مرة واحدة من الخدمات السحابية عبر NATS JetStream.
- **أمان شامل** — mTLS، توقيع HMAC لكل كشك، هويات SPIFFE، ومسار دفع متوافق مع PCI-DSS (بيانات البطاقة لا تلمس ذاكرة الكشك أبداً).
- **جسر Verifone FFI** — غلاف Rust آمن (`astra-verifone-ffi`) فوق SDK المورد C لتكامل طرفية الدفع.
- **واجهة كشك بيئية** — واجهة React 19 مصغرة مبنية باستخدام Module Federation، آلة حالة XState v5، وإدارة الحالة Zustand/TanStack Query.
- **ذكاء متقدم** — Arabots، التعرف على المنتجات (ONNX)، ذكاء المسار (TFLite)، WebAuthn/memorized keys، وتحليلات الخصوصية التفاضلية.
- **CI مستعد للفوضى** — يتم حقن انقسامات الشبكة أثناء اختبارات التكامل للتحقق من المرونة، وtrossover CRDT، و排队 المدفوعات.
- **واجهة كشك متعددة اللغات** — يختار العملاء لغتهم المفضلة في بداية الجلسة من أكثر من 17 لغة مدعومة (الإنجليزية، الإسبانية، الصينية، الفرنسية، اليابانية، الكورية، الهندية، العربية، البرتغالية، الروسية، البنغالية، الألمانية، الأردية، التركية، الصينية التقليدية، الفيتنامية، التايلندية، والمزيد). يتم عرض جميع نصوص الواجهة والإيصالات والمطالبات الصوتية باللغة المحددة.

---

## البنية التحتية

مقسمة Astra-System إلى **طبقة سحابية** و**حافة متجر / مجموعة أكشاك**.

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

لل_topology الكاملة، ونموذج الأمان، وتدفقات الدفع، وقابلية المراقبة، وتفاصيل التعافي من الكوارث، راجع [`ARCHITECTURE.md`](../../ARCHITECTURE.md).

### جرد الخدمات

| الخدمة | اللغة | المسؤولية |
| ----------------- | ---------- | ------------------------------------------------ |
| `api-gateway` | Go | التوجيه الحدودي، المصادقة/التفويض، تحديد المعدل |
| `order-svc` | Go | دورة حياة الطلب، حفظ السلة، التنفيذ |
| `payment-svc` | Go | تنسيق المدفوعات، تسوية الرموز |
| `inventory-svc` | Go | مستويات المخزون، الاحتفاظ الناعم، مزامنة الكتالوج |
| `cart-svc` | Go | دمج CRDT للسلة، حل السلة الشبحية |
| `sync-svc` | Go | بوابة الشبكة السحابية والابتلاع الدفعي |
| `astra-syncd` | Rust | عامل P2P للكشك، مزامنة CRDT، جسر Verifone FFI |
| `kiosk-shell` | TypeScript | واجهة العميل React 19، تكامل الأجهزة الطرفية |
| `update-server` | Go | تسليم ق manifests OTA موقعة |

---

## تقنيات المستخدم

- **الواجهة الأمامية** — TypeScript، React 19، Vite، Module Federation، XState v5، Zustand، TanStack Query، Tailwind CSS (v4 في التطبيقات، v3 في نظام التصميم).
- **الخلفية** — Go (Fiber / gRPC)، PostgreSQL 16، Redis 7، NATS JetStream.
- **الحافة** — Rust (`astra-syncd`، `astra-verifone-ffi`)، SQLite (SQLCipher)، libp2p.
- **التعلم الآلي** — ONNX Runtime، TensorFlow Lite.
- **البنية التحتية** — Kubernetes، Docker / Podman، Traefik، HashiCorp Vault، Nix flake.
- **قابلية المراقبة** — Prometheus، Grafana، Loki، Jaeger، OpenTelemetry.

---

## هيكل المستودع

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

## البدء

### المتطلبات الأساسية

- **Node.js 22** و **pnpm 9+**
- **Go 1.25**
- **Rust 1.82** (مع `protoc` لبناء عامل المزامنة)
- **Docker** و **Docker Compose**
- *(اختياري)* **Nix** ل トラック أداة قابل لإعادة الإنتاج:

  ```bash
  nix develop
  ```

### البدء السريع

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

قم بنسخ `.env.example` إلى `.env` وتعديل القيم حسب الحاجة قبل تشغيل الخدمات.

---

### المُثبّت

متاح اختبار binaries مسبوقاً الإنشاء لنظام macOS و Linux و Windows على [صفحة الإصدارات](https://github.com/xdfkenny/Astra-System/releases).

| المنصة | الملف التنفيذي |
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

## سير العمل أثناء التطوير

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

تشغيل حزمة واحدة عبر فلاتر Turborepo:

```bash
pnpm turbo run dev --filter=@astra/kiosk
pnpm turbo run test --filter=@astra/kiosk
```

### خدمات Go

```bash
cd astra-service/services
go test -race ./...
go vet ./...
```

### عوامل Rust

```bash
cd astra-service/sync-daemon
cargo test
cargo clippy -- -D warnings
cargo fmt --check
```

---

## البناء والتشغيل

### إنتاج Protocol Buffers

```bash
cd proto
buf generate        # or: protoc as documented in proto/README.md
```

### STACK محلي كامل

```bash
docker compose up -d
pnpm dev            # kiosk-shell hot reload
```

لقوائم الإنتاج، استخدم `docker-compose.prod.yml`.

---

## الاختبار

| الطبقة | الأدوات |
| ------------ | -------------------------------------------------- |
| الوحدة (TS) | Vitest + happy-dom |
| النهاية إلى النهاية (TS) | Playwright ضد `kiosk-shell` |
| الوحدة (Go) | `go test -race ./...` |
| الوحدة (Rust) | `cargo test`، `cargo clippy` |
| التكامل | Docker Compose stack (PostgreSQL, Redis, NATS) |
| الفوضى | حقن انقسام الشبكة أثناء التكامل |

> تتطلب اختبارات التكامل والفوضى Docker قيد التشغيل مع حاويات `postgres` و `redis` و `nats`.

---

## التوثيق

التوثيق الكامل متاح في [`docs/`](../../docs/):

| القسم | المحتويات |
|---------|----------|
| **البنية التحتية** | [نظرة عامة](../../docs/architecture/overview.md)، [تصميم النظام](../../docs/architecture/system-design.md)، [استراتيجية العمل دون اتصال](../../docs/architecture/offline-first.md)، [نموذج الأمان](../../docs/architecture/security-model.md) |
| **الخلفية** | [خدمات مصغرة](../../docs/backend/microservices.md)، [بوابة API](../../docs/backend/api-gateway.md)، [واجهة REST API](../../docs/backend/rest-api.md)، [واجهة gRPC API](../../docs/backend/grpc-api.md)، [منسق المدفوعات](../../docs/backend/payment-orchestrator.md) |
| **الواجهة الأمامية** | [واجهات مصغرة](../../docs/frontend/micro-frontends.md)، [تطبيقات الكشك](../../docs/frontend/kiosk-apps.md)، [إدارة الحالة](../../docs/frontend/state-management.md) |
| **قاعدة البيانات** | [المخطط](../../docs/database/schema.md)، [الترقيات](../../docs/database/migrations.md)، [الكيانات](../../docs/database/entities.md) |
| **البنية التحتية** | [Docker](../../docs/infrastructure/docker.md)، [Kubernetes](../../docs/infrastructure/kubernetes.md)، [المراقبة](../../docs/infrastructure/monitoring.md)، [CI/CD](../../docs/infrastructure/ci-cd.md) |
| **الشبكات** | [شبكة P2P](../../docs/networking/p2p-mesh.md)، [البروتوكولات](../../docs/networking/protocols.md) |
| **الأمان** | [نظرة عامة](../../docs/security/overview.md)، [المصادقة](../../docs/security/authentication.md)، [التشفير](../../docs/security/encryption.md) |

المراجع الرئيسية:
- [`ARCHITECTURE.md`](../../ARCHITECTURE.md) — تصميم النظام، نموذج الأمان، تدفقات الدفع، المراقبة، والتعافي من الكوارث.
- [`UX_UI_AUDIT_REPORT.md`](../../astra-service/UX_UI_AUDIT_REPORT.md) — مواصفات تصميم واجهة الكشك البيئية "النسيج الحي".
- [`docs/API-BACKEND-ASTRA.md`](../../docs/API-BACKEND-ASTRA.md) — جرد شامل لنقاط API.
- [`docs/Readme Translations/`](../../docs/Readme Translations/) — ترجمات README مساهمة من المجتمع بأكثر من 17 لغة.
- [`docs/runbooks/`](../../docs/runbooks/) — دليل العمليات (الاستجابة للحوادث، وضع عدم الاتصال، تعافي P2P، فشل الدفع).

---

## المساهمة

1. اتبع [الالتزامات التقليدية](https://www.conventionalcommits.org/) لجميع رسائل الالتزام.
2. قم بتشغيل `pnpm prepare` لتثبيت خطافات Lefthook قبل الالتزام.
3. تأكد من أن `lint ← typecheck ← test` جميعها تمر قبل فتح طلب سحب.
4. احتفظ بالتغييرات محدودة النطاق؛ CI مصفوف حسب المسار ويعمل فقط على سلاسل الأدوات ذات الصلة.

---

## الترخيص

مرخّص بموجب [ترخيص Apache، الإصدار 2.0](../../LICENSE).

---

<p align="center">
  <sub>Astra-System · مبني للتجزئة المتينة وبدون اتصال.</sub>
</p>
