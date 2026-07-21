# Astra-System

<p align="center">
  <img src="https://raw.githubusercontent.com/cat-milk/Anime-Girls-Holding-Programming-Books/master/Typescript/Beako_Reading_The_TypeScript_Programming_Language.png" width="420" alt="TypeScriptのプログラミング言語の本を読むアニメの女の子" />
</p>

<p align="center">
  <a href="../../README.md"><b>English</b></a> ·
  <a href="./README.es.md">Español</a> ·
  <a href="./README.zh.md">中文</a> ·
  <a href="./README.fr.md">Français</a>
  <br>
  <sub>
  <a href="./README.ja.md"><b>日本語</b></a> ·
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

[![CI](https://img.shields.io/badge/CI-passing-green.svg)](https://github.com/xdfkenny/Astra-System/actions)
[![License](https://img.shields.io/badge/license-Apache%202.0-blue.svg)](../../LICENSE)
[![Go](https://img.shields.io/badge/Go-1.25-00ADD8.svg)](https://go.dev/dl/)
[![Rust](https://img.shields.io/badge/Rust-1.82-dea584.svg)](https://www.rust-lang.org/tools/install)
[![TypeScript](https://img.shields.io/badge/TypeScript-5.7-3178C6.svg)](https://www.typescriptlang.org/download)

> 24時間365日の小売環境向けに設計された、本番品質のオフライン最優先セルフレジ・プラットフォーム。

**Astra-System** は、無人・有人のセルフレジキオスクを支える多言語モノレポです。**48時間のオフライン耐障害性**、ゼロトラスト・セキュリティ・モデル、および P2P メッシュ同期層により、クラウドに到達できない場合でも店舗内のすべてのキオスクを一貫性のある状態に保ちます。

---

## 目次

- [概要](#概要)
- [主な機能](#主な機能)
- [アーキテクチャ](#アーキテクチャ)
- [技術スタック](#技術スタック)
- [リポジトリ構成](#リポジトリ構成)
- [はじめに](#はじめに)
- [開発ワークフロー](#開発ワークフロー)
- [ビルドと実行](#ビルドと実行)
- [テスト](#テスト)
- [ドキュメント](#ドキュメント)
- [コントリビューション](#コントリビューション)
- [ライセンス](#ライセンス)

---

## 概要

Astra-System により、小売事業者はインターネット接続なしで**最大48時間自律稼働する**セルフレジキオスクのフリートを展開できます。耐障害性は3層で構成されます。

1. **ローカル・データ層** — 各キオスクの暗号化 SQLite（SQLCipher）ストアに、完全なメニューカタログ、在庫、保留中の取引、オフライン決済トークンを保持します。
2. **P2P メッシュ層** — キオスクはローカルネットワーク（mDNS + libp2p/QUIC）で互いを発見し、CRDT で状態を複製します。3台以上ある場合は Raft リーダーを選出します。
3. **グレースフル・デグラデーション** — 決済、在庫、注文の取り込みはローカルで継続され、接続復旧時にクラウドと照合されます。

クラウド層（Go マイクロサービス、PostgreSQL 16、Redis 7、NATS JetStream）は、イベントソーシングを真実のソースとするストア、決済、フリート管理を提供します。

### 設計目標

| 目標             | ターゲット                                                     |
| ---------------- | -------------------------------------------------------------- |
| オフライン耐障害性 | クラウド接続なしで48時間の自律運用                             |
| レイテンシ       | メニュー読み込み < 200ms、P2P 在庫同期 < 500ms、リーダー故障切替 < 3秒 |
| 可用性           | クラウド層 99.99% 稼働率；ローカル専用モード 100% 稼働率       |
| セキュリティ     | ゼロトラスト、全経路 mTLS、PCI-DSS 準拠の決済経路              |
| スケール         | テナントあたり 1–10,000 台のキオスク；クラウド多重リージョン展開 |

---

## 主な機能

- **オフライン最優先エンジン** — 決定的な CRDT マージ（PN-Counter、LWW-Register、OR-Set）とハイブリッド論理クロック（HLC）によるキオスク間の因果順序付け。
- **P2P メッシュと Raft 合意** — libp2p QUIC トランスポート、Noise プロトコル暗号化、3秒未満のリーダー故障切替。
- **トランザクショナル・アウトボックス** — NATS JetStream によるクラウドサービスの「正確に1回」のイベント発行。
- **ゼロトラスト・セキュリティ** — mTLS、キオスクごとの HMAC 署名、SPIFFE アイデンティティ、PCI-DSS 準拠の決済経路（カードデータはキオスクのメモリに決して残らない）。
- **Verifone FFI ブリッジ** — ベンダー製 C SDK 上の安全な Rust ラッパー（`astra-verifone-ffi`）で決済端末を統合。
- **バイオフィリックなキオスク UI** — Module Federation ベースの React 19 マイクロフロントエンド、XState v5 ワークフロー状態機械、Zustand/TanStack Query の状態管理。
- **高度なインテリジェンス** — Ghost Cart、商品認識（ONNX）、レーン・インテリジェンス（TFLite）、WebAuthn/Passkey、差分プライバシ分析。
- **カオス対応 CI** — 統合テスト中にネットワーク分割を注入し、耐障害性・CRDT 収束・決済キューイングを検証。

---

## アーキテクチャ

Astra-System は**クラウド層**と**店舗エッジ / キオスク・クラスタ**に分かれます。

```text
┌─────────────────────────────────────────────────────────────────┐
│                         クラウド層                               │
│  API Gateway · Order Svc · Payment Svc · Inventory Svc ·       │
│  Cart Svc · Sync Svc · PostgreSQL 16 · Redis 7 · NATS JetStream │
└──────────────────────────────────┬──────────────────────────────┘
                                   │ TLS 1.3
┌──────────────────────────────────┴──────────────────────────────┐
│              店舗エッジ / キオスク・クラスタ                      │
│  キオスク1 ─┐  キオスク2 ─┐  キオスクN ─┐                       │
│  React 19   │  React 19   │  React 19   │  （ローカル QUIC メッシュ）│
│  Rust P2P   │  Rust P2P   │  Rust P2P   │                       │
│  SQLite     │  SQLite     │  SQLite     │                       │
│  Verifone · プリンタ · スキャナ · NFC/はかり                     │
└─────────────────────────────────────────────────────────────────┘
```

完全なトポロジ、セキュリティモデル、決済フロー、観測性、災害復旧の詳細は [`ARCHITECTURE.md`](../../ARCHITECTURE.md) を参照してください。

### サービス一覧

| サービス          | 言語       | 責務                                               |
| ----------------- | ---------- | -------------------------------------------------- |
| `api-gateway`     | Go         | エッジルーティング、authN/authZ、レート制限        |
| `order-svc`       | Go         | 注文ライフサイクル、カート永続化、フルフィルメント |
| `payment-svc`     | Go         | 決済オーケストレーション、トークン決済             |
| `inventory-svc`   | Go         | 在庫水準、ソフトホールド、カタログ同期             |
| `cart-svc`        | Go         | カート CRDT マージ、Ghost Cart 解決                |
| `sync-svc`        | Go         | クラウド側メッシュゲートウェイとバッチ取り込み     |
| `astra-syncd`     | Rust       | キオスク P2P デーモン、CRDT 同期、Verifone FFI ブリッジ |
| `kiosk-shell`     | TypeScript | React 19 顧客 UI、周辺機器統合                     |
| `update-server`   | Go         | 署名付き OTA マニフェスト配信                      |

---

## 技術スタック

- **フロントエンド** — TypeScript、React 19、Vite、Module Federation、XState v5、Zustand、TanStack Query、Tailwind CSS（アプリは v4、デザインシステムは v3）。
- **バックエンド** — Go（Fiber / gRPC）、PostgreSQL 16、Redis 7、NATS JetStream。
- **エッジ** — Rust（`astra-syncd`、`astra-verifone-ffi`）、SQLite（SQLCipher）、libp2p。
- **ML** — ONNX Runtime、TensorFlow Lite。
- **インフラ** — Kubernetes、Docker / Podman、Traefik、HashiCorp Vault、Nix flake。
- **観測性** — Prometheus、Grafana、Loki、Jaeger、OpenTelemetry。

---

## リポジトリ構成

```text
astra-service/          サービスおよびアプリケーション・コード
  apps/                 TypeScript マイクロフロントエンド（kiosk-shell、kiosk-menu 等）
  packages/             共有ライブラリとデザインシステム
  services/             Go マイクロサービス
  sync-daemon/          astra-syncd（Rust）P2P デーモン
  daemons/              サイドカー・デーモン（payment-sidecar）
  tools/                運用ツール（chaos 等）
database/               スキーマ移行
proto/                  Protocol Buffer 定義と生成コード
docs/                   運用ランブック
infra/                  インフラツールとシークレット・ヘルパー
.github/                CI ワークフローとコミュニティ・ファイル
flake.nix               再現可能な Nix 開発シェル
docker-compose*.yml     ローカルおよび本番 Compose マニフェスト
```

---

## はじめに

### 前提条件

- **Node.js 22** および **pnpm 9+**
- **Go 1.25**
- **Rust 1.82**（sync デーモンのビルドには `protoc` が必要）
- **Docker** および **Docker Compose**
- *（任意）* 完全に再現可能なツールチェーン向けの **Nix**：

  ```bash
  nix develop
  ```

### クイックスタート

```bash
# 1. フロントエンドの依存関係をインストール
pnpm install

# 2. ローカル・バックエンド・スタックを起動（PostgreSQL、Redis、NATS）
docker compose up -d

# 3. ホットリロードで全 TypeScript アプリを実行
pnpm dev

# 4. Rust sync デーモンをビルド
cd astra-service/sync-daemon && cargo build --release
```

サービス実行前に `.env.example` を `.env` にコピーし、必要に応じて値を調整してください。

---

### インストーラー

macOS、Linux、Windows 用のプレビルドテストバイナリは [Releases ページ](https://github.com/xdfkenny/Astra-System/releases) から入手できます。

| プラットフォーム | バイナリ                         |
| ---------------- | -------------------------------- |
| macOS (Intel)    | `astra-installer-darwin-amd64`   |
| macOS (Apple Silicon) | `astra-installer-darwin-arm64` |
| Linux (x86_64)   | `astra-installer-linux-amd64`    |
| Linux (ARM64)    | `astra-installer-linux-arm64`    |
| Windows (x86_64) | `astra-installer-windows-amd64.exe` |

```bash
# macOS / Linux — ブートストラップスクリプトをダウンロードして実行
curl -sL https://raw.githubusercontent.com/xdfkenny/Astra-System/main/installer/scripts/install.sh | bash

# または Releases からバイナリを直接ダウンロードし、実行権限を付与して実行：
./astra-installer-<platform>
```

---

## 開発ワークフロー

```bash
# Lint、typecheck、test（順序が重要）
pnpm lint
pnpm typecheck
pnpm test

# E2E テスト（Playwright）
pnpm test:e2e

# フォーマット
pnpm format && pnpm format:check

# 全パッケージをビルド
pnpm build
```

Turborepo フィルタで単一パッケージを実行：

```bash
pnpm turbo run dev --filter=@astra/kiosk
pnpm turbo run test --filter=@astra/kiosk
```

### Go サービス

```bash
cd astra-service/services
go test -race ./...
go vet ./...
```

### Rust デーモン

```bash
cd astra-service/sync-daemon
cargo test
cargo clippy -- -D warnings
cargo fmt --check
```

---

## ビルドと実行

### Protobuf 生成

```bash
cd proto
buf generate        # または proto/README.md の protoc を使用
```

### ローカル完全スタック

```bash
docker compose up -d
pnpm dev            # kiosk-shell のホットリロード
```

本番マニフェストには `docker-compose.prod.yml` を使用してください。

---

## テスト

| 層             | ツール                                              |
| -------------- | --------------------------------------------------- |
| 単体（TS）     | Vitest + happy-dom                                  |
| E2E（TS）      | `kiosk-shell` 対象の Playwright                     |
| 単体（Go）     | `go test -race ./...`                                |
| 単体（Rust）   | `cargo test`、`cargo clippy`                        |
| 統合           | Docker Compose スタック（PostgreSQL、Redis、NATS）  |
| カオス         | 統合テスト中のネットワーク分割注入                  |

> 統合テストおよびカオス・テストには、`postgres`、`redis`、`nats` コンテナが起動している Docker が必要です。

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

## コントリビューション

1. すべてのコミットメッセージは [Conventional Commits](https://www.conventionalcommits.org/) に従ってください。
2. Lefthook の pre-commit フックをインストールするには `pnpm prepare` を実行してください。
3. Pull Request を作成する前に `lint → typecheck → test` がすべて通過することを確認してください。
4. 変更はパス単位で範囲を絞ってください。CI はパス・フィルタ方式で、関連するツールチェインのみを実行します。

---

## ライセンス

[Apache License 2.0](LICENSE) に基づいてライセンス供与されています。

---

<p align="center">
  <sub>Astra-System · 耐障害性のあるオフライン最優先リテールのために構築。</sub>
</p>
