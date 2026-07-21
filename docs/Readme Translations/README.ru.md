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
  <a href="./README.ru.md"><b>Русский</b></a> ·
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

> Платформа автоматизированного самообслуживания уровня производства, работающая в офлайн-режиме и предназначенная для круглосуточной розничной торговли.

**Astra-System** — это многоязычный монорепозиторий, обеспечивающий работу неосмотренных и осмотренных касс самообслуживания. Он обеспечивает бесперебойную работу магазина с **48 часами офлайн-устойчивости**, моделью безопасности zero-trust и уровнем синхронизации в пиринговой mesh-сети, который поддерживает согласованность каждой кассы в магазине — даже когда облако недоступно.

---

## Содержание

- [Обзор](#обзор)
- [Основные возможности](#основные-возможности)
- [Архитектура](#архитектура)
- [Технологический стек](#технологический-стек)
- [Структура репозитория](#структура-репозитория)
- [Начало работы](#начало-работы)
- [Рабочий процесс разработки](#рабочий-процесс-разработки)
- [Сборка и запуск](#сборка-и-запуск)
- [Тестирование](#тестирование)
- [Документация](#документация)
- [Участие в проекте](#участие-в-проекте)
- [Лицензия](#лицензия)

---

## Обзор

Astra-System позволяет ритейлерам разворачивать флоты касс самообслуживания, которые работают **автономно до 48 часов** без подключения к интернету. Устойчивость распределена по трём уровням:

1. **Локальный уровень данных** — зашифрованное хранилище SQLite (SQLCipher) на каждой кассе с полным каталогом меню, инвентарём, ожидающими транзакциями и офлайн-токенами оплаты.
2. **Пиринговая mesh-сеть** — кассы обнаруживают друг друга в локальной сети (mDNS + libp2p/QUIC) и реплицируют состояние с помощью CRDT, избирая Raft-лидера при наличии трёх и более касс.
3. **Graceful degradation** — оплата, инвентарь и приём заказов продолжают работу локально и сверяются с облаком при восстановлении связности.

Облачный уровень (Go-микросервисы, PostgreSQL 16, Redis 7, NATS JetStream) обеспечивает авторитетное событийно-ориентированное хранилище, расчёты и управление флотом.

### Цели проектирования

| Цель | Значение |
| --- | --- |
| Устойчивость в офлайне | 48 часов автономной работы без подключения к облаку |
| Латентность | < 200 мс загрузка меню, < 500 мс P2P-синхронизация инвентаря, < 3 с переключение лидера |
| Доступность | 99,99% аптайма (облачный уровень); 100% аптайма в режиме «только локально» |
| Безопасность | Zero trust, mTLS везде, PCI-DSS-совместимый путь оплаты |
| Масштаб | 1–10 000 касс на арендатора; мультирегиональное облачное развёртывание |

---

## Основные возможности

- **Offline-first движок** — детерминированное слияние CRDT (PN-Counter, LWW-Register, OR-Set) с гибридными логическими часами для причинно-следственного порядка между кассами.
- **P2P mesh и консенсус Raft** — транспорт QUIC от libp2p, шифрование Noise Protocol и переключение лидера менее чем за 3 секунды.
- **Транзакционный outbox** — публикация событий ровно один раз из облачных сервисов через NATS JetStream.
- **Безопасность zero-trust** — mTLS, HMAC-подпись для каждой кассы, идентичности SPIFFE и PCI-DSS-совместимый путь оплаты (данные карты никогда не попадают в память кассы).
- **Мост Verifone FFI** — безопасная обёртка Rust над C-SDK производителя для интеграции платёжных терминалов.
- **Биофильный UI кассы** — микрофронтенд React 19, построенный с Module Federation, конечным автоматом XState v5 и управлением состоянием Zustand/TanStack Query.
- **Продвинутая аналитика** — Ghost Carts, распознавание продукции (ONNX), аналитика полосы (TFLite), WebAuthn/passkeys и аналитика с дифференциальной приватностью.
- **CI, устойчивый к хаосу** — сетевые партиции вводятся во время интеграционных тестов для проверки устойчивости, сходимости CRDT и постановки платежей в очередь.
- **Многоязычный UI кассы** — клиенты выбирают предпочтительный язык в начале сеанса из 17+ поддерживаемых языков. Все тексты интерфейса, чеки и аудиоинструкции отображаются на выбранном языке.

---

## Архитектура

Astra-System разделён на **облачный уровень** и **кластер граничных касс магазина**.

Для полной топологии, модели безопасности, потоков оплаты, наблюдаемости и деталей восстановления после сбоев см. [`ARCHITECTURE.md`](../../ARCHITECTURE.md).

### Реестр сервисов

| Сервис | Язык | Ответственность |
| --- | --- | --- |
| `api-gateway` | Go | Граничную маршрутизация, authN/authZ, ограничение скорости |
| `order-svc` | Go | Жизненный цикл заказов, сохранение корзины, исполнение |
| `payment-svc` | Go | Оркестрация платежей, расчёты токенов |
| `inventory-svc` | Go | Уровни запасов, мягкие удержания, синхронизация каталога |
| `cart-svc` | Go | Слияние CRDT корзины, разрешение Ghost Carts |
| `sync-svc` | Go | Облачный mesh-шлюз и пакетная загрузка |
| `astra-syncd` | Rust | P2P-демон кассы, CRDT-синхронизация, мост Verifone FFI |
| `kiosk-shell` | TypeScript | Клиентский UI React 19, интеграция периферии |
| `update-server` | Go | Доставка подписанных OTA-манифестов |

---

## Технологический стек

- **Frontend** — TypeScript, React 19, Vite, Module Federation, XState v5, Zustand, TanStack Query, Tailwind CSS.
- **Backend** — Go (Fiber / gRPC), PostgreSQL 16, Redis 7, NATS JetStream.
- **Граница** — Rust (`astra-syncd`, `astra-verifone-ffi`), SQLite (SQLCipher), libp2p.
- **ML** — ONNX Runtime, TensorFlow Lite.
- **Инфра** — Kubernetes, Docker / Podman, Traefik, HashiCorp Vault, Nix flake.
- **Наблюдаемость** — Prometheus, Grafana, Loki, Jaeger, OpenTelemetry.

---

## Структура репозитория

```text
astra-service/          Код сервисов и приложений
  apps/                 Микрофронтенды TypeScript (kiosk-shell, kiosk-menu, …)
  packages/             Общие библиотеки и дизайн-система
  services/             Микросервисы Go
  sync-daemon/          astra-syncd (Rust) P2P-демон
  daemons/              Sidecar-демоны (payment-sidecar)
  tools/                Операционные инструменты (chaos и т.д.)
services/               Автономные сервисы (update-server, …)
database/               Миграции схемы
proto/                  Определения Protocol Buffer и сгенерированный код
docs/                   Операционные руководства
infra/                  Инструменты инфраструктуры и вспомогательные средства секретов
.github/                CI-воркфлоу и файлы сообщества
flake.nix               Воспроизводимая среда разработки Nix
docker-compose*.yml     Манифесты compose для локальной и.production среды
```

---

## Начало работы

### Предварительные требования

- **Node.js 22** и **pnpm 9+**
- **Go 1.25**
- **Rust 1.82** (с `protoc` для сборки демона синхронизации)
- **Docker** и **Docker Compose**
- *(Опционально)* **Nix** для полностью воспроизводимого набора инструментов:

  ```bash
  nix develop
  ```

### Быстрый старт

```bash
# 1. Установить зависимости фронтенда
pnpm install

# 2. Запустить локальный бэкенд-стек (PostgreSQL, Redis, NATS)
docker compose up -d

# 3. Запустить все TypeScript-приложения с горячей перезагрузкой
pnpm dev

# 4. Собрать Rust-демон синхронизации
cd astra-service/sync-daemon && cargo build --release
```

Скопируйте `.env.example` в `.env` и при необходимости измените значения перед запуском сервисов.

---

### Установщик

Готовые тестовые бинарные файлы для macOS, Linux и Windows доступны на [странице Releases](https://github.com/xdfkenny/Astra-System/releases).

| Платформа | Бинарный файл |
| --- | --- |
| macOS (Intel) | `astra-installer-darwin-amd64` |
| macOS (Apple Silicon) | `astra-installer-darwin-arm64` |
| Linux (x86_64) | `astra-installer-linux-amd64` |
| Linux (ARM64) | `astra-installer-linux-arm64` |
| Windows (x86_64) | `astra-installer-windows-amd64.exe` |

```bash
# macOS / Linux — загрузить и запустить скрипт загрузки
curl -sL https://raw.githubusercontent.com/xdfkenny/Astra-System/main/installer/scripts/install.sh | bash

# Или загрузить бинарный файл напрямую из Releases, сделать его исполняемым и запустить:
./astra-installer-<platform>
```

---

## Рабочий процесс разработки

```bash
# Линт, проверка типов и тесты (порядок важен)
pnpm lint
pnpm typecheck
pnpm test

# End-to-end тесты (Playwright)
pnpm test:e2e

# Форматирование
pnpm format && pnpm format:check

# Собрать все пакеты
pnpm build
```

Запуск отдельного пакета через фильтры Turborepo:

```bash
pnpm turbo run dev --filter=@astra/kiosk
pnpm turbo run test --filter=@astra/kiosk
```

### Сервисы Go

```bash
cd astra-service/services
go test -race ./...
go vet ./...
```

### Демоны Rust

```bash
cd astra-service/sync-daemon
cargo test
cargo clippy -- -D warnings
cargo fmt --check
```

---

## Сборка и запуск

### Генерация Protobuf

```bash
cd proto
buf generate
```

### Локальный полный стек

```bash
docker compose up -d
pnpm dev
```

Для producción-манифестов используйте `docker-compose.prod.yml`.

---

## Тестирование

| Уровень | Инструменты |
| --- | --- |
| Юнит (TS) | Vitest + happy-dom |
| E2E (TS) | Playwright для `kiosk-shell` |
| Юнит (Go) | `go test -race ./...` |
| Юнит (Rust) | `cargo test`, `cargo clippy` |
| Интеграция | Стек Docker Compose (PostgreSQL, Redis, NATS) |
| Хаос | Инъекция сетевых партиций во время интеграции |

> Интеграционные и хаос-тесты требуют запущенного Docker с контейнерами `postgres`, `redis` и `nats`.

---

## Документация

Полная документация доступна в [`docs/`](../../docs/):

| Раздел | Содержание |
| --- | --- |
| **Архитектура** | [Обзор](../../docs/architecture/overview.md), [Проектирование системы](../../docs/architecture/system-design.md), [Стратегия Offline-First](../../docs/architecture/offline-first.md), [Модель безопасности](../../docs/architecture/security-model.md) |
| **Бэкенд** | [Микросервисы](../../docs/backend/microservices.md), [API Gateway](../../docs/backend/api-gateway.md), [REST API](../../docs/backend/rest-api.md), [gRPC API](../../docs/backend/grpc-api.md), [Оркестратор платежей](../../docs/backend/payment-orchestrator.md) |
| **Фронтенд** | [Микрофронтенды](../../docs/frontend/micro-frontends.md), [Приложения касс](../../docs/frontend/kiosk-apps.md), [Управление состоянием](../../docs/frontend/state-management.md) |
| **База данных** | [Схема](../../docs/database/schema.md), [Миграции](../../docs/database/migrations.md), [Сущности](../../docs/database/entities.md) |
| **Инфраструктура** | [Docker](../../docs/infrastructure/docker.md), [Kubernetes](../../docs/infrastructure/kubernetes.md), [Наблюдаемость](../../docs/infrastructure/monitoring.md), [CI/CD](../../docs/infrastructure/ci-cd.md) |
| **Сеть** | [P2P mesh](../../docs/networking/p2p-mesh.md), [Протоколы](../../docs/networking/protocols.md) |
| **Безопасность** | [Обзор](../../docs/security/overview.md), [Аутентификация](../../docs/security/authentication.md), [Шифрование](../../docs/security/encryption.md) |

Основные справочники:
- [`ARCHITECTURE.md`](../../ARCHITECTURE.md) — проектирование системы, модель безопасности, потоки оплаты, наблюдаемость и DR.
- [`UX_UI_AUDIT_REPORT.md`](../../astra-service/UX_UI_AUDIT_REPORT.md) — спецификация дизайна биофильного UI кассы «Living Weave».
- [`docs/API-BACKEND-ASTRA.md`](../../docs/API-BACKEND-ASTRA.md) — полный реестр эндпоинтов API.
- [`docs/Readme Translations/`](../../docs/Readme%20Translations/) — перевода README, внесённые сообществом, на 17+ языках.
- [`docs/runbooks/`](../../docs/runbooks/) — операционные руководства (реагирование на инциденты, офлайн-режим, восстановление P2P, сбой оплаты).

---

## Участие в проекте

1. Соблюдайте [Conventional Commits](https://www.conventionalcommits.org/) для всех сообщений коммитов.
2. Запустите `pnpm prepare` для установки pre-commit-хуков Lefthook.
3. Убедитесь, что `lint → typecheck → test` проходят успешно перед созданием pull request.
4. Ограничивайте изменения областью путей; CI фильтрует по путям и запускает только соответствующие цепочки инструментов.

---

## Лицензия

Лицензировано по [Apache License, Version 2.0](../../LICENSE).

---

<p align="center">
  <sub>Astra-System · Создано для устойчивого офлайн-первого ритейла.</sub>
</p>
