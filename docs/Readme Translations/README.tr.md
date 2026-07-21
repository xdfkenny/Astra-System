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
  <a href="./README.tr.md"><b>Türkçe</b></a> ·
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

> 7/24 perakende ortamları için üretilmiş, çevrimdışı öncelikli otomatik self-checkout platformu.

**Astra-System**, gözetimli ve gözetimsiz self-checkout kiosk'larını çalıştıran çok dilli bir monorepo'dur. **48 saat çevrimdışı dayanıklılık** ile sıfır kesinti süresi mağaza işletmesi, sıfır güven güvenlik modeli ve her kiosk'u bulut erişilemez olduğunda bile tutarlı kalan eşler arası mesh senkronizasyon katmanı sunar.

---

## İçindekiler

- [Genel Bakış](#overview)
- [Temel Özellikler](#key-features)
- [Mimari](#architecture)
- [Teknoloji Yığını](#technology-stack)
- [Depo Düzeni](#repository-layout)
- [Başlangıç](#getting-started)
- [Geliştirme İş Akışı](#development-workflow)
- [Derleme ve Çalıştırma](#build--run)
- [Test Etme](#testing)
- [Dokümantasyon](#documentation)
- [Katkı Sağlama](#contributing)
- [Lisans](#license)

---

## Genel Bakış

Astra-System, perakendecilerin internet bağlantısı olmadan **48 saate kadar bağımsız çalışabilen** self-checkout kiosk filoları dağıtmasını sağlar. Dayanıklılık üç katmanda katmanlıdır:

1. **Yerel veri katmanı** — Her kiosk'ta şifreli bir SQLite (SQLCipher) deposu; tam menü kataloğu, envanter, bekleyen işlemler ve çevrimdışı ödeme belirteçleri içerir.
2. **Eşler arası mesh** — Kiosk'lar yerel ağ üzerinden birbirlerini keşfeder (mDNS + libp2p/QUIC) ve CRDT'ler kullanarak durumu çoğaltır; üç veya daha fazla kiosk olduğunda bir Raft lideri seçer.
3. **Yumuşak bozulma** — Ödemeler, envanter ve sipariş yakalama yerel olarak devam eder ve bağlantıcılık geri döndüğünde bulut ile uzlaşma sağlar.

Bulut katmanı (Go mikro servisleri, PostgreSQL 16, Redis 7, NATS JetStream) yetkili olay kaynaklı depoyu, mutabakatı ve filo yönetimini sağlar.

### Tasarım Hedefleri

| Hedef | Hedef |
| ------------------ | ---------------------------------------------------------------------- |
| Çevrimdışı dayanıklılık | Bulut bağlantısı olmadan 48 saat bağımsız çalışma |
| Gecikme süresi | Menü yükleme < 200 ms, P2P envanter senkronizasyonu < 500 ms, lider devralma < 3 saniye |
| Kullanılabilirlik | %99.99 çalışma süresi (bulut katmanı); yalnızca yerel modda %100 çalışma süresi |
| Güvenlik | Sıfır güven, her yerde mTLS, PCI-DSS uyumlu ödeme yolu |
| Ölçek | Kiracı başına 1–10.000 kiosk; çok bölgeli bulut dağıtımı |

---

## Temel Özellikler

- **Çevrimdışı öncelikli motor** — Kiosk'lar arası nedensel sıralama için Hibrit Mantiksel Saatler ile deterministik CRDT birleştirme (PN-Counter, LWW-Register, OR-Set).
- **P2P mesh ve fikir birliği** — libp2p QUIC aktarımı, Noise protokolü şifrelemesi ve 3 saniyenin altında lider devralma.
- **İşlemsel dış kutu** — NATS JetStream üzerinden bulut servislerinden kesin olarak bir kez olay yayını.
- **Sıfır güven güvenlik** — mTLS, kiosk başına HMAC imzası, SPIFFE kimlikleri ve PCI-DSS uyumlu ödeme yolu (kart verileri kiosk belleğine as嚆 dokunmaz).
- **Verifone FFI köprüsü** — Ödeme terminali entegrasyonu için satıcı C SDK üzerinde güvenli bir Rust sarmalayıcısı (`astra-verifone-ffi`).
- **Biyofilik kiosk arayüzü** — Module Federation, XState v5 iş akışı makinesi ve Zustand/TanStack Query durum yönetimi ile oluşturulmuş React 19 mikro ön yüzü.
- **İleri düzey zeka** — Ghost Carts, ürün tanıma (ONNX), şerit zekası (TFLite), WebAuthn/passkeys ve diferansiyel gizlilik analitiği.
- **Kaosa hazır CI** — Entegrasyon testleri sırasında ağ bölmeleri enjekte edilerek dayanıklılık, CRDT yakınsaması ve ödeme kuyruğu doğrulanır.
- **Çok dilli kiosk arayüzü** — Müşteriler oturum başında 17+ desteklenen dilden (İngilizce, İspanyolca, Çince, Fransızca, Japonca, Korece, Hintçe, Arapça, Portekizce, Rusça, Bengalce, Almanca, Urduca, Türkçe, Geleneksel Çince, Vietnamca, Tayca ve daha fazlası) tercih ettikleri dili seçer. Tüm UI metinleri, makbuzları ve sesli komutlar seçilen dilde görüntülenir.

---

## Mimari

Astra-System **Bulut Katmanı** ve **Mağaza Kenarı / Kiosk Kümesi** olarak ikiye ayrılır.

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

Tam topoloji, güvenlik modeli, ödeme akışları, gözlemlenebilirlik ve felaket kurtarma detayları için [`ARCHITECTURE.md`](../../ARCHITECTURE.md) dosyasına bakın.

### Servis Envanteri | Servis | Dil | Sorumluluk |
| ----------------- | ---------- | ------------------------------------------------ |
| `api-gateway` | Go | Kenar yönlendirme, authN/authZ, hız sınırlama |
| `order-svc` | Go | Sipariş yaşam döngüsü, sepet kalıcılığı, teslimat |
| `payment-svc` | Go | Ödeme orkestirasyonu, belirteç mutabakatı |
| `inventory-svc` | Go | Stok seviyeleri, yumuşak tutmalar, katalog senkronizasyonu |
| `cart-svc` | Go | Sepet CRDT birleştirme, hayalet sepet çözümleme |
| `sync-svc` | Go | Bulut tarafı mesh ağ geçidi ve toplu alma |
| `astra-syncd` | Rust | Kiosk P2P daemon, CRDT senkronizasyonu, Verifone FFI köprüsü |
| `kiosk-shell` | TypeScript | React 19 müşteri arayüzü, çevre birimi entegrasyonu |
| `update-server` | Go | İmzalı OTA manifest dağıtımı |

---

## Teknoloji Yığını

- **Ön yüz** — TypeScript, React 19, Vite, Module Federation, XState v5, Zustand, TanStack Query, Tailwind CSS (uygulamalarda v4, tasarım sisteminde v3).
- **Arka yüz** — Go (Fiber / gRPC), PostgreSQL 16, Redis 7, NATS JetStream.
- **Kenar** — Rust (`astra-syncd`, `astra-verifone-ffi`), SQLite (SQLCipher), libp2p.
- **ML** — ONNX Runtime, TensorFlow Lite.
- **Altyapı** — Kubernetes, Docker / Podman, Traefik, HashiCorp Vault, Nix flake.
- **Gözlemlenebilirlik** — Prometheus, Grafana, Loki, Jaeger, OpenTelemetry.

---

## Depo Düzeni

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

## Başlangıç

### Ön Gereksinimler

- **Node.js 22** ve **pnpm 9+**
- **Go 1.25**
- **Rust 1.82** (senkronizasyon daemon'ını derlemek için `protoc` ile birlikte)
- **Docker** ve **Docker Compose**
- *(İsteğe bağlı)* Tamamen tekrarlanabilir bir araç zinciri için **Nix**:

  ```bash
  nix develop
  ```

### Hızlı Başlangıç

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

Servisleri çalıştırmadan önce `.env.example` dosyasını `.env` olarak kopyalayın ve değerleri ihtiyaca göre ayarlayın.

---

### Yükleyici

macOS, Linux ve Windows için önceden derlenmiş test dosyaları [Sürüm sayfasında](https://github.com/xdfkenny/Astra-System/releases) mevcuttur.

| Platform | Dosya |
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

## Geliştirme İş Akışı

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

Turborepo filtreleri aracılığıyla tek bir paket çalıştırın:

```bash
pnpm turbo run dev --filter=@astra/kiosk
pnpm turbo run test --filter=@astra/kiosk
```

### Go servisleri

```bash
cd astra-service/services
go test -race ./...
go vet ./...
```

### Rust daemon'ları

```bash
cd astra-service/sync-daemon
cargo test
cargo clippy -- -D warnings
cargo fmt --check
```

---

## Derleme ve Çalıştırma

### Protobuf oluşturma

```bash
cd proto
buf generate        # or: protoc as documented in proto/README.md
```

### Yerel tam yığın

```bash
docker compose up -d
pnpm dev            # kiosk-shell hot reload
```

Üretim manifestleri için `docker-compose.prod.yml` dosyasını kullanın.

---

## Test Etme

| Katman | Araçlar |
| ------------ | -------------------------------------------------- |
| Birim (TS) | Vitest + happy-dom |
| Uçtan Uca (TS) | Playwright `kiosk-shell`'e karşı |
| Birim (Go) | `go test -race ./...` |
| Birim (Rust) | `cargo test`, `cargo clippy` |
| Entegrasyon | Docker Compose yığını (PostgreSQL, Redis, NATS) |
| Kaos | Entegrasyon sırasında ağ bölmeleri enjeksiyonu |

> Entegrasyon ve kaos testleri, `postgres`, `redis` ve `nats`容器ları ile çalışan Docker gerektirir.

---

## Dokümantasyon

Tam dokümantasyon [`docs/`](../../docs/) dizininde mevcuttur:

| Bölüm | İçerik |
|---------|----------|
| **Mimari** | [Genel Bakış](../../docs/architecture/overview.md), [Sistem Tasarımı](../../docs/architecture/system-design.md), [Çevrimdışı Öncelikli Strateji](../../docs/architecture/offline-first.md), [Güvenlik Modeli](../../docs/architecture/security-model.md) |
| **Arka yüz** | [Mikro Servisler](../../docs/backend/microservices.md), [API Ağ Geçidi](../../docs/backend/api-gateway.md), [REST API](../../docs/backend/rest-api.md), [gRPC API](../../docs/backend/grpc-api.md), [Ödeme Orkestraörü](../../docs/backend/payment-orchestrator.md) |
| **Ön yüz** | [Mikro Ön Yüzler](../../docs/frontend/micro-frontends.md), [Kiosk Uygulamaları](../../docs/frontend/kiosk-apps.md), [Durum Yönetimi](../../docs/frontend/state-management.md) |
| **Veritabanı** | [Şema](../../docs/database/schema.md), [Göçler](../../docs/database/migrations.md), [Varlıklar](../../docs/database/entities.md) |
| **Altyapı** | [Docker](../../docs/infrastructure/docker.md), [Kubernetes](../../docs/infrastructure/kubernetes.md), [Gözlemlenebilirlik](../../docs/infrastructure/monitoring.md), [CI/CD](../../docs/infrastructure/ci-cd.md) |
| **Ağ** | [P2P Mesh](../../docs/networking/p2p-mesh.md), [Protokoller](../../docs/networking/protocols.md) |
| **Güvenlik** | [Genel Bakış](../../docs/security/overview.md), [Kimlik Doğrulama](../../docs/security/authentication.md), [Şifreleme](../../docs/security/encryption.md) |

Temel referanslar:
- [`ARCHITECTURE.md`](../../ARCHITECTURE.md) — Sistem tasarımı, güvenlik modeli, ödeme akışları, gözlemlenebilirlik ve felaket kurtarma.
- [`UX_UI_AUDIT_REPORT.md`](../../astra-service/UX_UI_AUDIT_REPORT.md) — "Yaşayan Dokuma" biyofilik kiosk UI tasarım spesifikasyonu.
- [`docs/API-BACKEND-ASTRA.md`](../../docs/API-BACKEND-ASTRA.md) — Tam API uç noktası envanteri.
- [`docs/Readme Translations/`](../../docs/Readme Translations/) — 17+ dilde topluluk katkıları ile README çevirileri.
- [`docs/runbooks/`](../../docs/runbooks/) — Operasyonel runbook'lar (olay müdahalesi, çevrimdışı mod, P2P kurtarma, ödeme hatası).

---

## Katkı Sağlama

1. Tüm commit mesajları için [Conventional Commits](https://www.conventionalcommits.org/) kurallarına uyun.
2. Lefthook pre-commit hook'larını yüklemek için `pnpm prepare` çalıştırın.
3. Bir pull request açmadan önce `lint ← typecheck ← test` adımlarının hepsinin geçtiğinden emin olun.
4. Değişiklikleri yola özgü tutun; CI yola göre filtrelenir ve yalnızca ilgili araç zincirlerini çalıştırır.

---

## Lisans

[Lisans Apache, Sürüm 2.0](../../LICENSE) altında lisanslanmıştır.

---

<p align="center">
  <sub>Astra-System · Dayanıklı, çevrimdışı öncelikli perakende için tasarlandı.</sub>
</p>
