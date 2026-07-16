# Astra-System

<p align="center">
  <img src="https://raw.githubusercontent.com/cat-milk/Anime-Girls-Holding-Programming-Books/master/Typescript/Beako_Reading_The_TypeScript_Programming_Language.png" width="420" alt="Anime girl reading the TypeScript programming language book" />
</p>

<p align="center">
  <a href="../README.md">English</a> ·
  <a href="./README.es.md"><b>Español</b></a> ·
  <a href="./README.zh.md">中文</a> ·
  <a href="./README.ko.md">한국어</a> ·
  <a href="./README.ja.md">日本語</a>
</p>

[![CI](https://img.shields.io/badge/CI-pass-green.svg)](https://github.com/anomalyco/astra-system/actions)
[![License](https://img.shields.io/badge/license-Apache%202.0-blue.svg)](../LICENSE)
[![Go](https://img.shields.io/badge/Go-1.25-00ADD8.svg)](https://go.dev)
[![Rust](https://img.shields.io/badge/Rust-1.82-dea584.svg)](https://www.rust-lang.org)
[![TypeScript](https://img.shields.io/badge/TypeScript-5.x-3178C6.svg)](https://www.typescriptlang.org)

> Plataforma de autopago automatizado, de grado producción y con prioridad en el modo sin conexión, diseñada para entornos comerciales 24/7.

**Astra-System** es un monorepositorio multilenguaje que impulsa quioscos de autopago sin atención y con atención. Ofrece operación de la tienda sin tiempo de inactividad con **48 horas de resiliencia sin conexión**, un modelo de seguridad de confianza cero y una capa de sincronización en malla punto a punto que mantiene coherente cada quiosco de la tienda, incluso cuando la nube no está disponible.

---

## Tabla de Contenidos

- [Resumen](#resumen)
- [Características Principales](#características-principales)
- [Arquitectura](#arquitectura)
- [Pila Tecnológica](#pila-tecnológica)
- [Estructura del Repositorio](#estructura-del-repositorio)
- [Primeros Pasos](#primeros-pasos)
- [Flujo de Trabajo de Desarrollo](#flujo-de-trabajo-de-desarrollo)
- [Compilación y Ejecución](#compilación-y-ejecución)
- [Pruebas](#pruebas)
- [Documentación](#documentación)
- [Cómo Contribuir](#cómo-contribuir)
- [Licencia](#licencia)

---

## Resumen

Astra-System permite a los comercios desplegar flotas de quioscos de autopago que operan **de forma autónoma hasta por 48 horas** sin conectividad a internet. La resiliencia se distribuye en tres niveles:

1. **Capa de datos local** — una base de datos SQLite cifrada (SQLCipher) en cada quiosco con el catálogo de menú completo, inventario, transacciones pendientes y tokens de pago sin conexión.
2. **Malla punto a punto** — los quioscos se descubren en la red local (mDNS + libp2p/QUIC) y replican estado mediante CRDT, eligiendo un líder Raft cuando hay tres o más presentes.
3. **Degradación elegante** — los pagos, el inventario y la captura de pedidos continúan localmente y se concilian con la nube al restablecerse la conectividad.

La capa en la nube (microservicios Go, PostgreSQL 16, Redis 7, NATS JetStream) proporciona el almacenamiento basado en eventos como fuente de verdad, la liquidación y la gestión de la flota.

### Objetivos de Diseño

| Objetivo          | Meta                                                                  |
| ----------------- | --------------------------------------------------------------------- |
| Resiliencia offline | 48 horas de operación autónoma sin conectividad a la nube            |
| Latencia          | < 200 ms carga de menú, < 500 ms sync P2P, < 3 s failover de líder  |
| Disponibilidad    | 99,99 % uptime (nube); 100 % uptime en modo solo local               |
| Seguridad         | Confianza cero, mTLS en todas partes, ruta de pago conforme a PCI-DSS |
| Escala            | 1–10 000 quioscos por inquilino; despliegue multirregión en la nube  |

---

## Características Principales

- **Motor offline-first** — fusión CRDT determinista (PN-Counter, LWW-Register, OR-Set) con Relojes Lógicos Híbridos para orden causal entre quioscos.
- **Malla P2P y consenso Raft** — transporte libp2p QUIC, cifrado con protocolo Noise y failover de líder menor a 3 segundos.
- **Outbox transaccional** — publicación de eventos exactamente una vez desde los servicios en la nube vía NATS JetStream.
- **Seguridad de confianza cero** — mTLS, firma HMAC por quiosco, identidades SPIFFE y ruta de pago conforme a PCI-DSS (los datos de tarjeta nunca tocan la memoria del quiosco).
- **Puente Verifone FFI** — un wrapper seguro en Rust (`astra-verifone-ffi`) sobre el SDK C del fabricante para integración con terminales de pago.
- **UI de quiosco biofílica** — un micro-frontend React 19 con Module Federation, máquina de estados XState v5 y gestión de estado con Zustand/TanStack Query.
- **Inteligencia avanzada** — Ghost Carts, reconocimiento de productos (ONNX), inteligencia de cola (TFLite), WebAuthn/passkeys y analítica con privacidad diferencial.
- **CI preparada para caos** — se inyectan particiones de red durante las pruebas de integración para verificar resiliencia, convergencia CRDT y encolado de pagos.

---

## Arquitectura

Astra-System se divide en una **Capa en la Nube** y un **Clúster de Borde de Tienda / Quioscos**.

```text
┌─────────────────────────────────────────────────────────────────┐
│                         Capa en la Nube                          │
│  API Gateway · Order Svc · Payment Svc · Inventory Svc ·       │
│  Cart Svc · Sync Svc · PostgreSQL 16 · Redis 7 · NATS JetStream │
└──────────────────────────────────┬──────────────────────────────┘
                                   │ TLS 1.3
┌──────────────────────────────────┴──────────────────────────────┐
│              Clúster de Borde de Tienda / Quioscos               │
│  Quiosco 1 ─┐  Quiosco 2 ─┐  Quiosco N ─┐                      │
│  React 19   │  React 19   │  React 19   │  (malla local QUIC)  │
│  Rust P2P   │  Rust P2P   │  Rust P2P   │                      │
│  SQLite     │  SQLite     │  SQLite     │                      │
│  Verifone · Impresora · Escáner · NFC/Balanza                   │
└─────────────────────────────────────────────────────────────────┘
```

Para la topología completa, el modelo de seguridad, los flujos de pago, la observabilidad y los detalles de recuperación ante desastres, consulte [`ARCHITECTURE.md`](./ARCHITECTURE.md).

### Inventario de Servicios

| Servicio         | Lenguaje   | Responsabilidad                                  |
| ---------------- | ---------- | ------------------------------------------------ |
| `api-gateway`    | Go         | Enrutamiento de borde, authN/authZ, rate limiting |
| `order-svc`      | Go         | Ciclo de vida de pedidos, carritos, cumplimiento |
| `payment-svc`    | Go         | Orquestación de pagos, liquidación de tokens     |
| `inventory-svc`  | Go         | Niveles de stock, reservas, sync de catálogo     |
| `cart-svc`       | Go         | Fusión CRDT de carritos, Ghost Carts             |
| `sync-svc`       | Go         | Pasarela de malla en la nube e ingesta por lotes |
| `astra-syncd`    | Rust       | Daemon P2P, sync CRDT, puente Verifone FFI       |
| `kiosk-shell`    | TypeScript | UI React 19 del cliente, integración periférica  |
| `update-server`  | Go         | Entrega firmada de manifiestos OTA               |

---

## Pila Tecnológica

- **Frontend** — TypeScript, React 19, Vite, Module Federation, XState v5, Zustand, TanStack Query, Tailwind CSS (v4 en apps, v3 en el design system).
- **Backend** — Go (Fiber / gRPC), PostgreSQL 16, Redis 7, NATS JetStream.
- **Borde** — Rust (`astra-syncd`, `astra-verifone-ffi`), SQLite (SQLCipher), libp2p.
- **ML** — ONNX Runtime, TensorFlow Lite.
- **Infra** — Kubernetes, Docker / Podman, Traefik, HashiCorp Vault, Nix flake.
- **Observabilidad** — Prometheus, Grafana, Loki, Jaeger, OpenTelemetry.

---

## Estructura del Repositorio

```text
astra-service/          Código de servicios y aplicaciones
  apps/                 Micro-frontends TypeScript (kiosk-shell, kiosk-menu, …)
  packages/             Bibliotecas compartidas y design system
  services/             Microservicios Go
  sync-daemon/          astra-syncd (Rust) daemon P2P
  daemons/              Daemons sidecar (payment-sidecar)
  tools/                Herramientas operativas (chaos, etc.)
database/               Migraciones de esquema
proto/                  Definiciones Protocol Buffer y código generado
docs/                   Runbooks operativos
infra/                  Herramientas de infraestructura y secretos
.github/                Workflows de CI y archivos de comunidad
flake.nix               Shell de desarrollo Nix reproducible
docker-compose*.yml     Manifiestos compose local y de producción
```

---

## Primeros Pasos

### Requisitos previos

- **Node.js 22** y **pnpm 9+**
- **Go 1.25**
- **Rust 1.82** (con `protoc` para compilar el daemon de sync)
- **Docker** y **Docker Compose**
- *(Opcional)* **Nix** para una toolchain totalmente reproducible:

  ```bash
  nix develop
  ```

### Inicio Rápido

```bash
# 1. Instalar dependencias de frontend
pnpm install

# 2. Levantar la pila backend local (PostgreSQL, Redis, NATS)
docker compose up -d

# 3. Ejecutar todas las apps TypeScript con hot reload
pnpm dev

# 4. Compilar el daemon de sync en Rust
cd astra-service/sync-daemon && cargo build --release
```

Copie `.env.example` a `.env` y ajuste los valores según sea necesario antes de ejecutar los servicios.

---

## Flujo de Trabajo de Desarrollo

```bash
# Lint, typecheck y test (el orden importa)
pnpm lint
pnpm typecheck
pnpm test

# Pruebas end-to-end (Playwright)
pnpm test:e2e

# Formato
pnpm format && pnpm format:check

# Compilar todos los paquetes
pnpm build
```

Ejecute un solo paquete mediante filtros de Turborepo:

```bash
pnpm turbo run dev --filter=@astra/kiosk
pnpm turbo run test --filter=@astra/kiosk
```

### Servicios Go

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

## Compilación y Ejecución

### Generación de Protobuf

```bash
cd proto
buf generate        # o: protoc según se documenta en proto/README.md
```

### Pila local completa

```bash
docker compose up -d
pnpm dev            # hot reload de kiosk-shell
```

Para manifiestos de producción, use `docker-compose.prod.yml`.

---

## Pruebas

| Capa          | Tooling                                                 |
| ------------- | ------------------------------------------------------- |
| Unit (TS)     | Vitest + happy-dom                                      |
| E2E (TS)      | Playwright contra `kiosk-shell`                         |
| Unit (Go)     | `go test -race ./...`                                    |
| Unit (Rust)   | `cargo test`, `cargo clippy`                            |
| Integración   | Pila Docker Compose (PostgreSQL, Redis, NATS)           |
| Caos          | Inyección de particiones de red durante la integración  |

> Las pruebas de integración y de caos requieren Docker con los contenedores `postgres`, `redis` y `nats` en ejecución.

---

## Documentación

- [`ARCHITECTURE.md`](./ARCHITECTURE.md) — diseño del sistema, modelo de seguridad, flujos de pago, observabilidad y DR.
- [`promt.md`](./promt.md) — la especificación de diseño de UI de quiosco biofílica "Living Weave".
- `proto/README.md`, `astra-service/sync-daemon/README.md` y `docs/` — subproyectos y runbooks operativos.

---

## Cómo Contribuir

1. Siga [Conventional Commits](https://www.conventionalcommits.org/) para todos los mensajes de commit.
2. Ejecute `pnpm prepare` para instalar los hooks de pre-commit de Lefthook.
3. Asegúrese de que `lint → typecheck → test` pasen antes de abrir un pull request.
4. Mantenga los cambios acotados a rutas; la CI es sensible a rutas y solo ejecuta las toolchains relevantes.

---

## Licencia

Licenciado bajo la [Licencia Apache, Versión 2.0](LICENSE).

---

<p align="center">
  <sub>Astra-System · Construido para un retail resiliente y sin conexión.</sub>
</p>
