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
  <a href="./README.zh-TW.md"><b>繁體中文</b></a> ·
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

> 生產級、離線優先的自動化自助結帳平台，專為 24/7 零售環境而設計。

**Astra-System** 是一個多語言 monorepo，為有人值守和無人值守的自助結帳資訊站提供支援。它提供零停機時間的商店營運，具備 **48 小時離線韌性**、零信任安全模型，以及點對點 mesh 同步層，確保商店中每個資訊站保持一致——即使雲端無法連線也是如此。

---

## 目錄

- [概述](#概述)
- [主要功能](#主要功能)
- [架構](#架構)
- [技術棧](#技術棧)
- [專案結構](#專案結構)
- [快速開始](#快速開始)
- [開發流程](#開發流程)
- [建構與執行](#建構與執行)
- [測試](#測試)
- [文件](#文件)
- [貢獻](#貢獻)
- [授權條款](#授權條款)

---

## 概述

Astra-System 讓零售商能夠部署自助結帳資訊站 fleet，**在沒有網路連線的情況下自動運作長達 48 小時**。韌性透過三個層級來實現：

1. **本地資料層** — 每個資訊站上的加密 SQLite (SQLCipher) 儲存，包含完整選單目錄、庫存、待處理交易和離線支付令牌。
2. **點對點 Mesh** — 資訊站透過區域網路 (mDNS + libp2p/QUIC) 互相發現，並使用 CRDT 同步狀態，當三個或更多資訊站存在時選舉 Raft leader。
3. **優雅降級** — 支付、庫存和訂單擷取在本地繼續運作，並在連線恢復時與雲端進行對帳。

雲端層 (Go 微服務、PostgreSQL 16、Redis 7、NATS JetStream) 提供具有權威性的事件溯源儲存、清算和 fleet 管理。

### 設計目標

| 目標               | 指標                                                                     |
| ------------------ | ------------------------------------------------------------------------ |
| 離線韌性           | 48 小時自動運作，無需雲端連線                                              |
| 延遲               | < 200 ms 選單載入，< 500 ms P2P 庫存同步，< 3 s leader 故障轉移            |
| 可用性             | 99.99% 正常運行時間（雲端層）；100% 正常運行時間（本地模式）                 |
| 安全性             | 零信任、全面 mTLS、符合 PCI-DSS 的支付路徑                                  |
| 規模               | 每個租戶 1–10,000 個資訊站；多區域雲端部署                                  |

---

## 主要功能

- **離線優先引擎** — 確定性 CRDT 合併 (PN-Counter、LWW-Register、OR-Set)，搭配 Hybrid Logical Clocks 實現跨資訊站的因果排序。
- **P2P Mesh 與 Raft 共識** — libp2p QUIC 傳輸、Noise 協定加密，以及低於 3 秒的 leader 故障轉移。
- **交易外箱** — 透過 NATS JetStream 實現雲端服務的 exactly-once 事件發布。
- **零信任安全** — mTLS、每個資訊站 HMAC 簽章、SPIFFE 身份，以及符合 PCI-DSS 的支付路徑（卡片資料永遠不會接觸資訊站記憶體）。
- **Verifone FFI 橋接** — 在供應商 C SDK 上的安全 Rust 封裝 (`astra-verifone-ffi`)，用於支付終端整合。
- **生物親和資訊站 UI** — 基於 Module Federation 建構的 React 19 微前端、XState v5 工作流程機器，以及 Zustand/TanStack Query 狀態管理。
- **進階智慧** — Ghost Carts、產品辨識 (ONNX)、車道智慧 (TFLite)、WebAuthn/passkeys，以及差分隱私分析。
- **混沌就緒 CI** — 在整合測試期間注入網路分割，以驗證韌性、CRDT 收斂和支付佇列。
- **多語言資訊站 UI** — 客戶在 Session 開始時從 17+ 種支援語言中選擇首選語言（英語、西班牙語、中文、法語、日語、韓語、印地語、阿拉伯語、葡萄牙語、俄語、孟加拉語、德語、烏爾都語、土耳其語、繁體中文、越南語、泰語等）。所有 UI 文字、收據和語音提示均以所選語言呈現。

---

## 架構

Astra-System 分為**雲端層**和**商店邊緣 / 資訊站叢集**。

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

如需完整的拓撲、安全模型、支付流程、可觀測性和災難恢復詳情，請參閱 [`ARCHITECTURE.md`](./ARCHITECTURE.md)。

### 服務清單

| 服務              | 語言       | 職責                                               |
| ----------------- | ---------- | -------------------------------------------------- |
| `api-gateway`     | Go         | 邊緣路由、驗證/授權、速率限制                         |
| `order-svc`       | Go         | 訂單生命週期、購物車持久化、履行                       |
| `payment-svc`     | Go         | 支付編排、令牌清算                                   |
| `inventory-svc`   | Go         | 庫存水準、軟性保留、目錄同步                          |
| `cart-svc`        | Go         | 購物車 CRDT 合併、ghost-cart 解析                    |
| `sync-svc`        | Go         | 雲端側 mesh 閘道和批量擷取                          |
| `astra-syncd`     | Rust       | 資訊站 P2P daemon、CRDT 同步、Verifone FFI 橋接      |
| `kiosk-shell`     | TypeScript | React 19 客戶 UI、周邊設備整合                       |
| `update-server`   | Go         | 已簽署 OTA manifest 遞送                            |

---

## 技術棧

- **前端** — TypeScript、React 19、Vite、Module Federation、XState v5、Zustand、TanStack Query、Tailwind CSS（apps 中使用 v4，設計系統中使用 v3）。
- **後端** — Go (Fiber / gRPC)、PostgreSQL 16、Redis 7、NATS JetStream。
- **邊緣** — Rust (`astra-syncd`、`astra-verifone-ffi`)、SQLite (SQLCipher)、libp2p。
- **ML** — ONNX Runtime、TensorFlow Lite。
- **基礎設施** — Kubernetes、Docker / Podman、Traefik、HashiCorp Vault、Nix flake。
- **可觀測性** — Prometheus、Grafana、Loki、Jaeger、OpenTelemetry。

---

## 專案結構

```text
astra-service/          服務和應用程式碼
  apps/                 TypeScript 微前端 (kiosk-shell, kiosk-menu, …)
  packages/             共用函式庫和設計系統
  services/             Go 微服務
  sync-daemon/          astra-syncd (Rust) P2P daemon
  daemons/              Sidecar daemon (payment-sidecar)
  tools/                運維工具（混沌測試等）
services/               獨立服務 (update-server, …)
database/               Schema 遷移
proto/                  Protocol Buffer 定義和產生的程式碼
docs/                   運維手冊
infra/                  基礎設施工具和密鑰輔助
.github/                CI 工作流程和社群檔案
flake.nix               可重現的 Nix 開發環境
docker-compose*.yml     本機和生產 compose 設定檔
```

---

## 快速開始

### 前置要求

- **Node.js 22** 和 **pnpm 9+**
- **Go 1.25**
- **Rust 1.82**（需安裝 `protoc` 以建構同步 daemon）
- **Docker** 和 **Docker Compose**
- *（選用）* **Nix** 以獲得完全可重現的工具鏈：

  ```bash
  nix develop
  ```

### 快速啟動

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

複製 `.env.example` 為 `.env` 並在運行服務前根據需要調整數值。

---

### 安裝程式

macOS、Linux 和 Windows 的預建測試二進位檔可在[發佈頁面](https://github.com/xdfkenny/Astra-System/releases)取得。

| 平台            | 二進位檔                         |
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

## 開發流程

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

透過 Turborepo 篩選器執行單一套件：

```bash
pnpm turbo run dev --filter=@astra/kiosk
pnpm turbo run test --filter=@astra/kiosk
```

### Go 服務

```bash
cd astra-service/services
go test -race ./...
go vet ./...
```

### Rust Daemon

```bash
cd astra-service/sync-daemon
cargo test
cargo clippy -- -D warnings
cargo fmt --check
```

---

## 建構與執行

### Protocol Buffer 產生

```bash
cd proto
buf generate        # or: protoc as documented in proto/README.md
```

### 本機完整堆疊

```bash
docker compose up -d
pnpm dev            # kiosk-shell hot reload
```

如需生產環境設定檔，請使用 `docker-compose.prod.yml`。

---

## 測試

| 層級         | 工具                                                 |
| ------------ | ---------------------------------------------------- |
| 單元 (TS)    | Vitest + happy-dom                                   |
| E2E (TS)     | Playwright 對 `kiosk-shell` 進行測試                  |
| 單元 (Go)    | `go test -race ./...`                                |
| 單元 (Rust)  | `cargo test`、`cargo clippy`                         |
| 整合         | Docker Compose 堆疊 (PostgreSQL、Redis、NATS)        |
| 混沌         | 整合測試期間注入網路分割                              |

> 整合和混沌測試需要 Docker 正在運行，並啟動 `postgres`、`redis` 和 `nats` 容器。

---

## 文件

完整文件位於 [`docs/`](./docs/)：

| 章節 | 內容 |
|------|------|
| **架構** | [概述](./docs/architecture/overview.md)、[系統設計](./docs/architecture/system-design.md)、[離線優先策略](./docs/architecture/offline-first.md)、[安全模型](./docs/architecture/security-model.md) |
| **後端** | [微服務](./docs/backend/microservices.md)、[API Gateway](./docs/backend/api-gateway.md)、[REST API](./docs/backend/rest-api.md)、[gRPC API](./docs/backend/grpc-api.md)、[支付編排器](./docs/backend/payment-orchestrator.md) |
| **前端** | [微前端](./docs/frontend/micro-frontends.md)、[資訊站應用](./docs/frontend/kiosk-apps.md)、[狀態管理](./docs/frontend/state-management.md) |
| **資料庫** | [Schema](./docs/database/schema.md)、[遷移](./docs/database/migrations.md)、[實體](./docs/database/entities.md) |
| **基礎設施** | [Docker](./docs/infrastructure/docker.md)、[Kubernetes](./docs/infrastructure/kubernetes.md)、[可觀測性](./docs/infrastructure/monitoring.md)、[CI/CD](./docs/infrastructure/ci-cd.md) |
| **網路** | [P2P Mesh](./docs/networking/p2p-mesh.md)、[協定](./docs/networking/protocols.md) |
| **安全** | [概述](./docs/security/overview.md)、[驗證](./docs/security/authentication.md)、[加密](./docs/security/encryption.md) |

重要參考文件：
- [`ARCHITECTURE.md`](./ARCHITECTURE.md) — 系統設計、安全模型、支付流程、可觀測性和災難恢復。
- [`UX_UI_AUDIT_REPORT.md`](./astra-service/UX_UI_AUDIT_REPORT.md) — 「Living Weave」生物親和資訊站 UI 設計規範。
- [`docs/API-BACKEND-ASTRA.md`](./docs/API-BACKEND-ASTRA.md) — 完整的 API endpoint 清單。
- [`docs/Readme Translations/`](./docs/Readme Translations/) — 社群貢獻的 17+ 種語言 README 翻譯。
- [`docs/runbooks/`](./docs/runbooks/) — 運維手冊（事件回應、離線模式、P2P 恢復、支付失敗）。

---

## 貢獻

1. 所有 commit 訊息請遵循 [Conventional Commits](https://www.conventionalcommits.org/)。
2. 執行 `pnpm prepare` 安裝 Lefthook pre-commit hooks。
3. 在開啟 pull request 前，確保 `lint → typecheck → test` 全部通過。
4. 保持變更具備路徑範圍；CI 進行路徑過濾，僅執行相關工具鏈。

---

## 授權條款

採用 [Apache 授權條款，版本 2.0](../../LICENSE) 授權。

---

<p align="center">
  <sub>Astra-System · 為韌性、離線優先的零售而生。</sub>
</p>
