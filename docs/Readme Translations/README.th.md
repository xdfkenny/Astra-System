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
  <a href="./README.bn.md">বাংলা</a> ·
  <a href="./README.de.md">Deutsch</a> ·
  <a href="./README.ur.md">اردو</a> ·
  <a href="./README.tr.md">Türkçe</a> ·
  <a href="./README.zh-TW.md">繁體中文</a> ·
  <a href="./README.vi.md">Tiếng Việt</a> ·
  <a href="./README.th.md"><b>ไทย</b></a> ·
  <a href="./README.la.md">Latina</a> ·
  <a href="./README.tlh.md">tlhIngan Hol</a>
  </sub>
</p>

[![CI](https://img.shields.io/badge/CI-passing-green.svg)](https://github.com/xdfkenny/Astra-System/actions)
[![License](https://img.shields.io/badge/license-Apache%202.0-blue.svg)](../../LICENSE)
[![Go](https://img.shields.io/badge/Go-1.25-00ADD8.svg)](https://go.dev/dl/)
[![Rust](https://img.shields.io/badge/Rust-1.82-dea584.svg)](https://www.rust-lang.org/tools/install)
[![TypeScript](https://img.shields.io/badge/TypeScript-5.7-3178C6.svg)](https://www.typescriptlang.org/download)

> แพลตฟอร์มชำระเงินด้วยตนเองอัตโนมัติระดับการผลิตที่ออกแบบมาเพื่อสภาพแวดล้อมการค้าปลีก 24/7 โดยเน้นระบบออฟไลน์เป็นหลัก

**Astra-System** เป็นโมโนเรโพแบบหลายภาษาที่ขับเคลื่อนตู้ชำระเงินด้วยตนเองทั้งแบบไม่มีและมีพนักงานดูแล มอบการดำเนินงานร้านค้าแบบไม่มีเวลาหยุดทำงานด้วย **ความทนทานออฟไลน์ 48 ชั่วโมง** โมเดลความปลอดภัยแบบ zero-trust และชั้นการซิงค์แบบ peer-to-peer mesh ที่ทำให้ทุกตู้ในร้านค้ามีความสอดคล้องกัน — แม้เมื่อระบบคลาวด์ไม่สามารถเข้าถึงได้

---

## สารบัญ

- [ภาพรวม](#ภาพรวม)
- [คุณสมบัติหลัก](#คุณสมบัติหลัก)
- [สถาปัตยกรรม](#สถาปัตยกรรม)
- [เทคโนโลยีที่ใช้](#เทคโนโลยีที่ใช้)
- [โครงสร้างคลังเก็บ](#โครงสร้างคลังเก็บ)
- [เริ่มต้นใช้งาน](#เริ่มต้นใช้งาน)
- [ขั้นตอนการพัฒนา](#ขั้นตอนการพัฒนา)
- [สร้างและรัน](#สร้างและรัน)
- [การทดสอบ](#การทดสอบ)
- [เอกสารประกอบ](#เอกสารประกอบ)
- [การมีส่วนร่วม](#การมีส่วนร่วม)
- [สัญญาอนุญาต](#สัญญาอนุญาต)

---

## ภาพรวม

Astra-System ทำให้ผู้ค้าปลีกสามารถdeploy ตู้ชำระเงินด้วยตนเองที่สามารถ **ทำงานได้อย่างอิสระนานถึง 48 ชั่วโมง** โดยไม่ต้องเชื่อมต่ออินเทอร์เน็ต ความทนทานถูกแบ่งออกเป็นสามชั้น:

1. **ชั้นข้อมูลท้องถิ่น** — ร้านค้า SQLite (SQLCipher) ที่เข้ารหัสบนทุกตู้ ซึ่งมีแคตตาล็อกเมนูเต็มรูปแบบ คลังสินค้า รายการธุรกรรมที่รอดำเนินการ และโทเค็นการชำระเงินแบบออฟไลน์
2. **Peer-to-peer mesh** — ตู้ค้นพบกันและกันผ่านเครือข่ายท้องถิ่น (mDNS + libp2p/QUIC) และจำลองสถานะโดยใช้ CRDTs โดยเลือกผู้นำ Raft เมื่อมีสามตู้ขึ้นไป
3. **การเสื่อมสภาพอย่างราบรื่น** — การชำระเงิน การคลังสินค้า และการรับออเดอร์ยังคงดำเนินต่อไปในระดับท้องถิ่น และกระทบยอดกับคลาวด์เมื่อการเชื่อมต่อกลับมา

ชั้นคลาวด์ (Go microservices, PostgreSQL 16, Redis 7, NATS JetStream) ให้บริการร้านค้าแบบ event-sourced ที่เป็นผู้มีอำนาจ การชำระเงิน และการจัดฝูงยาน

### เป้าหมายการออกแบบ

| เป้าหมาย           | เป้าหมาย                                                               |
| ------------------- | ---------------------------------------------------------------------- |
| ความทนทานออฟไลน์    | การทำงานอิสระ 48 ชั่วโมงโดยไม่มีการเชื่อมต่อคลาวด์                     |
| ความหน่วง           | < 200 ms โหลดเมนู, < 500 ms ซิงค์คลังสินค้า P2P, < 3 s failover ผู้นำ |
| ความพร้อมใช้งาน     | 99.99% uptime (ชั้นคลาวด์); 100% uptime ในโหมดเฉพาะท้องถิ่น            |
| ความปลอดภัย        | Zero-trust, mTLS ทุกที่ เส้นทางการชำระเงิน PCI-DSS ที่สอดคล้อง        |
| ขนาด               | 1–10,000 ตู้ต่อผู้เช่า; การ deploy คลาวด์แบบหลายภูมิภาค                |

---

## คุณสมบัติหลัก

- **เครื่องยนต์ออฟไลน์เป็นหลัก** — การผสาน CRDT แบบ deterministic (PN-Counter, LWW-Register, OR-Set) พร้อม Hybrid Logical Clocks สำหรับลำดับเชิงเหตุผลระหว่างตู้
- **P2P mesh และ Raft consensus** — การขนส่ง libp2p QUIC, การเข้ารหัส Noise protocol และ failover ผู้นำต่ำกว่า 3 วินาที
- **Transactional outbox** — การเผยแพร่เหตุการณ์แบบ exactly-once จากบริการคลาวด์ผ่าน NATS JetStream
- **ความปลอดภัยแบบ zero-trust** — mTLS, การลงนาม HMAC ต่อตู้, ตัวตน SPIFFE และเส้นทางการชำระเงิน PCI-DSS ที่สอดคล้อง (ข้อมูลบัตรจะไม่ถูกเก็บในหน่วยความจำตู้)
- **Verifone FFI bridge** — Rust wrapper ที่ปลอดภัย (`astra-verifone-ffi`) บน vendor C SDK สำหรับการรวมเทอร์มินัลการชำระเงิน
- **UI ตู้แบบชีวภาพ** — React 19 micro-frontend ที่สร้างด้วย Module Federation, XState v5 workflow machine และ Zustand/TanStack Query สำหรับจัดการสถานะ
- **ระบบอัจฉริยะขั้นสูง** — Ghost Carts, การจดจำผักผลไม้ (ONNX), lane intelligence (TFLite), WebAuthn/passkeys และ differential-privacy analytics
- **CI พร้อมรับ chaos** — มีการฉีด network partition ระหว่างการทดสอบแบบ integration เพื่อตรวจสอบความทนทาน การบรรจบของ CRDT และคิวการชำระเงิน
- **UI ตู้หลายภาษา** — ลูกค้าเลือกภาษาที่ต้องการเมื่อเริ่มเซสชันจาก 17+ ภาษาที่รองรับ (English, Spanish, Chinese, French, Japanese, Korean, Hindi, Arabic, Portuguese, Russian, Bengali, German, Urdu, Turkish, Traditional Chinese, Vietnamese, Thai และอื่นๆ) ข้อความ UI ทั้งหมด ใบเสร็จ และเสียงแจ้งเตือนจะแสดงใน locale ที่เลือก

---

## สถาปัตยกรรม

Astra-System แบ่งออกเป็น **ชั้นคลาวด์** และ **Store Edge / Kiosk Cluster**

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

สำหรับ拓扑ที่สมบูรณ์ โมเดลความปลอดภัย กระแสการชำระเงิน การสังเกตการณ์ และรายละเอียดการกู้คืนจากภัยพิบัติ ดู [`ARCHITECTURE.md`](./ARCHITECTURE.md)

### รายการบริการ

| บริการ           | ภาษา      | ความรับผิดชอบ                                    |
| ----------------- | ---------- | ------------------------------------------------ |
| `api-gateway`     | Go         | การกำหนดเส้นทาง edge, authN/authZ, การจำกัดอัตรา |
| `order-svc`       | Go         | วงจรชีวิตออเดอร์ การเก็บข้อมูลตะกร้า การulfillment |
| `payment-svc`     | Go         | การจัดการชำระเงิน การชำระเงินโทเค็น              |
| `inventory-svc`   | Go         | ระดับสต็อก การ hold แบบอ่อน การซิงค์แคตตาล็อก     |
| `cart-svc`        | Go         การผสาน CRDT ของตะกร้า การแก้ไข ghost-cart     |
| `sync-svc`        | Go         | mesh gateway ฝั่งคลาวด์ และ batch ingestion       |
| `astra-syncd`     | Rust       | P2P daemon ตู้, CRDT sync, Verifone FFI bridge    |
| `kiosk-shell`     | TypeScript | UI ลูกค้า React 19, การรวมอุปกรณ์ต่อพ่วง         |
| `update-server`   | Go         | การส่ง OTA manifest แบบลงนาม                      |

---

## เทคโนโลยีที่ใช้

- **Frontend** — TypeScript, React 19, Vite, Module Federation, XState v5, Zustand, TanStack Query, Tailwind CSS (v4 ใน apps, v3 ในระบบออกแบบ)
- **Backend** — Go (Fiber / gRPC), PostgreSQL 16, Redis 7, NATS JetStream
- **Edge** — Rust (`astra-syncd`, `astra-verifone-ffi`), SQLite (SQLCipher), libp2p
- **ML** — ONNX Runtime, TensorFlow Lite
- **Infra** — Kubernetes, Docker / Podman, Traefik, HashiCorp Vault, Nix flake
- **Observability** — Prometheus, Grafana, Loki, Jaeger, OpenTelemetry

---

## โครงสร้างคลังเก็บ

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

## เริ่มต้นใช้งาน

### ข้อกำหนดเบื้องต้น

- **Node.js 22** และ **pnpm 9+**
- **Go 1.25**
- **Rust 1.82** (พร้อม `protoc` สำหรับสร้าง sync daemon)
- **Docker** และ **Docker Compose**
- *(ไม่บังคับ)* **Nix** สำหรับ toolchain ที่สามารถทำซ้ำได้อย่างสมบูรณ์:

  ```bash
  nix develop
  ```

### เริ่มต้นอย่างรวดเร็ว

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

คัดลอก `.env.example` ไปเป็น `.env` และปรับค่าตามต้องการก่อนรันบริการ

---

### ตัวติดตั้ง

ไบนารีทดสอบที่สร้างไว้ล่วงหน้าสำหรับ macOS, Linux และ Windows มีอยู่ใน [หน้า Releases](https://github.com/xdfkenny/Astra-System/releases)

| แพลตฟอร์ม        | ไบนารี                          |
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

## ขั้นตอนการพัฒนา

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

รันแพ็กเกจเดียวผ่าน Turborepo filter:

```bash
pnpm turbo run dev --filter=@astra/kiosk
pnpm turbo run test --filter=@astra/kiosk
```

### บริการ Go

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

## สร้างและรัน

### การสร้าง Protobuf

```bash
cd proto
buf generate        # or: protoc as documented in proto/README.md
```

### สแต็กเต็มรูปแบบท้องถิ่น

```bash
docker compose up -d
pnpm dev            # kiosk-shell hot reload
```

สำหรับ manifest ระดับ production ให้ใช้ `docker-compose.prod.yml`

---

## การทดสอบ

| ชั้น        | เครื่องมือ                                          |
| ------------ | -------------------------------------------------- |
| Unit (TS)    | Vitest + happy-dom                                 |
| E2E (TS)     | Playwright กับ `kiosk-shell`                        |
| Unit (Go)    | `go test -race ./...`                              |
| Unit (Rust)  | `cargo test`, `cargo clippy`                       |
| Integration  | Docker Compose stack (PostgreSQL, Redis, NATS)     |
| Chaos        | Network-partition injection ระหว่าง integration  |

> การทดสอบแบบ integration และ chaos ต้องใช้ Docker ที่รัน container `postgres`, `redis` และ `nats`

---

## เอกสารประกอบ

เอกสารฉบับสมบูรณ์มีอยู่ใน [`docs/`](./docs/):

| ส่วน | เนื้อหา |
|---------|----------|
| **สถาปัตยกรรม** | [ภาพรวม](./docs/architecture/overview.md), [การออกแบบระบบ](./docs/architecture/system-design.md), [กลยุทธ์ออฟไลน์เป็นหลัก](./docs/architecture/offline-first.md), [โมเดลความปลอดภัย](./docs/architecture/security-model.md) |
| **Backend** | [Microservices](./docs/backend/microservices.md), [API Gateway](./docs/backend/api-gateway.md), [REST API](./docs/backend/rest-api.md), [gRPC API](./docs/backend/grpc-api.md), [Payment Orchestrator](./docs/backend/payment-orchestrator.md) |
| **Frontend** | [Micro-Frontends](./docs/frontend/micro-frontends.md), [Kiosk Apps](./docs/frontend/kiosk-apps.md), [State Management](./docs/frontend/state-management.md) |
| **ฐานข้อมูล** | [Schema](./docs/database/schema.md), [Migrations](./docs/database/migrations.md), [Entities](./docs/database/entities.md) |
| **โครงสร้างพื้นฐาน** | [Docker](./docs/infrastructure/docker.md), [Kubernetes](./docs/infrastructure/kubernetes.md), [Observability](./docs/infrastructure/monitoring.md), [CI/CD](./docs/infrastructure/ci-cd.md) |
| **เครือข่าย** | [P2P Mesh](./docs/networking/p2p-mesh.md), [Protocols](./docs/networking/protocols.md) |
| **ความปลอดภัย** | [ภาพรวม](./docs/security/overview.md), [การยืนยันตัวตน](./docs/security/authentication.md), [การเข้ารหัส](./docs/security/encryption.md) |

เอกสารอ้างอิงหลัก:
- [`ARCHITECTURE.md`](./ARCHITECTURE.md) — การออกแบบระบบ โมเดลความปลอดภัย กระแสการชำระเงิน การสังเกตการณ์ และ DR
- [`UX_UI_AUDIT_REPORT.md`](./astra-service/UX_UI_AUDIT_REPORT.md) — ข้อกำหนดการออกแบบ UI ตู้แบบ "Living Weave" ชีวภาพ
- [`docs/API-BACKEND-ASTRA.md`](./docs/API-BACKEND-ASTRA.md) — รายการ API endpoint ที่สมบูรณ์
- [`docs/Readme Translations/`](./docs/Readme Translations/) — การแปล README ที่ช่วยโดยชุมชนใน 17+ ภาษา
- [`docs/runbooks/`](./docs/runbooks/) — runbook การปฏิบัติงาน (การตอบสนองต่อเหตุการณ์ โหมดออฟไลน์ การกู้คืน P2E ความล้มเหลวในการชำระเงิน)

---

## การมีส่วนร่วม

1. ปฏิบัติตาม [Conventional Commits](https://www.conventionalcommits.org/) สำหรับข้อความ commit ทั้งหมด
2. รัน `pnpm prepare` เพื่อติดตั้ง Lefthook pre-commit hooks
3. ตรวจสอบให้แน่ใจว่า `lint → typecheck → test` ผ่านทั้งหมดก่อนเปิด pull request
4. ให้การเปลี่ยนแปลงอยู่ในขอบเขต path; CI จะกรองตาม path และรันเฉพาะเครื่องมือที่เกี่ยวข้องเท่านั้น

---

## สัญญาอนุญาต

ได้รับอนุญาตภายใต้ [Apache License, Version 2.0](../../LICENSE)

---

<p align="center">
  <sub>Astra-System · สร้างเพื่อการค้าปลีกที่ทนทานและออฟไลน์เป็นหลัก</sub>
</p>
