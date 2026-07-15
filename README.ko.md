# Astra-System

<p align="center">
  <img src="https://raw.githubusercontent.com/cat-milk/Anime-Girls-Holding-Programming-Books/master/Typescript/Beako_Reading_The_TypeScript_Programming_Language.png" width="420" alt="Anime girl reading the TypeScript programming language book" />
</p>

[![CI](https://img.shields.io/badge/CI-pass-green.svg)](https://github.com/anomalyco/astra-system/actions)
[![License](https://img.shields.io/badge/license-proprietary-blue.svg)](#라이선스)
[![Go](https://img.shields.io/badge/Go-1.25-00ADD8.svg)](https://go.dev)
[![Rust](https://img.shields.io/badge/Rust-1.82-dea584.svg)](https://www.rust-lang.org)
[![TypeScript](https://img.shields.io/badge/TypeScript-5.x-3178C6.svg)](https://www.typescriptlang.org)

> 24/7 리테일 환경을 위해 설계된 프로덕션급 오프라인 우선 자동 셀프 계산대 플랫폼.

**Astra-System**은 무인 및 유인 셀프 계산대 키오스크를 지원하는 다국어 모노레포입니다. **48시간 오프라인 복원력**, 제로 트러스트 보안 모델, 그리고 P2P 메시 동기화 계층을 통해 클라우드에 접근할 수 없을 때도 매장 내 모든 키오스크를 일관되게 유지합니다.

---

## 목차

- [개요](#개요)
- [주요 기능](#주요-기능)
- [아키텍처](#아키텍처)
- [기술 스택](#기술-스택)
- [저장소 구조](#저장소-구조)
- [시작하기](#시작하기)
- [개발 워크플로](#개발-워크플로)
- [빌드 및 실행](#빌드-및-실행)
- [테스트](#테스트)
- [문서](#문서)
- [기여하기](#기여하기)
- [라이선스](#라이선스)

---

## 개요

Astra-System을 통해 소매업체는 인터넷 연결 없이도 **최대 48시간 동안 자율적으로 운영되는** 셀프 계산대 키오스크 fleet을 배포할 수 있습니다. 복원력은 세 계층으로 구성됩니다.

1. **로컬 데이터 계층** — 각 키오스크의 암호화된 SQLite(SQLCipher) 저장소에 전체 메뉴 카탈로그, 재고, 대기 중인 거래, 오프라인 결제 토큰을 보관합니다.
2. **P2P 메시 계층** — 키오스크가 로컬 네트워크(mDNS + libp2p/QUIC)에서 서로를 발견하고 CRDT로 상태를 복제하며, 3대 이상일 때 Raft 리더를 선출합니다.
3. **우아한 성능 저하** — 결제, 재고, 주문 수집이 로컬에서 계속 진행되며 연결이 복구되면 클라우드와 조정됩니다.

클라우드 계층(Go 마이크로서비스, PostgreSQL 16, Redis 7, NATS JetStream)은 이벤트 소싱을 진실의 원천으로 하는 저장소, 정산, fleet 관리를 제공합니다.

### 설계 목표

| 목표             | 목표치                                                              |
| ---------------- | ------------------------------------------------------------------- |
| 오프라인 복원력 | 클라우드 연결 없이 48시간 자율 운영                                 |
| 지연 시간        | 메뉴 로드 < 200ms, P2P 재고 동기화 < 500ms, 리더 장애 조치 < 3초  |
| 가용성          | 클라우드 99.99% 가동률; 로컬 전용 모드 100% 가동률                 |
| 보안            | 제로 트러스트, 전 구간 mTLS, PCI-DSS 준수 결제 경로                |
| 규모            | 테넌트당 1–10,000대 키오스크; 다중 리전 클라우드 배포              |

---

## 주요 기능

- **오프라인 우선 엔진** — 결정적 CRDT 병합(PN-Counter, LWW-Register, OR-Set)과 하이브리드 논리 클럭(HLC)으로 키오스크 간 인과 순서를 보장합니다.
- **P2P 메시 및 Raft 합의** — libp2p QUIC 전송, Noise 프로토콜 암호화, 3초 미만 리더 장애 조치.
- **트랜잭션 아웃박스** — NATS JetStream을 통한 클라우드 서비스 이벤트 정확히 한 번(exactly-once) 발행.
- **제로 트러스트 보안** — mTLS, 키오스크별 HMAC 서명, SPIFFE identity, PCI-DSS 준수 결제 경로(카드 데이터는 키오스크 메모리에 절대 남지 않음).
- **Verifone FFI 브리지** — 벤더 C SDK 위에 안전한 Rust 래퍼(`astra-verifone-ffi`)로 결제 단말기를 통합합니다.
- **생체 친화적 키오스크 UI** — Module Federation 기반 React 19 마이크로 프론트엔드, XState v5 워크플로 머신, Zustand/TanStack Query 상태 관리.
- **고급 지능** — Ghost Cart, 상품 인식(ONNX), 차선 지능(TFLite), WebAuthn/Passkey, 차등 프라이버시 분석.
- **카오스 대비 CI** — 통합 테스트 중 네트워크 분할을 주입하여 복원력, CRDT 수렴, 결제 큐잉을 검증합니다.

---

## 아키텍처

Astra-System은 **클라우드 계층**과 **매장 엣지 / 키오스크 클러스터**로 나뉩니다.

```text
┌─────────────────────────────────────────────────────────────────┐
│                         클라우드 계층                            │
│  API Gateway · Order Svc · Payment Svc · Inventory Svc ·       │
│  Cart Svc · Sync Svc · PostgreSQL 16 · Redis 7 · NATS JetStream │
└──────────────────────────────────┬──────────────────────────────┘
                                   │ TLS 1.3
┌──────────────────────────────────┴──────────────────────────────┐
│              매장 엣지 / 키오스크 클러스터                       │
│  키오스크 1 ─┐  키오스크 2 ─┐  키오스크 N ─┐                    │
│  React 19    │  React 19    │  React 19    │  (로컬 QUIC 메시) │
│  Rust P2P    │  Rust P2P    │  Rust P2P    │                    │
│  SQLite      │  SQLite      │  SQLite      │                    │
│  Verifone · 프린터 · 스캐너 · NFC/저울                          │
└─────────────────────────────────────────────────────────────────┘
```

전체 토폴로지, 보안 모델, 결제 흐름, 관측성, 재해 복구 세부 사항은 [`ARCHITECTURE.md`](./ARCHITECTURE.md)를 참조하세요.

### 서비스 목록

| 서비스            | 언어       | 책임                                               |
| ----------------- | ---------- | -------------------------------------------------- |
| `api-gateway`     | Go         | 엣지 라우팅, authN/authZ, 레이트 리미팅            |
| `order-svc`       | Go         | 주문 라이프사이클, 장바구니 영속화, 이행           |
| `payment-svc`     | Go         | 결제 오케스트레이션, 토큰 정산                     |
| `inventory-svc`   | Go         | 재고 수준, 소프트 홀드, 카탈로그 동기화            |
| `cart-svc`        | Go         | 장바구니 CRDT 병합, Ghost Cart 해결                |
| `sync-svc`        | Go         | 클라우드 측 메시 게이트웨이 및 배치 수집           |
| `astra-syncd`     | Rust       | 키오스크 P2P 데몬, CRDT 동기화, Verifone FFI 브리지|
| `kiosk-shell`     | TypeScript | React 19 고객 UI, 주변 기기 통합                   |
| `update-server`   | Go         | 서명된 OTA 매니페스트 전달                         |

---

## 기술 스택

- **프론트엔드** — TypeScript, React 19, Vite, Module Federation, XState v5, Zustand, TanStack Query, Tailwind CSS(앱은 v4, 디자인 시스템은 v3).
- **백엔드** — Go(Fiber / gRPC), PostgreSQL 16, Redis 7, NATS JetStream.
- **엣지** — Rust(`astra-syncd`, `astra-verifone-ffi`), SQLite(SQLCipher), libp2p.
- **ML** — ONNX Runtime, TensorFlow Lite.
- **인프라** — Kubernetes, Docker / Podman, Traefik, HashiCorp Vault, Nix flake.
- **관측성** — Prometheus, Grafana, Loki, Jaeger, OpenTelemetry.

---

## 저장소 구조

```text
astra-service/          서비스 및 애플리케이션 코드
  apps/                 TypeScript 마이크로 프론트엔드(kiosk-shell, kiosk-menu 등)
  packages/             공유 라이브러리 및 디자인 시스템
  services/             Go 마이크로서비스
  sync-daemon/          astra-syncd(Rust) P2P 데몬
  daemons/              사이드카 데몬(payment-sidecar)
  tools/                운영 도구(chaos 등)
database/               스키마 마이그레이션
proto/                  Protocol Buffer 정의 및 생성 코드
docs/                   운영 런북
infra/                  인프라 도구 및 시크릿 헬퍼
.github/                CI 워크플로 및 커뮤니티 파일
flake.nix               재현 가능한 Nix 개발 셸
docker-compose*.yml     로컬 및 프로덕션 compose 매니페스트
```

---

## 시작하기

### 사전 요구 사항

- **Node.js 22** 및 **pnpm 9+**
- **Go 1.25**
- **Rust 1.82**(sync 데몬 빌드를 위해 `protoc` 필요)
- **Docker** 및 **Docker Compose**
- *(선택)* 완전히 재현 가능한 툴체인을 위한 **Nix**:

  ```bash
  nix develop
  ```

### 빠른 시작

```bash
# 1. 프론트엔드 의존성 설치
pnpm install

# 2. 로컬 백엔드 스택 기동(PostgreSQL, Redis, NATS)
docker compose up -d

# 3. hot reload로 모든 TypeScript 앱 실행
pnpm dev

# 4. Rust sync 데몬 빌드
cd astra-service/sync-daemon && cargo build --release
```

서비스 실행 전에 `.env.example`을 `.env`로 복사하고 필요에 따라 값을 조정하세요.

---

## 개발 워크플로

```bash
# Lint, typecheck, test(순서가 중요)
pnpm lint
pnpm typecheck
pnpm test

# E2E 테스트(Playwright)
pnpm test:e2e

# 포맷
pnpm format && pnpm format:check

# 모든 패키지 빌드
pnpm build
```

Turborepo 필터로 단일 패키지 실행:

```bash
pnpm turbo run dev --filter=@astra/kiosk
pnpm turbo run test --filter=@astra/kiosk
```

### Go 서비스

```bash
cd astra-service/services
go test -race ./...
go vet ./...
```

### Rust 데몬

```bash
cd astra-service/sync-daemon
cargo test
cargo clippy -- -D warnings
cargo fmt --check
```

---

## 빌드 및 실행

### Protobuf 생성

```bash
cd proto
buf generate        # 또는 proto/README.md의 protoc 사용
```

### 로컬 전체 스택

```bash
docker compose up -d
pnpm dev            # kiosk-shell hot reload
```

프로덕션 매니페스트는 `docker-compose.prod.yml`을 사용하세요.

---

## 테스트

| 계층           | 도구                                                       |
| -------------- | ---------------------------------------------------------- |
| 단위(TS)       | Vitest + happy-dom                                         |
| E2E(TS)        | `kiosk-shell` 대상 Playwright                              |
| 단위(Go)       | `go test -race ./...`                                       |
| 단위(Rust)     | `cargo test`, `cargo clippy`                               |
| 통합           | Docker Compose 스택(PostgreSQL, Redis, NATS)               |
| 카오스         | 통합 테스트 중 네트워크 분할 주입                          |

> 통합 및 카오스 테스트에는 `postgres`, `redis`, `nats` 컨테이너가 실행 중인 Docker가 필요합니다.

---

## 문서

- [`ARCHITECTURE.md`](./ARCHITECTURE.md) — 시스템 설계, 보안 모델, 결제 흐름, 관측성, DR.
- [`AGENTS.md`](./AGENTS.md) — 본 저장소에서 작업하는 AI 코딩 에이전트를 위한 가이드.
- [`promt.md`](./promt.md) — "Living Weave" 생체 친화적 키오스크 UI 디자인 명세.
- `proto/README.md`, `astra-service/sync-daemon/README.md`, `docs/` — 하위 프로젝트 및 운영 런북.

---

## 기여하기

1. 모든 커밋 메시지는 [Conventional Commits](https://www.conventionalcommits.org/)를 따르세요.
2. Lefthook pre-commit 훅을 설치하려면 `pnpm prepare`를 실행하세요.
3. Pull Request를 열기 전에 `lint → typecheck → test`가 모두 통과하는지 확인하세요.
4. 변경 사항은 경로별로 범위를 한정하세요. CI는 경로 필터 기반이며 관련 툴체인만 실행합니다.

---

## 라이선스

독점 소프트웨어 — 자세한 내용은 `LICENSE` 파일을 참조하세요. 모든 권리 보유.

---

<p align="center">
  <sub>Astra-System · 복원력 있는 오프라인 우선 리테일을 위해 구축.</sub>
</p>
