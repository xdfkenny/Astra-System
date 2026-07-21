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
  <a href="./README.ur.md">اردو</a> ·
  <a href="./README.tr.md">Türkçe</a> ·
  <a href="./README.zh-TW.md">繁體中文</a> ·
  <a href="./README.vi.md"><b>Tiếng Việt</b></a> ·
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

> Nền tảng thanh toán tự động cấp sản xuất, ưu tiên ngoại tuyến, được thiết kế cho môi trường bán lẻ hoạt động 24/7.

**Astra-System** là một monorepo đa ngôn ngữ hỗ trợ các ki-ốt tự phục vụ có giám sát và không giám sát. Hệ thống vận hành cửa hàng không gián đoạn với **48 giờ chống chịu ngoại tuyến**, mô hình bảo mật zero-trust và lớp đồng bộ mesh ngang hàng giữ cho mọi ki-ốt trong cửa hàng luôn nhất quán — ngay cả khi không thể kết nối đám mây.

---

## Mục Lục

- [Tổng Quan](#tổng-quan)
- [Tính Năng Chính](#tính-năng-chính)
- [Kiến Trúc](#kiến-trúc)
- [Công Nghệ](#công-nghệ)
- [Cấu Trúc Kho](#cấu-trúc-kho)
- [Bắt Đầu](#bắt-đầu)
- [Quy Trình Phát Triển](#quy-trình-phát-triển)
- [Xây Dựng & Chạy](#xây-dựng--chạy)
- [Kiểm Thử](#kiểm-thử)
- [Tài Liệu](#tài-liệu)
- [Đóng Góp](#đóng-góp)
- [Giấy Phép](#giấy-phép)

---

## Tổng Quan

Astra-System cho phép các nhà bán lẻ triển khai đội ngũ ki-ốt tự phục vụ hoạt động **tự động lên đến 48 giờ** mà không cần kết nối internet. Khả năng chống chịu được phân tầng qua ba lớp:

1. **Lớp dữ liệu cục bộ** — kho SQLite mã hóa (SQLCipher) trên mỗi ki-ốt chứa toàn bộ danh mục menu, kho hàng, giao dịch đang chờ và token thanh toán ngoại tuyến.
2. **Mesh ngang hàng** — các ki-ốt khám phá nhau qua mạng cục bộ (mDNS + libp2p/QUIC) và sao chép trạng thái bằng CRDTs, bầu Raft leader khi có ba ki-ốt trở lên.
3. **Suy giảm nhẹ nhàng** — thanh toán, kho hàng và bắt lệnh tiếp tục hoạt động cục bộ và đối chiếu với đám mây khi kết nối được khôi phục.

Lớp đám mây (microservice Go, PostgreSQL 16, Redis 7, NATS JetStream) cung cấp kho sự kiện có nguồn gốc, thanh toán bù trừ và quản lý đội ngũ.

### Mục Tiêu Thiết Kế

| Mục Tiêu           | Chỉ Tiêu                                                                 |
| ------------------ | ------------------------------------------------------------------------ |
| Chống chịu ngoại tuyến | 48 giờ hoạt động tự động không kết nối đám mây                          |
| Độ trễ             | < 200 ms tải menu, < 500 ms đồng bộ kho P2P, < 3 s chuyển đổi leader    |
| Khả dụng           | 99.99% thời gian hoạt động (lớp đám mây); 100% khi hoạt động cục bộ     |
| Bảo mật            | Zero trust, mTLS mọi nơi, đường dẫn thanh toán tuân thủ PCI-DSS          |
| Quy mô             | 1–10.000 ki-ốt mỗi khách hàng; triển khai đám mây đa vùng               |

---

## Tính Năng Chính

- **Động cơ ưu tiên ngoại tuyến** — kết hợp CRDT xác định (PN-Counter, LWW-Register, OR-Set) với Hybrid Logical Clocks để sắp xếp nhân quả giữa các ki-ốt.
- **Mesh P2P & đồng thuận Raft** — vận chuyển libp2p QUIC, mã hóa giao thức Noise và chuyển đổi leader dưới 3 giây.
- **Hộp thư giao dịch** — xuất sự kiện exactly-once từ dịch vụ đám mây qua NATS JetStream.
- **Bảo mật zero-trust** — mTLS, ký HMAC cho từng ki-ốt, danh tính SPIFFE và đường dẫn thanh toán tuân thủ PCI-DSS (dữ liệu thẻ không bao giờ tiếp xúc bộ nhớ ki-ốt).
- **Cầu nối FFI Verifone** — gói bọc Rust an toàn (`astra-verifone-ffi`) trên SDK C của nhà cung cấp để tích hợp thiết bị thanh toán.
- **Giao diện ki-ốt sinh học** — micro-frontend React 19 xây dựng bằng Module Federation, máy trạng thái XState v5 và quản lý trạng thái Zustand/TanStack Query.
- **Trí tuệ nâng cao** — Ghost Carts, nhận dạng sản phẩm (ONNX), trí tuệ làn (TFLite), WebAuthn/passkeys và phân tích bảo mật差 phân.
- **CI sẵn sàng hỗn loạn** — tiêm chia mạng trong quá trình kiểm thử tích hợp để xác minh khả năng chống chịu, hội tụ CRDT và hàng đợi thanh toán.
- **Giao diện ki-ốt đa ngôn ngữ** — khách hàng chọn ngôn ngữưa thích khi bắt đầu phiên từ 17+ ngôn ngữ được hỗ trợ (Tiếng Anh, Tây Ban Nha, Trung Quốc, Pháp, Nhật Bản, Hàn Quốc, Hindi, Ả Rập, Bồ Đào Nha, Nga, Bengal, Đức, Urdu, Thổ Nhĩ Kỳ, Trung Phồn Thể, Tiếng Việt, Thái Lan và nhiều hơn nữa). Tất cả văn bản giao diện, hóa đơn và hướng dẫn âm thanh hiển thị theo ngôn ngữ đã chọn.

---

## Kiến Trúc

Astra-System được chia thành **Lớp Đám Mây** và **Cạnh Cửa Hàng / Cụm Ki-ốt**.

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

Để xem chi tiết đầy đủ về topo, mô hình bảo mật, luồng thanh toán, khả năng quan sát và khôi phục thảm họa, xem [`ARCHITECTURE.md`](./ARCHITECTURE.md).

### Bảng Dịch Vụ

| Dịch Vụ           | Ngôn Ngôn  | Trách Nhiệm                                      |
| ----------------- | ---------- | ------------------------------------------------ |
| `api-gateway`     | Go         | Định tuyến cạnh, xác thực/phân quyền, giới hạn tốc độ |
| `order-svc`       | Go         | Vòng đời đơn hàng, lưu trữ giỏ, thực hiện        |
| `payment-svc`     | Go         | Điều phối thanh toán, bù trừ token               |
| `inventory-svc`   | Go         | Mức tồn kho, giữ mềm, đồng bộ danh mục          |
| `cart-svc`        | Go         | Kết hợp CRDT giỏ hàng, giải quyết ghost-cart     |
| `sync-svc`        | Go         | Cổng mesh phía đám mây và nhập hàng loạt         |
| `astra-syncd`     | Rust       | Daemon P2P ki-ốt, đồng bộ CRDT, cầu nối FFI Verifone |
| `kiosk-shell`     | TypeScript | Giao diện khách hàng React 19, tích hợp thiết bị ngoại vi |
| `update-server`   | Go         | Phát hành biểu mẫu OTA đã ký                     |

---

## Công Nghệ

- **Frontend** — TypeScript, React 19, Vite, Module Federation, XState v5, Zustand, TanStack Query, Tailwind CSS (v4 trong apps, v3 trong hệ thống thiết kế).
- **Backend** — Go (Fiber / gRPC), PostgreSQL 16, Redis 7, NATS JetStream.
- **Cạnh** — Rust (`astra-syncd`, `astra-verifone-ffi`), SQLite (SQLCipher), libp2p.
- **ML** — ONNX Runtime, TensorFlow Lite.
- **Hạ Tầng** — Kubernetes, Docker / Podman, Traefik, HashiCorp Vault, Nix flake.
- **Quan Sát** — Prometheus, Grafana, Loki, Jaeger, OpenTelemetry.

---

## Cấu Trúc Kho

```text
astra-service/          Mã dịch vụ và ứng dụng
  apps/                 Micro-frontend TypeScript (kiosk-shell, kiosk-menu, …)
  packages/             Thư viện chia sẻ và hệ thống thiết kế
  services/             Microservice Go
  sync-daemon/          astra-syncd (Rust) daemon P2P
  daemons/              Daemon sidecar (payment-sidecar)
  tools/                Công cụ vận hành (hỗn loạn, v.v.)
services/               Dịch vụ độc lập (update-server, …)
database/               Chuyển đổi schema
proto/                  Định nghĩa Protocol Buffer và mã được tạo
docs/                   Sổ tay vận hành
infra/                  Công cụ hạ tầng và trợ lý bí mật
.github/                Quy trình CI và tệp cộng đồng
flake.nix               Vỏ Nix phát triển có thể tái tạo
docker-compose*.yml     Biểu mẫu compose cục bộ và sản xuất
```

---

## Bắt Đầu

### Yêu Cầu

- **Node.js 22** và **pnpm 9+**
- **Go 1.25**
- **Rust 1.82** (với `protoc` để xây dựng daemon đồng bộ)
- **Docker** và **Docker Compose**
- *(Tùy chọn)* **Nix** để có chuỗi công cụ hoàn toàn có thể tái tạo:

  ```bash
  nix develop
  ```

### Bắt Đầu Nhanh

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

Sao chép `.env.example` thành `.env` và điều chỉnh giá trị khi cần trước khi chạy dịch vụ.

---

### Trình Cài Đặt

Các tệp nhị phân kiểm thử đã xây dựng sẵn cho macOS, Linux và Windows có trên [Trang Phát Hành](https://github.com/xdfkenny/Astra-System/releases).

| Nền Tảng        | Tệp Nhị Phân                   |
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

## Quy Trình Phát Triển

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

Chạy một gói duy nhất qua bộ lọc Turborepo:

```bash
pnpm turbo run dev --filter=@astra/kiosk
pnpm turbo run test --filter=@astra/kiosk
```

### Dịch vụ Go

```bash
cd astra-service/services
go test -race ./...
go vet ./...
```

### Daemon Rust

```bash
cd astra-service/sync-daemon
cargo test
cargo clippy -- -D warnings
cargo fmt --check
```

---

## Xây Dựng & Chạy

### Tạo Protocol Buffer

```bash
cd proto
buf generate        # or: protoc as documented in proto/README.md
```

### Đống Đầy Đủ Cục Bộ

```bash
docker compose up -d
pnpm dev            # kiosk-shell hot reload
```

Để xem biểu mẫu sản xuất, sử dụng `docker-compose.prod.yml`.

---

## Kiểm Thử

| Lớp         | Công Cụ                                            |
| ------------ | -------------------------------------------------- |
| Đơn vị (TS) | Vitest + happy-dom                                 |
| E2E (TS)     | Playwright trên `kiosk-shell`                      |
| Đơn vị (Go) | `go test -race ./...`                              |
| Đơn vị (Rust) | `cargo test`, `cargo clippy`                     |
| Tích hợp     | Đống Docker Compose (PostgreSQL, Redis, NATS)     |
| Hỗn loạn     | Tiêm chia mạng trong quá trình tích hợp           |

> Kiểm thử tích hợp và hỗn loạn yêu cầu Docker đang chạy với các container `postgres`, `redis` và `nats`.

---

## Tài Liệu

Tài liệu đầy đủ có trong [`docs/`](./docs/):

| Mục | Nội Dung |
|-----|----------|
| **Kiến Trúc** | [Tổng Quan](./docs/architecture/overview.md), [Thiết Kế Hệ Thống](./docs/architecture/system-design.md), [Chiến Lược Ưu Tiên Ngoại Tuyến](./docs/architecture/offline-first.md), [Mô Hình Bảo Mật](./docs/architecture/security-model.md) |
| **Backend** | [Microservice](./docs/backend/microservices.md), [API Gateway](./docs/backend/api-gateway.md), [REST API](./docs/backend/rest-api.md), [gRPC API](./docs/backend/grpc-api.md), [Điều Phối Thanh Toán](./docs/backend/payment-orchestrator.md) |
| **Frontend** | [Micro-Frontend](./docs/frontend/micro-frontends.md), [Ứng Dụng Ki-ốt](./docs/frontend/kiosk-apps.md), [Quản Lý Trạng Thái](./docs/frontend/state-management.md) |
| **Cơ Sở Dữ Liệu** | [Schema](./docs/database/schema.md), [Chuyển Đổi](./docs/database/migrations.md), [Thực Thể](./docs/database/entities.md) |
| **Hạ Tầng** | [Docker](./docs/infrastructure/docker.md), [Kubernetes](./docs/infrastructure/kubernetes.md), [Quan Sát](./docs/infrastructure/monitoring.md), [CI/CD](./docs/infrastructure/ci-cd.md) |
| **Mạng** | [Mesh P2P](./docs/networking/p2p-mesh.md), [Giao Thức](./docs/networking/protocols.md) |
| **Bảo Mật** | [Tổng Quan](./docs/security/overview.md), [Xác Thực](./docs/security/authentication.md), [Mã Hóa](./docs/security/encryption.md) |

Tài liệu tham khảo chính:
- [`ARCHITECTURE.md`](./ARCHITECTURE.md) — thiết kế hệ thống, mô hình bảo mật, luồng thanh toán, khả năng quan sát và khôi phục thảm họa.
- [`UX_UI_AUDIT_REPORT.md`](./astra-service/UX_UI_AUDIT_REPORT.md) — đặc tả thiết kế giao diện ki-ốt "Living Weave" sinh học.
- [`docs/API-BACKEND-ASTRA.md`](./docs/API-BACKEND-ASTRA.md) — danh sách đầy đủ các endpoint API.
- [`docs/Readme Translations/`](./docs/Readme Translations/) — bản dịch README đóng góp bởi cộng đồng với 17+ ngôn ngữ.
- [`docs/runbooks/`](./docs/runbooks/) — sổ tay vận hành (phản ứng sự cố, chế độ ngoại tuyến, khôi phục P2P, lỗi thanh toán).

---

## Đóng Góp

1. Tuân theo [Conventional Commits](https://www.conventionalcommits.org/) cho tất cả thông báo commit.
2. Chạy `pnpm prepare` để cài đặt hook trước commit Lefthook.
3. Đảm bảo `lint → typecheck → test` đều thông qua trước khi mở pull request.
4. Giữ thay đổi theo phạm vi đường dẫn; CI lọc theo đường dẫn và chỉ chạy chuỗi công cụ liên quan.

---

## Giấy Phép

Được cấp phép theo [Giấy Phép Apache, Phiên Bản 2.0](../../LICENSE).

---

<p align="center">
  <sub>Astra-System · Được xây dựng cho bán lẻ kiên cường, ưu tiên ngoại tuyến.</sub>
</p>
