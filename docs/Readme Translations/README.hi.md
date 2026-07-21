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
  <a href="./README.hi.md"><b>हिन्दी</b></a> ·
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

> उत्पादन-ग्रेड, ऑफ़लाइन-प्रथम स्वचालित सेल्फ-चेकआउट प्लेटफ़ॉर्म जो 24/7 खुदरा वातावरण के लिए इंजीनियर किया गया है।

**Astra-System** एक बहु-भाषा मोनोरेपो है जो बिना पर्यवेक्षित और पर्यवेक्षित सेल्फ-चेकआउट कियोस्क को संचालित करता है। यह **48 घंटे की ऑफ़लाइन लचीलेपन** के साथ ज़ीरो-डाउनटाइम स्टोर संचालन प्रदान करता है, एक ज़ीरो-ट्रस्ट सुरक्षा मॉडल, और एक पीयर-टू-पीयर मेश सिंक लेयर जो प्रत्येक कियोस्क को स्टोर में सुसंगत रखता है — भले ही क्लाउड अनुपलब्ध हो।

---

## विषय सूची

- [अवलोकन](#अवलोकन)
- [प्रमुख विशेषताएँ](#प्रमुख-विशेषताएँ)
- [वास्तुकला](#वास्तुकला)
- [प्रौद्योगिकी स्टैक](#प्रौद्योगिकी-स्टैक)
- [रिपॉज़िटरी लेआउट](#रिपॉज़िटरी-लेआउट)
- [शुरुआत कैसे करें](#शुरुआत-कैसे-करें)
- [विकास कार्यप्रवाह](#विकास-कार्यप्रवाह)
- [बिल्ड और रन](#बिल्ड-और-रन)
- [परीक्षण](#परीक्षण)
- [दस्तावेज़ीकरण](#दस्तावेज़ीकरण)
- [योगदान](#योगदान)
- [लाइसेंस](#लाइसेंस)

---

## अवलोकन

Astra-System खुदरा विक्रेताओं को ऐसे सेल्फ-चेकआउट कियोस्क के बेड़े तैनात करने में सक्षम बनाता है जो इंटरनेट कनेक्टिविटी के बिना **48 घंटे तक स्वायत्त रूप से संचालित** हो सकते हैं। लचीलापन तीन स्तरों में विभाजित है:

1. **स्थानीय डेटा लेयर** — प्रत्येक कियोस्क पर एक एन्क्रिप्टेड SQLite (SQLCipher) स्टोर जिसमें पूरा मेनू कैटलॉग, इन्वेंट्री, लंबित लेनदेन, और ऑफ़लाइन भुगतान टोकन हैं।
2. **पीयर-टू-पीयर मेश** — कियोस्क स्थानीय नेटवर्क पर एक-दूसरे की खोज करते हैं (mDNS + libp2p/QUIC) और CRDTs का उपयोग करके अवस्था को दोहराते हैं, जब तीन या अधिक उपस्थित हों तो Raft लीडर चुनते हैं।
3. **सुचारु गिरावट** — भुगतान, इन्वेंट्री, और ऑर्डर कैप्चर स्थानीय रूप से जारी रहते हैं और कनेक्टिविटी लौटने पर क्लाउड के साथ समन्वित होते हैं।

क्लाउड टियर (Go माइक्रोसर्विस, PostgreSQL 16, Redis 7, NATS JetStream) अधिकृत इवेंट-सोर्स्ड स्टोर, सेटलमेंट, और फ्लीट प्रबंधन प्रदान करता है।

### डिज़ाइन लक्ष्य

| लक्ष्य               | लक्ष्य                                                                 |
| ------------------- | ---------------------------------------------------------------------- |
| ऑफ़लाइन लचीलापन     | क्लाउड कनेक्टिविटी के बिना 48 घंटे का स्वायत्त संचालन                  |
| विलंबता              | < 200 ms मेनू लोड, < 500 ms P2P इन्वेंट्री सिंक, < 3 s लीडर फेलओवर   |
| उपलब्धता             | 99.99% अपटाइम (क्लाउड टियर); स्थानीय-केवल मोड के दौरान 100% अपटाइम     |
| सुरक्षा              | ज़ीरो ट्रस्ट, हर जगह mTLS, PCI-DSS अनुपालित भुगतान पथ                  |
| स्केल               | 1–10,000 कियोस्क प्रति टेनेंट; मल्टी-रीजन क्लाउड तैनाती               |

---

## प्रमुख विशेषताएँ

- **ऑफ़लाइन-प्रथम इंजन** — निर्धारित CRDT विलय (PN-Counter, LWW-Register, OR-Set) कियोस्क में कार्यकारी क्रम के लिए Hybrid Logical Clocks के साथ।
- **P2P मेश और Raft सहमति** — libp2p QUIC ट्रांसपोर्ट, Noise प्रोटोकॉल एन्क्रिप्शन, और 3 सेकंड से कम लीडर फेलओवर।
- **ट्रांज़ैक्शनल आउटबॉक्स** — NATS JetStream के माध्यम से क्लाउड सर्विस से एक्ज़ैक्ट-वन्स इवेंट प्रकाशन।
- **ज़ीरो-ट्रस्ट सुरक्षा** — mTLS, प्रति-कियोस्क HMAC साइनिंग, SPIFFE पहचान, और PCI-DSS अनुपालित भुगतान पथ (कार्ड डेटा कभी कियोस्क मेमोरी को नहीं छूता)।
- **Verifone FFI ब्रिज** — वेंडर C SDK पर एक सुरक्षित Rust रैपर (`astra-verifone-ffi`) भुगतान टर्मिनल एकीकरण के लिए।
- **बायोफिलिक कियोस्क UI** — React 19 माइक्रो-फ्रंटएंड जो Module Federation, XState v5 वर्कफ़्लो मशीन, और Zustand/TanStack Query स्टेट प्रबंधन के साथ बनाया गया है।
- **उन्नत बुद्धिमत्ता** — Ghost Carts, उत्पाद पहचान (ONNX), लेन इंटेलिजेंस (TFLite), WebAuthn/passkeys, और डिफ़रेंशियल-प्राइवेसी एनालिटिक्स।
- **केओस-तैयार CI** — इंटीग्रेशन परीक्षणों के दौरान नेटवर्क पार्टीशन इंजेक्ट किए जाते हैं ताकि लचीलापन, CRDT कन्वर्जेंस, और भुगतान कतार की पुष्टि की जा सके।
- **बहुभाषी कियोस्क UI** — ग्राहक सत्र की शुरुआत में 17+ समर्थित भाषाओं में से अपनी पसंदीदा भाषा चुनते हैं (English, Spanish, Chinese, French, Japanese, Korean, Hindi, Arabic, Portuguese, Russian, Bengali, German, Urdu, Turkish, Traditional Chinese, Vietnamese, Thai, और अन्य)। सभी UI टेक्स्ट, रसीदें, और ऑडियो प्रॉम्प्ट चयनित लोकेल में रेंडर होते हैं।

---

## वास्तुकला

Astra-System को **क्लाउड टियर** और **स्टोर एज / कियोस्क क्लस्टर** में विभाजित किया गया है।

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

पूर्ण टोपोलॉजी, सुरक्षा मॉडल, भुगतान प्रवाह, ऑब्ज़र्वेबिलिटी, और डिज़ास्टर-रिकवरी विवरण के लिए, [`ARCHITECTURE.md`](./ARCHITECTURE.md) देखें।

### सर्विस इन्वेंट्री

| सर्विस           | भाषा      | ज़िम्मेदारी                                        |
| ----------------- | ---------- | ------------------------------------------------ |
| `api-gateway`     | Go         | एज राउटिंग, authN/authZ, रेट लिमिटिंग            |
| `order-svc`       | Go         | ऑर्डर लाइफ़साइकल, कार्ट पर्सिस्टेंस, फ़ुलफ़िलमेंट |
| `payment-svc`     | Go         | भुगतान ऑर्केस्ट्रेशन, टोकन सेटलमेंट               |
| `inventory-svc`   | Go         | स्टॉक लेवल, सॉफ्ट होल्ड्स, कैटलॉग सिंक           |
| `cart-svc`        | Go         | कार्ट CRDT विलय, ghost-cart रिज़ॉल्यूशन          |
| `sync-svc`        | Go         | क्लाउड-साइड मेश गेटवे और बैच इन्जेशन            |
| `astra-syncd`     | Rust       | कियोस्क P2P डेमन, CRDT सिंक, Verifone FFI ब्रिज   |
| `kiosk-shell`     | TypeScript | React 19 ग्राहक UI, परिधि एकीकरण                  |
| `update-server`   | Go         | साइन्ड OTA मैनिफेस्ट डिलीवरी                      |

---

## प्रौद्योगिकी स्टैक

- **फ्रंटएंड** — TypeScript, React 19, Vite, Module Federation, XState v5, Zustand, TanStack Query, Tailwind CSS (apps में v4, डिज़ाइन सिस्टम में v3)।
- **बैकएंड** — Go (Fiber / gRPC), PostgreSQL 16, Redis 7, NATS JetStream।
- **एज** — Rust (`astra-syncd`, `astra-verifone-ffi`), SQLite (SQLCipher), libp2p।
- **ML** — ONNX Runtime, TensorFlow Lite।
- **इन्फ्रा** — Kubernetes, Docker / Podman, Traefik, HashiCorp Vault, Nix flake।
- **ऑब्ज़र्वेबिलिटी** — Prometheus, Grafana, Loki, Jaeger, OpenTelemetry।

---

## रिपॉज़िटरी लेआउट

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

## शुरुआत कैसे करें

### आवश्यकताएँ

- **Node.js 22** और **pnpm 9+**
- **Go 1.25**
- **Rust 1.82** (सिंक डेमन बनाने के लिए `protoc` के साथ)
- **Docker** और **Docker Compose**
- *(वैकल्पिक)* पूरी तरह से पुनरुत्पादनीय टूलचेन के लिए **Nix**:

  ```bash
  nix develop
  ```

### त्वरित आरंभ

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

सर्विस चलाने से पहले `.env.example` को `.env` में कॉपी करें और आवश्यकतानुसार मान समायोजित करें।

---

### इंस्टॉलर

macOS, Linux, और Windows के लिए पूर्व-निर्मित परीक्षण बाइनरी [Releases पेज](https://github.com/xdfkenny/Astra-System/releases) पर उपलब्ध हैं।

| प्लेटफ़ॉर्म        | बाइनरी                         |
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

## विकास कार्यप्रवाह

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

Turborepo फ़िल्टर के माध्यम से एकल पैकेज चलाएँ:

```bash
pnpm turbo run dev --filter=@astra/kiosk
pnpm turbo run test --filter=@astra/kiosk
```

### Go सर्विस

```bash
cd astra-service/services
go test -race ./...
go vet ./...
```

### Rust डेमन

```bash
cd astra-service/sync-daemon
cargo test
cargo clippy -- -D warnings
cargo fmt --check
```

---

## बिल्ड और रन

### Protobuf जेनरेशन

```bash
cd proto
buf generate        # or: protoc as documented in proto/README.md
```

### स्थानीय पूर्ण स्टैक

```bash
docker compose up -d
pnpm dev            # kiosk-shell hot reload
```

प्रोडक्शन मैनिफ़ेस्ट के लिए, `docker-compose.prod.yml` का उपयोग करें।

---

## परीक्षण

| परत        | टूलिंग                                            |
| ------------ | -------------------------------------------------- |
| यूनिट (TS)    | Vitest + happy-dom                                 |
| E2E (TS)     | Playwright `kiosk-shell` के विरुद्ध                |
| यूनिट (Go)    | `go test -race ./...`                              |
| यूनिट (Rust)  | `cargo test`, `cargo clippy`                       |
| इंटीग्रेशन   | Docker Compose स्टैक (PostgreSQL, Redis, NATS)     |
| केओस         | इंटीग्रेशन के दौरान नेटवर्क-पार्टीशन इंजेक्शन     |

> इंटीग्रेशन और केओस परीक्षणों के लिए `postgres`, `redis`, और `nats` कंटेनर चलाने वाले Docker की आवश्यकता होती है।

---

## दस्तावेज़ीकरण

पूरा दस्तावेज़ीकरण [`docs/`](./docs/) में उपलब्ध है:

| अनुभाग | सामग्री |
|---------|----------|
| **वास्तुकला** | [अवलोकन](./docs/architecture/overview.md), [सिस्टम डिज़ाइन](./docs/architecture/system-design.md), [ऑफ़लाइन-प्रथम रणनीति](./docs/architecture/offline-first.md), [सुरक्षा मॉडल](./docs/architecture/security-model.md) |
| **बैकएंड** | [माइक्रोसर्विस](./docs/backend/microservices.md), [API गेटवे](./docs/backend/api-gateway.md), [REST API](./docs/backend/rest-api.md), [gRPC API](./docs/backend/grpc-api.md), [भुगतान ऑर्केस्ट्रेटर](./docs/backend/payment-orchestrator.md) |
| **फ्रंटएंड** | [माइक्रो-फ्रंटएंड](./docs/frontend/micro-frontends.md), [कियोस्क ऐप्स](./docs/frontend/kiosk-apps.md), [स्टेट प्रबंधन](./docs/frontend/state-management.md) |
| **डेटाबेस** | [स्कीमा](./docs/database/schema.md), [माइग्रेशन](./docs/database/migrations.md), [एंटिटीज़](./docs/database/entities.md) |
| **इन्फ्रास्ट्रक्चर** | [Docker](./docs/infrastructure/docker.md), [Kubernetes](./docs/infrastructure/kubernetes.md), [ऑब्ज़र्वेबिलिटी](./docs/infrastructure/monitoring.md), [CI/CD](./docs/infrastructure/ci-cd.md) |
| **नेटवर्किंग** | [P2P मेश](./docs/networking/p2p-mesh.md), [प्रोटोकॉल](./docs/networking/protocols.md) |
| **सुरक्षा** | [अवलोकन](./docs/security/overview.md), [प्रमाणीकरण](./docs/security/authentication.md), [एन्क्रिप्शन](./docs/security/encryption.md) |

प्रमुख संदर्भ:
- [`ARCHITECTURE.md`](./ARCHITECTURE.md) — सिस्टम डिज़ाइन, सुरक्षा मॉडल, भुगतान प्रवाह, ऑब्ज़र्वेबिलिटी, और DR।
- [`UX_UI_AUDIT_REPORT.md`](./astra-service/UX_UI_AUDIT_REPORT.md) — "Living Weave" बायोफिलिक कियोस्क UI डिज़ाइन स्पेसिफिकेशन।
- [`docs/API-BACKEND-ASTRA.md`](./docs/API-BACKEND-ASTRA.md) — संपूर्ण API एंडपॉइंट इन्वेंट्री।
- [`docs/Readme Translations/`](./docs/Readme Translations/) — 17+ भाषाओं में समुदाय-योगदानित README अनुवाद।
- [`docs/runbooks/`](./docs/runbooks/) — संचालन रनबुक्स (घटना प्रतिक्रिया, ऑफ़लाइन मोड, P2P रिकवरी, भुगतान विफलता)।

---

## योगदान

1. सभी कमिट संदेशों के लिए [Conventional Commits](https://www.conventionalcommits.org/) का पालन करें।
2. Lefthook प्री-कमिट हुक इंस्टॉल करने के लिए `pnpm prepare` चलाएँ।
3. पुल अनुरोध खोलने से पहले सुनिश्चित करें कि `lint → typecheck → test` सभी पास हों।
4. परिवर्तनों को पाथ-स्कोप्ड रखें; CI पाथ-फ़िल्टर्ड है और केवल प्रासंगिक टूलचेन चलाता है।

---

## लाइसेंस

[Apache License, Version 2.0](../../LICENSE) के तहत लाइसेंस प्राप्त।

---

<p align="center">
  <sub>Astra-System · लचीले, ऑफ़लाइन-प्रथम खुदरा के लिए बनाया गया।</sub>
</p>
