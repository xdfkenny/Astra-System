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
  <a href="./README.pt.md"><b>Português</b></a> ·
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

> Plataforma de autoatendimento automatizada de nível de produção, projetada para operar em primeiro lugar offline em ambientes de varejo 24/7.

**Astra-System** é um monorepo multilíngue que suporta quiosques de autoatendimento não supervisionados e supervisionados. Ele fornece operação de loja sem tempo de inatividade com **48 horas de resiliência offline**, um modelo de segurança zero-trust e uma camada de sincronização mesh peer-to-peer que mantém cada quiosque em uma loja consistente — mesmo quando a nuvem está inacessível.

---

## Índice

- [Visão Geral](#visão-geral)
- [Principais Funcionalidades](#principais-funcionalidades)
- [Arquitetura](#arquitetura)
- [Pilha Tecnológica](#pilha-tecnológica)
- [Estrutura do Repositório](#estrutura-do-repositório)
- [Primeiros Passos](#primeiros-passos)
- [Fluxo de Trabalho de Desenvolvimento](#fluxo-de-trabalho-de-desenvolvimento)
- [Compilação e Execução](#compilação-e-execução)
- [Testes](#testes)
- [Documentação](#documentação)
- [Contribuir](#contribuir)
- [Licença](#licença)

---

## Visão Geral

O Astra-System permite que varejistas implantem frotas de quiosques de autoatendimento que operam **de forma autônoma por até 48 horas** sem conexão com a internet. A resiliência é distribuída em três camadas:

1. **Camada de dados local** — um armazenamento SQLite criptografado (SQLCipher) em cada quiosque com o catálogo completo de menu, estoque, transações pendentes e tokens de pagamento offline.
2. **Mesh peer-to-peer** — os quiosques se descobrem na rede local (mDNS + libp2p/QUIC) e replicam o estado usando CRDTs, elegendo um líder Raft quando três ou mais estão presentes.
3. **Degradção elegante** — pagamentos, estoque e captura de pedidos continuam localmente e são reconciliados com a nuvem quando a conectividade retorna.

A camada de nuvem (microsserviços Go, PostgreSQL 16, Redis 7, NATS JetStream) fornece o armazenamento de eventos como fonte de verdade, liquidação e gerenciamento de frota.

### Objetivos de Design

| Objetivo | Meta |
| --- | --- |
| Resiliência offline | 48 horas de operação autônoma sem conectividade com a nuvem |
| Latência | < 200 ms carga de menu, < 500 ms sincronização P2P de estoque, < 3 s failover do líder |
| Disponibilidade | 99,99% de uptime (camada de nuvem); 100% de uptime no modo apenas local |
| Segurança | Zero-trust, mTLS em todos os lugares, caminho de pagamento compatível com PCI-DSS |
| Escala | 1–10.000 quiosques por inquilino; implantação de nuvem multirregião |

---

## Principais Funcionalidades

- **Motor offline-first** — fusão determinística de CRDT (PN-Counter, LWW-Register, OR-Set) com Relógios Lógicos Híbridos para ordenação causal entre quiosques.
- **Mesh P2P e consenso Raft** — transporte QUIC do libp2p, criptografia Noise Protocol e failover do líder em menos de 3 segundos.
- **Outbox transacional** — publicação de eventos exatamente uma vez a partir de serviços na nuvem via NATS JetStream.
- **Segurança zero-trust** — mTLS, assinatura HMAC por quiosque, identidades SPIFFE e caminho de pagamento compatível com PCI-DSS (dados do cartão nunca tocam a memória do quiosque).
- **Bridge Verifone FFI** — um wrapper Rust seguro sobre o SDK C do fabricante para integração de terminais de pagamento.
- **UI biofílica de quiosque** — micro-frontend React 19 construído com Module Federation, máquina de estados XState v5 e gerenciamento de estado Zustand/TanStack Query.
- **Inteligência avançada** — Ghost Carts, reconhecimento de produtos (ONNX), inteligência de faixa (TFLite), WebAuthn/passkeys e analíticos de privacidade diferencial.
- **CI preparado para caos** — partições de rede são injetadas durante testes de integração para verificar resiliência, convergência de CRDT e enfileiramento de pagamentos.
- **UI multilíngue de quiosque** — os clientes selecionam seu idioma preferido no início da sessão entre mais de 17 idiomas suportados. Todo o texto da UI, recibos e prompts de áudio são renderizados no idioma selecionado.

---

## Arquitetura

O Astra-System é dividido em uma **Camada de Nuvem** e um **Cluster de Borda da Loja / Quiosques**.

Para a topologia completa, modelo de segurança, fluxos de pagamento, observabilidade e detalhes de recuperação de desastres, consulte [`ARCHITECTURE.md`](../../ARCHITECTURE.md).

### Inventário de Serviços

| Serviço | Linguagem | Responsabilidade |
| --- | --- | --- |
| `api-gateway` | Go | Roteamento de borda, authN/authZ, limitação de taxa |
| `order-svc` | Go | Ciclo de vida do pedido, persistência do carrinho, fulfillment |
| `payment-svc` | Go | Orquestração de pagamentos, liquidação de tokens |
| `inventory-svc` | Go | Níveis de estoque, retenções suaves, sincronização de catálogo |
| `cart-svc` | Go | Fusão CRDT do carrinho, resolução Ghost Carts |
| `sync-svc` | Go | Gateway mesh do lado da nuvem e ingestão em lote |
| `astra-syncd` | Rust | Daemon P2P do quiosque, sincronização CRDT, bridge Verifone FFI |
| `kiosk-shell` | TypeScript | UI do cliente React 19, integração de periféricos |
| `update-server` | Go | Entrega de manifestos OTA assinados |

---

## Pilha Tecnológica

- **Frontend** — TypeScript, React 19, Vite, Module Federation, XState v5, Zustand, TanStack Query, Tailwind CSS.
- **Backend** — Go (Fiber / gRPC), PostgreSQL 16, Redis 7, NATS JetStream.
- **Borda** — Rust (`astra-syncd`, `astra-verifone-ffi`), SQLite (SQLCipher), libp2p.
- **ML** — ONNX Runtime, TensorFlow Lite.
- **Infra** — Kubernetes, Docker / Podman, Traefik, HashiCorp Vault, Nix flake.
- **Observabilidade** — Prometheus, Grafana, Loki, Jaeger, OpenTelemetry.

---

## Estrutura do Repositório

```text
astra-service/          Código de serviços e aplicações
  apps/                 Micro-frontends TypeScript (kiosk-shell, kiosk-menu, …)
  packages/             Bibliotecas compartilhadas e design system
  services/             Microsserviços Go
  sync-daemon/          astra-syncd (Rust) daemon P2P
  daemons/              Daemons sidecar (payment-sidecar)
  tools/                Ferramentas operacionais (chaos, etc.)
services/               Serviços independentes (update-server, …)
database/               Migrações de esquema
proto/                  Definições Protocol Buffer e código gerado
docs/                   Runbooks operacionais
infra/                  Ferramentas de infraestrutura e helpers de segredos
.github/                Workflows CI e arquivos da comunidade
flake.nix               Shell de desenvolvimento Nix reproduzível
docker-compose*.yml     Manifestos compose local e de produção
```

---

## Primeiros Passos

### Pré-requisitos

- **Node.js 22** e **pnpm 9+**
- **Go 1.25**
- **Rust 1.82** (com `protoc` para compilar o daemon de sincronização)
- **Docker** e **Docker Compose**
- *(Opcional)* **Nix** para uma cadeia de ferramentas totalmente reproduzível:

  ```bash
  nix develop
  ```

### Início Rápido

```bash
# 1. Instalar dependências do frontend
pnpm install

# 2. Iniciar a pilha backend local (PostgreSQL, Redis, NATS)
docker compose up -d

# 3. Executar todas as aplicações TypeScript com hot reload
pnpm dev

# 4. Compilar o daemon de sincronização Rust
cd astra-service/sync-daemon && cargo build --release
```

Copie `.env.example` para `.env` e ajuste os valores conforme necessário antes de executar os serviços.

---

### Instalador

Bins de teste pré-compilados para macOS, Linux e Windows estão disponíveis na [página de Releases](https://github.com/xdfkenny/Astra-System/releases).

| Plataforma | Binário |
| --- | --- |
| macOS (Intel) | `astra-installer-darwin-amd64` |
| macOS (Apple Silicon) | `astra-installer-darwin-arm64` |
| Linux (x86_64) | `astra-installer-linux-amd64` |
| Linux (ARM64) | `astra-installer-linux-arm64` |
| Windows (x86_64) | `astra-installer-windows-amd64.exe` |

```bash
# macOS / Linux — baixar e executar o script de bootstrap
curl -sL https://raw.githubusercontent.com/xdfkenny/Astra-System/main/installer/scripts/install.sh | bash

# Ou baixar um binário diretamente do Releases, torná-lo executável e executar:
./astra-installer-<platform>
```

---

## Fluxo de Trabalho de Desenvolvimento

```bash
# Lint, typecheck e testes (a ordem importa)
pnpm lint
pnpm typecheck
pnpm test

# Testes end-to-end (Playwright)
pnpm test:e2e

# Formatar
pnpm format && pnpm format:check

# Compilar todos os pacotes
pnpm build
```

Executar um único pacote via filtros do Turborepo:

```bash
pnpm turbo run dev --filter=@astra/kiosk
pnpm turbo run test --filter=@astra/kiosk
```

### Serviços Go

```bash
cd astra-service/services
go test -race ./...
go vet ./...
```

### Daemons Rust

```bash
cd astra-service/sync-daemon
cargo test
cargo clippy -- -D warnings
cargo fmt --check
```

---

## Compilação e Execução

### Geração de Protobuf

```bash
cd proto
buf generate
```

### Pilha local completa

```bash
docker compose up -d
pnpm dev
```

Para manifestos de produção, use `docker-compose.prod.yml`.

---

## Testes

| Camada | Ferramentas |
| --- | --- |
| Unitário (TS) | Vitest + happy-dom |
| E2E (TS) | Playwright contra `kiosk-shell` |
| Unitário (Go) | `go test -race ./...` |
| Unitário (Rust) | `cargo test`, `cargo clippy` |
| Integração | Pilha Docker Compose (PostgreSQL, Redis, NATS) |
| Caos | Injeção de partições de rede durante integração |

> Os testes de integração e caos requerem Docker com os contêineres `postgres`, `redis` e `nats` em execução.

---

## Documentação

A documentação completa está disponível em [`docs/`](../../docs/):

| Seção | Conteúdo |
| --- | --- |
| **Arquitetura** | [Visão Geral](../../docs/architecture/overview.md), [Design do Sistema](../../docs/architecture/system-design.md), [Estratégia Offline-First](../../docs/architecture/offline-first.md), [Modelo de Segurança](../../docs/architecture/security-model.md) |
| **Backend** | [Microsserviços](../../docs/backend/microservices.md), [API Gateway](../../docs/backend/api-gateway.md), [API REST](../../docs/backend/rest-api.md), [API gRPC](../../docs/backend/grpc-api.md), [Orquestrador de Pagamentos](../../docs/backend/payment-orchestrator.md) |
| **Frontend** | [Micro-Frontends](../../docs/frontend/micro-frontends.md), [Aplicativos de Quiosque](../../docs/frontend/kiosk-apps.md), [Gerenciamento de Estado](../../docs/frontend/state-management.md) |
| **Banco de Dados** | [Esquema](../../docs/database/schema.md), [Migrações](../../docs/database/migrations.md), [Entidades](../../docs/database/entities.md) |
| **Infraestrutura** | [Docker](../../docs/infrastructure/docker.md), [Kubernetes](../../docs/infrastructure/kubernetes.md), [Observabilidade](../../docs/infrastructure/monitoring.md), [CI/CD](../../docs/infrastructure/ci-cd.md) |
| **Rede** | [Mesh P2P](../../docs/networking/p2p-mesh.md), [Protocolos](../../docs/networking/protocols.md) |
| **Segurança** | [Visão Geral](../../docs/security/overview.md), [Autenticação](../../docs/security/authentication.md), [Criptografia](../../docs/security/encryption.md) |

Referências principais:
- [`ARCHITECTURE.md`](../../ARCHITECTURE.md) — design do sistema, modelo de segurança, fluxos de pagamento, observabilidade e DR.
- [`UX_UI_AUDIT_REPORT.md`](../../astra-service/UX_UI_AUDIT_REPORT.md) — a especificação de design da UI biofílica "Living Weave".
- [`docs/API-BACKEND-ASTRA.md`](../../docs/API-BACKEND-ASTRA.md) — inventário completo de endpoints da API.
- [`docs/Readme Translations/`](../../docs/Readme%20Translations/) — traduções de README contribuídas pela comunidade em 17+ idiomas.
- [`docs/runbooks/`](../../docs/runbooks/) — runbooks operacionais (resposta a incidentes, modo offline, recuperação P2P, falha de pagamento).

---

## Contribuir

1. Siga [Conventional Commits](https://www.conventionalcommits.org/) para todas as mensagens de commit.
2. Execute `pnpm prepare` para instalar os hooks pre-commit do Lefthook.
3. Certifique-se de que `lint → typecheck → test` passem antes de abrir um pull request.
4. Mantenha as alterações com escopo de caminho; a CI é filtrada por caminho e só executa as cadeias de ferramentas relevantes.

---

## Licença

Licenciado sob a [Apache License, Version 2.0](../../LICENSE).

---

<p align="center">
  <sub>Astra-System · Construído para varejo resiliente e offline-first.</sub>
</p>
