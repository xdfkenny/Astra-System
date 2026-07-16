# Astra-System

<p align="center">
  <img src="https://raw.githubusercontent.com/cat-milk/Anime-Girls-Holding-Programming-Books/master/Typescript/Beako_Reading_The_TypeScript_Programming_Language.png" width="420" alt="Anime girl reading the TypeScript programming language book" />
</p>

<p align="center">
  <a href="../README.md">English</a> ·
  <a href="./README.es.md">Español</a> ·
  <a href="./README.zh.md"><b>中文</b></a> ·
  <a href="./README.ko.md">한국어</a> ·
  <a href="./README.ja.md">日本語</a>
</p>

[![CI](https://img.shields.io/badge/CI-pass-green.svg)](https://github.com/anomalyco/astra-system/actions)
[![License](https://img.shields.io/badge/license-Apache%202.0-blue.svg)](../LICENSE)
[![Go](https://img.shields.io/badge/Go-1.25-00ADD8.svg)](https://go.dev)
[![Rust](https://img.shields.io/badge/Rust-1.82-dea584.svg)](https://www.rust-lang.org)
[![TypeScript](https://img.shields.io/badge/TypeScript-5.x-3178C6.svg)](https://www.typescriptlang.org)

> 面向 24/7 零售环境打造、生产级、离线优先的自动化自助结账平台。

**Astra-System** 是一个多语言 Monorepo，为无人值守与有人值守的自助结账终端提供支撑。它以**48 小时离线韧性**、零信任安全模型，以及点对点网状同步层，确保门店内的每一台终端在云端不可达时依然保持一致。

---

## 目录

- [概述](#概述)
- [核心特性](#核心特性)
- [架构](#架构)
- [技术栈](#技术栈)
- [仓库结构](#仓库结构)
- [快速开始](#快速开始)
- [开发工作流](#开发工作流)
- [构建与运行](#构建与运行)
- [测试](#测试)
- [文档](#文档)
- [贡献指南](#贡献指南)
- [许可证](#许可证)

---

## 概述

Astra-System 让零售商能够部署自助结账终端集群，在**无互联网连接的情况下自主运行长达 48 小时**。韧性由三层构成：

1. **本地数据层** — 每台终端内置加密 SQLite（SQLCipher）存储，包含完整菜单目录、库存、待处理交易与离线支付令牌。
2. **点对点网状层** — 终端通过本地网络（mDNS + libp2p/QUIC）互相发现，并使用 CRDT 复制状态；当存在 3 台及以上终端时选举 Raft 领导者。
3. **优雅降级** — 支付、库存与订单采集在本地持续进行，并在连接恢复后与云端对账。

云端层（Go 微服务、PostgreSQL 16、Redis 7、NATS JetStream）提供以事件溯源为唯一事实来源的存储、结算与集群管理。

### 设计目标

| 目标           | 指标                                                                |
| -------------- | ------------------------------------------------------------------- |
| 离线韧性       | 无云端连接下自主运行 48 小时                                        |
| 延迟           | 菜单加载 < 200 ms，P2P 库存同步 < 500 ms，领导者故障切换 < 3 秒     |
| 可用性         | 云端层 99.99% 可用性；纯本地模式下 100% 可用性                      |
| 安全性         | 零信任、处处 mTLS、符合 PCI-DSS 的支付链路                          |
| 规模           | 每租户 1–10,000 台终端；云端多区域部署                              |

---

## 核心特性

- **离线优先引擎** — 基于确定性 CRDT 合并（PN-Counter、LWW-Register、OR-Set），并使用混合逻辑时钟（HLC）实现跨终端因果排序。
- **P2P 网状与 Raft 共识** — libp2p QUIC 传输、Noise 协议加密、领导者故障切换低于 3 秒。
- **事务性 Outbox** — 借助 NATS JetStream 实现云端服务事件「精确一次」发布。
- **零信任安全** — mTLS、每台终端 HMAC 签名、SPIFFE 身份，以及符合 PCI-DSS 的支付链路（卡数据永不进入终端内存）。
- **Verifone FFI 桥接** — 基于 Rust 的安全封装（`astra-verifone-ffi`）对接厂商 C SDK 以集成支付终端。
- **仿生学终端 UI** — 基于 Module Federation 的 React 19 微前端，采用 XState v5 工作流状态机与 Zustand/TanStack Query 状态管理。
- **高级智能** — Ghost Cart、商品识别（ONNX）、通道智能（TFLite）、WebAuthn/Passkey，以及差分隐私分析。
- **混沌就绪 CI** — 在集成测试中注入网络分区，以验证韧性、CRDT 收敛与支付排队能力。

---

## 架构

Astra-System 分为**云端层**与**门店边缘 / 终端集群**。

```text
┌─────────────────────────────────────────────────────────────────┐
│                         云端层                                   │
│  API Gateway · Order Svc · Payment Svc · Inventory Svc ·       │
│  Cart Svc · Sync Svc · PostgreSQL 16 · Redis 7 · NATS JetStream │
└──────────────────────────────────┬──────────────────────────────┘
                                   │ TLS 1.3
┌──────────────────────────────────┴──────────────────────────────┐
│              门店边缘 / 终端集群                                  │
│  终端 1 ─┐   终端 2 ─┐   终端 N ─┐                               │
│  React 19│   React 19│   React 19│  （本地 QUIC 网状）          │
│  Rust P2P│   Rust P2P│   Rust P2P│                              │
│  SQLite  │   SQLite  │   SQLite  │                              │
│  Verifone · 打印机 · 扫描枪 · NFC/电子秤                         │
└─────────────────────────────────────────────────────────────────┘
```

完整拓扑、安全模型、支付流程、可观测性与灾难恢复详情，请参阅 [`ARCHITECTURE.md`](./ARCHITECTURE.md)。

### 服务清单

| 服务             | 语言       | 职责                                             |
| ---------------- | ---------- | ------------------------------------------------ |
| `api-gateway`    | Go         | 边缘路由、authN/authZ、限流                       |
| `order-svc`      | Go         | 订单生命周期、购物车持久化、履约                 |
| `payment-svc`    | Go         | 支付编排、令牌结算                               |
| `inventory-svc`  | Go         | 库存水平、软占锁、目录同步                       |
| `cart-svc`       | Go         | 购物车 CRDT 合并、Ghost Cart 解析                |
| `sync-svc`       | Go         | 云端网状网关与批量摄取                           |
| `astra-syncd`    | Rust       | 终端 P2P 守护进程、CRDT 同步、Verifone FFI 桥接  |
| `kiosk-shell`    | TypeScript | React 19 客户 UI、外设集成                       |
| `update-server`  | Go         | 签名 OTA 清单下发                                |

---

## 技术栈

- **前端** — TypeScript、React 19、Vite、Module Federation、XState v5、Zustand、TanStack Query、Tailwind CSS（应用内 v4，设计系统 v3）。
- **后端** — Go（Fiber / gRPC）、PostgreSQL 16、Redis 7、NATS JetStream。
- **边缘** — Rust（`astra-syncd`、`astra-verifone-ffi`）、SQLite（SQLCipher）、libp2p。
- **机器学习** — ONNX Runtime、TensorFlow Lite。
- **基础设施** — Kubernetes、Docker / Podman、Traefik、HashiCorp Vault、Nix flake。
- **可观测性** — Prometheus、Grafana、Loki、Jaeger、OpenTelemetry。

---

## 仓库结构

```text
astra-service/          服务与应用代码
  apps/                 TypeScript 微前端（kiosk-shell、kiosk-menu 等）
  packages/             共享库与设计系统
  services/             Go 微服务
  sync-daemon/          astra-syncd（Rust）P2P 守护进程
  daemons/              边车守护进程（payment-sidecar）
  tools/                运维工具（chaos 等）
database/               数据库模式迁移
proto/                  Protocol Buffer 定义与生成代码
docs/                   运维手册
infra/                  基础设施工具与密钥助手
.github/                CI 工作流与社区文件
flake.nix               可复现的 Nix 开发环境
docker-compose*.yml     本地与生产 Compose 清单
```

---

## 快速开始

### 环境要求

- **Node.js 22** 与 **pnpm 9+**
- **Go 1.25**
- **Rust 1.82**（编译同步守护进程需 `protoc`）
- **Docker** 与 **Docker Compose**
- *（可选）* **Nix** 以获得完全可复现的工具链：

  ```bash
  nix develop
  ```

### 快速上手

```bash
# 1. 安装前端依赖
pnpm install

# 2. 启动本地后端栈（PostgreSQL、Redis、NATS）
docker compose up -d

# 3. 以热重载方式运行所有 TypeScript 应用
pnpm dev

# 4. 构建 Rust 同步守护进程
cd astra-service/sync-daemon && cargo build --release
```

运行服务前，请将 `.env.example` 复制为 `.env` 并按需调整。

---

## 开发工作流

```bash
# Lint、类型检查与测试（顺序很重要）
pnpm lint
pnpm typecheck
pnpm test

# 端到端测试（Playwright）
pnpm test:e2e

# 格式化
pnpm format && pnpm format:check

# 构建所有包
pnpm build
```

通过 Turborepo 过滤器运行单个包：

```bash
pnpm turbo run dev --filter=@astra/kiosk
pnpm turbo run test --filter=@astra/kiosk
```

### Go 服务

```bash
cd astra-service/services
go test -race ./...
go vet ./...
```

### Rust 守护进程

```bash
cd astra-service/sync-daemon
cargo test
cargo clippy -- -D warnings
cargo fmt --check
```

---

## 构建与运行

### Protobuf 生成

```bash
cd proto
buf generate        # 或：按 proto/README.md 使用 protoc
```

### 本地完整栈

```bash
docker compose up -d
pnpm dev            # kiosk-shell 热重载
```

生产清单请使用 `docker-compose.prod.yml`。

---

## 测试

| 层级           | 工具                                                         |
| -------------- | ------------------------------------------------------------ |
| 单元（TS）     | Vitest + happy-dom                                           |
| 端到端（TS）   | 针对 `kiosk-shell` 的 Playwright                             |
| 单元（Go）     | `go test -race ./...`                                         |
| 单元（Rust）   | `cargo test`、`cargo clippy`                                 |
| 集成           | Docker Compose 栈（PostgreSQL、Redis、NATS）                 |
| 混沌           | 集成阶段注入网络分区                                         |

> 集成测试与混沌测试需要运行中的 Docker，并启动 `postgres`、`redis`、`nats` 容器。

---

## 文档

- [`ARCHITECTURE.md`](./ARCHITECTURE.md) — 系统设计、安全模型、支付流程、可观测性与灾难恢复。
- [`promt.md`](./promt.md) —「Living Weave」仿生学终端 UI 设计规范。
- `proto/README.md`、`astra-service/sync-daemon/README.md` 与 `docs/` — 子项目与运维手册。

---

## 贡献指南

1. 所有提交信息请遵循 [Conventional Commits](https://www.conventionalcommits.org/)。
2. 运行 `pnpm prepare` 安装 Lefthook 预提交钩子。
3. 提交 Pull Request 前，请确保 `lint → typecheck → test` 全部通过。
4. 保持变更按路径聚焦；CI 采用路径过滤，仅运行相关工具链。

---

## 许可证

依据 [Apache 许可证 2.0 版](LICENSE) 授权。

---

<p align="center">
  <sub>Astra-System · 为具备离线韧性的零售而生。</sub>
</p>
