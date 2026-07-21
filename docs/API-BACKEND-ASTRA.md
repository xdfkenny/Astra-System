# Backend API Full Inventory — Astra-System

*Document version: 1.1.0 • Last updated: 2026-07-20*

> **See also:**
> - [REST API Reference](./backend/rest-api.md) — structured endpoint documentation with request/response examples
> - [gRPC API Reference](./backend/grpc-api.md) — protobuf service definitions and message types
> - [API Gateway](./backend/api-gateway.md) — gateway middleware, rate limiting, auth
> - [Microservices Overview](./backend/microservices.md) — service map, dependencies, ports

---

## 1. System Overview

- **System name:** Astra-Service (Self-Checkout Kiosk Platform)
- **Stack:**
  - **API gateway:** Go Fiber v3 (`gateway` service)
  - **Microservices:** Go 1.25 (menu, cart, order, inventory, payment-orchestrator, sync, webauthn, admin-graphql)
  - **Language:** TypeScript (kiosk apps), Go (services), Rust (sync-daemon, Verifone FFI)
  - **Auth:** JWT Bearer (EdDSA primary, RS256 fallback) on production gateway; kiosk mesh Bearer token on sync; Admin JWT on GraphQL
  - **Storage:** PostgreSQL 16 (source of truth), Redis 7 (menu cache), NATS JetStream (events)
  - **Offline/P2P:** Rust sync-daemon (libp2p + QUIC), CRDT state replication
  - **Payments:** Verifone via Rust FFI sidecar; offline token queueing
  - **Observability:** OpenTelemetry, Prometheus, Grafana, Jaeger
- **Deployment:** Docker Compose (dev) / multi-container prod stack; single entrypoint **gateway** on port **8080**
- **Base URL (dev):** `http://localhost:8080`
- **API versioning:** Path prefix `/v1/...` (no major version in hostname)

### Meriandes → Astra product mapping (integration note)

If you previously called Meriandes `GET /api/products`, use Astra instead:

| Meriandes concept | Astra equivalent |
|-------------------|------------------|
| `GET /api/products` | `GET /v1/menu?store_id={uuid}` |
| Product | **Item** (`items` table / `Item` proto) |
| `productId` | `item_id` (UUID) |
| `unitPrice` / price | `price_cents` (integer cents) |
| Category | `categories[]` in menu response |
| P2P cached catalog | Redis cache in `menu-service` + client TanStack Query / Service Worker |
| No auth on products | **JWT Bearer required** on gateway |

---

## 2. API Endpoint Inventory

### 2.1 Gateway (primary entrypoint) — port 8080

| # | Module | Method | Path | Description | Auth | Risk |
|---|--------|--------|------|-------------|------|------|
| 1 | Health | GET | `/health` | Liveness | None | Low |
| 2 | Health | GET | `/live` | Alive check | None | Low |
| 3 | Health | GET | `/ready` | Readiness (deps) | None | Low |
| 4 | Metrics | GET | `/metrics` | Prometheus | None | Low |
| 5 | Docs | GET | `/docs/*` | Embedded Swagger | None | Low |
| 6 | **Menu** | **GET** | **`/v1/menu`** | **Full store catalog (products)** | **JWT** | **Med** |
| 7 | Menu | ALL | `/v1/menu/*` | Proxy → menu-service REST | JWT | Med |
| 8 | Cart | GET | `/v1/carts/:cartId` | Get cart (gRPC bridge) | JWT | Med |
| 9 | Cart | ALL | `/v1/cart/*` | Proxy → cart-service | JWT | High |
| 10 | Order | ALL | `/v1/order/*` | Proxy → order-service | JWT | High |
| 11 | Inventory | ALL | `/v1/inventory/*` | Proxy → inventory-service | JWT | Med |
| 12 | Payment | ALL | `/v1/payment/*` | Proxy → payment-orchestrator | JWT | High |
| 13 | Sync | ALL | `/v1/sync/*` | Proxy → sync-service | JWT (+ kiosk leader at service) | High |

### 2.2 Menu Service (direct / via proxy) — HTTP 8085, gRPC 50051

| # | Method | Path | Description | Auth | Risk |
|---|--------|------|-------------|------|------|
| 14 | GET | `/v1/menu/{store_id}` | Full menu for store | None at service* | Med |
| 15 | GET | `/v1/stores/{store_id}/menu` | Full menu (alt path) | None at service* | Med |
| 16 | GET | `/v1/categories/{store_id}` | Categories only | None at service* | Low |
| 17 | GET | `/v1/items/{item_id}` | Single item | None at service* | Low |
| 18 | GET | `/v1/items/search` | Search items | None at service* | Low |

\*Services trust network boundary; production clients must go through gateway JWT.

### 2.3 Order Service — HTTP 8084, gRPC 8083

| # | Method | Path | Description | Auth | Risk |
|---|--------|------|-------------|------|------|
| 19 | GET | `/healthz` | Health | None | Low |
| 20 | POST | `/v1/orders` | Create order | None* + `Idempotency-Key` | High |
| 21 | GET | `/v1/orders` | List orders | None* | Med |
| 22 | GET | `/v1/orders/{id}` | Get order | None* | Med |
| 23 | PATCH | `/v1/orders/{id}/status` | Update status | None* | High |
| 24 | POST | `/v1/orders/{id}/fulfill` | Fulfill order | None* | High |

### 2.4 Inventory Service — HTTP 8082, gRPC 9092

| # | Method | Path | Description | Auth | Risk |
|---|--------|------|-------------|------|------|
| 25 | GET | `/health`, `/live` | Health | None | Low |
| 26 | GET | `/stock` | Stock level | None* | Med |
| 27 | POST | `/reserve` | Reserve stock | None* | High |
| 28 | POST | `/release` | Release reservation | None* | High |
| 29 | POST | `/adjust` | Adjust stock | None* | High |

### 2.5 Payment Orchestrator — HTTP 8086, gRPC 50086

| # | Method | Path | Description | Auth | Risk |
|---|--------|------|-------------|------|------|
| 30 | GET | `/health`, `/live`, `/ready`, `/metrics` | Observability | None | Low |
| 31 | POST | `/v1/payments/` | Create payment | None* + `Idempotency-Key` | High |
| 32 | POST | `/v1/payments/:id/capture` | Capture | None* | High |
| 33 | POST | `/v1/payments/:id/settle` | Settle | None* | High |
| 34 | POST | `/v1/payments/:id/refund` | Refund | None* | High |
| 35 | POST | `/v1/payments/webhooks/verifone` | Verifone webhook | `X-Verifone-Signature` HMAC | High |
| 36 | POST | `/v1/offline-tokens/settle` | Batch offline settlement | None* | High |

### 2.6 Sync Service — HTTP 8087, gRPC 50051

| # | Method | Path | Description | Auth | Risk |
|---|--------|------|-------------|------|------|
| 37 | GET | `/health`, `/live`, `/ready` | Observability | None | Low |
| 38 | POST | `/v1/sync/upload` | Upload CRDT batch | Bearer = kiosk signing key hash + mesh leader | High |
| 39 | POST | `/v1/sync/download` | Download batch | Bearer + mesh leader | High |
| 40 | POST | `/v1/sync/heartbeat` | Kiosk heartbeat | Bearer + mesh leader | Med |

### 2.7 WebAuthn Service (prod) — HTTP 8091, gRPC 8090

| # | Method | Path | Description | Auth | Risk |
|---|--------|------|-------------|------|------|
| 41 | GET | `/healthz` | Health | None | Low |
| 42 | POST | `/v1/auth/webauthn/begin` | Begin employee override | None | Med |
| 43 | POST | `/v1/auth/webauthn/verify` | Verify assertion | None | High |
| 44 | POST | `/v1/auth/override/validate` | Validate override token | None | High |
| 45 | POST | `/v1/webauthn/register/begin` | Begin credential registration | None | High |
| 46 | POST | `/v1/webauthn/register/finish` | Finish registration | None | High |
| 47 | POST | `/v1/webauthn/authenticate/begin` | Begin authentication | None | Med |
| 48 | POST | `/v1/webauthn/authenticate/finish` | Finish authentication | None | High |

### 2.8 Admin GraphQL (prod) — HTTP 8092

| # | Method | Path | Description | Auth | Risk |
|---|--------|------|-------------|------|------|
| 49 | GET | `/healthz` | Health | None | Low |
| 50 | POST | `/graphql` | Admin GraphQL | Admin JWT (`is_admin` or `role=admin`) | High |

### 2.9 Update Server — HTTP 8090

| # | Method | Path | Description | Auth | Risk |
|---|--------|------|-------------|------|------|
| 51 | GET | `/manifest.json` | Signed OTA manifest | None | Low |
| 52 | POST | `/webhook/health` | Kiosk health report | None | Low |
| 53 | GET | `/healthz` | Health | None | Low |

### 2.10 Cart Service — gRPC only (50051)

| # | gRPC RPC | Description | Auth | Risk |
|---|----------|-------------|------|------|
| 54 | `CreateCart` | New cart | None* | Med |
| 55 | `GetCart` | Get cart (also exposed at gateway `GET /v1/carts/:cartId`) | None* | Med |
| 56 | `AddItem` | Add menu item | None* | High |
| 57 | `UpdateItem` | Update line quantity | None* | Med |
| 58 | `RemoveItem` | Remove line | None* | Med |
| 59 | `FinalizeCart` | Finalize for checkout | None* | High |
| 60 | `MergeGhostCart` | Merge mobile ghost cart | None* | High |

### 2.11 Legacy / stub api-gateway (not in docker-compose)

Alternate scaffold at `astra-service/services/api-gateway` with mock handlers and OpenAPI HMAC spec — **not the production gateway**.

| Method | Path | Notes |
|--------|------|-------|
| GET | `/v1/menu` | Hard-coded mock catalog |
| GET | `/v1/menu/stream` | SSE stub |
| POST | `/v1/carts/:cartId/items` | NATS publish stub |
| POST | `/v1/orders/` | Stub order creation |

---

## 3. Detailed API Documentation

### 3.1 Health Module

#### `GET /health`

```json
{
  "status": "ok",
  "service": "astra-gateway"
}
```

#### `GET /ready`

Returns `503` with `{"status":"not_ready","detail":"..."}` when PostgreSQL, Redis, or downstream services are unreachable.

**Test:**
```bash
curl http://localhost:8080/health
curl http://localhost:8080/ready
```

---

### 3.2 Menu / Products Module ⭐ (primary integration point)

#### `GET /v1/menu` — Fetch product catalog

- **Purpose:** Returns the full sellable catalog for a store — categories, items (products), modifier groups/options.
- **Auth:** `Authorization: Bearer <JWT>` (EdDSA or RS256)
- **Query parameters:**

| Param | Required | Description |
|-------|----------|-------------|
| `store_id` | **Yes** (UUID) | Store identifier |
| `include_inactive` | No | `"true"` to include inactive items |

- **Response** (`MenuResponse` from protobuf):

```json
{
  "store_id": "550e8400-e29b-41d4-a716-446655440000",
  "categories": [
    {
      "category_id": "...",
      "store_id": "...",
      "parent_id": "",
      "name": "Beverages",
      "description": "",
      "display_order": 1,
      "image_url": "https://...",
      "blurhash": "...",
      "is_active": true
    }
  ],
  "items": [
    {
      "item_id": "...",
      "store_id": "...",
      "category_id": "...",
      "name": "Coke",
      "description": "330ml can",
      "price_cents": 250,
      "cost_cents": 120,
      "plu": "4011",
      "barcode": "7891234567890",
      "sku": "BEV-COKE-330",
      "image_url": "https://...",
      "blurhash": "...",
      "tax_category": "ITEM_TAX_CATEGORY_STANDARD",
      "is_weight_based": false,
      "weight_unit": "WEIGHT_UNIT_UNSPECIFIED",
      "is_active": true,
      "modifier_groups": [
        {
          "modifier_group_id": "...",
          "name": "Size",
          "min_select": 1,
          "max_select": 1,
          "options": [
            {
              "modifier_option_id": "...",
              "name": "Large",
              "price_delta_cents": 50,
              "is_default": false
            }
          ]
        }
      ],
      "metadata": {}
    }
  ]
}
```

- **Data flow:**

```
PostgreSQL (items, categories, modifier_*)
  → menu-service repository
  → Redis cache (key: menu:menu:{storeId}, TTL 5m default)
  → gRPC MenuService.GetMenu
  → gateway handleGetMenu
  → Client
```

- **Alternative paths** (via gateway proxy to menu-service REST):

| Method | Path | Equivalent |
|--------|------|------------|
| GET | `/v1/menu/{store_id}` | Same as above with path param |
| GET | `/v1/stores/{store_id}/menu` | Same |
| GET | `/v1/categories/{store_id}` | Categories only |
| GET | `/v1/items/{item_id}` | Single product |
| GET | `/v1/items/search?store_id=&query=&category_id=` | Search |

- **Errors:**
  - `401` — missing/invalid JWT
  - `502` — `{"error":"menu_service_unavailable"}` (gRPC downstream failure)
  - `429` — Redis rate limit exceeded (gateway)

- **Client-side caching** (kiosk apps):
  - TanStack Query key `["menu-catalog"]`, `offlineFirst` network mode
  - Service Worker: Workbox `StaleWhileRevalidate` on `/v1/menu` (cache name `astra-menu-data`)
  - Default stale time: 300 seconds in kiosk shell

- **Test:**
```bash
# Replace TOKEN and STORE_ID
curl -H "Authorization: Bearer $TOKEN" \
  "http://localhost:8080/v1/menu?store_id=$STORE_ID"

# Search by name/barcode/plu
curl -H "Authorization: Bearer $TOKEN" \
  "http://localhost:8080/v1/menu/items/search?store_id=$STORE_ID&query=coke"
```

- **Field mapping for external clients:**

| Your field | Astra field | Notes |
|------------|-------------|-------|
| `id` / `productId` | `item_id` | UUID string |
| `name` | `name` | Plain string (not i18n object) |
| `price` | `price_cents / 100` | Always integer cents server-side |
| `category` | `category_id` + lookup in `categories[]` | |
| `sku` | `sku` | |
| `barcode` | `barcode` | Used by produce scanner PLU lookup |
| `plu` | `plu` | Produce/scale lookup |
| `available` | `is_active` | No live inventory join on read path today |
| `image` | `image_url` | |
| `modifiers` | `modifier_groups[].options[]` | `price_delta_cents` per option |

- **Events** (on catalog writes — not yet consumed for cache invalidation):
  - `astra.menu.updated.v1`
  - `astra.menu.item.price_changed.v1`

---

### 3.3 Cart Module

Cart operations are primarily **gRPC**. Gateway exposes:

#### `GET /v1/carts/:cartId`

Returns full `Cart` with lines, totals, version, expiry.

#### gRPC `AddItem`

Snapshots price at add time:
- `menu_item_id` — references catalog `item_id`
- `name_snapshot`, `unit_price_cents_snapshot` — frozen at add
- `modifiers[]` with `price_delta_cents_snapshot`

**Idempotency:** Client should pass stable keys for checkout flows; cart uses optimistic versioning (`version` field).

---

### 3.4 Order Module

#### `POST /v1/orders`

- **Header:** `Idempotency-Key` (required)
- **Body:** `CreateOrderRequest` — `cart_id`, `store_id`, `kiosk_id`, optional `customer_phone`
- **Response:** `Order` with `order_number`, status, line snapshots
- **Statuses:** `pending` → `paid` → `fulfilled` | `cancelled` | `refunded`

#### `GET /v1/orders/{id}`

Returns persisted order with price snapshots (server-authoritative, not client cart totals).

---

### 3.5 Payment Module

#### `POST /v1/payments/`

- **Header:** `Idempotency-Key`
- **Body:** `{ order_id, kiosk_id, amount_cents, currency, method, is_offline_token }`
- **Methods:** credit/debit, NFC (Apple/Google Pay), QR, cash recycler
- **Security:** `assertNoSensitiveCardData` — rejects PAN/CVV/EMV in request bodies; card data stays on Verifone terminal
- **Offline:** Tokens queued locally, settled via `POST /v1/offline-tokens/settle`

#### Verifone webhook

- **Path:** `POST /v1/payments/webhooks/verifone`
- **Auth:** `X-Verifone-Signature` HMAC (`PAYMENT_WEBHOOK_SECRET`)

---

### 3.6 Inventory Module

Separate from catalog — tracks stock per `(store_id, item_id)`.

#### `GET /stock?store_id=&item_id=`

Returns `StockLevel`. Menu read path does **not** currently derive `isAvailable` from inventory.

#### `POST /reserve` / `POST /release`

Used during cart finalize/checkout to prevent overselling.

---

### 3.7 Sync Module (offline mesh)

#### Auth model

1. Kiosk must exist in DB
2. Kiosk must be **mesh leader** (`is_leader = true`)
3. `Authorization: Bearer <signing_key_hash>` must match kiosk record

#### `POST /v1/sync/upload`

Upload CRDT delta batch (inventory updates, cart merges, transaction batches).

#### `POST /v1/sync/download`

Download deltas since cursor for peer reconciliation.

#### `POST /v1/sync/heartbeat`

Reports kiosk status; returns leader assignment and config version.

---

### 3.8 Admin GraphQL Module

**Endpoint:** `POST http://localhost:8092/graphql`

#### Catalog query (admin read)

```graphql
query {
  menus(storeId: "550e8400-e29b-41d4-a716-446655440000", includeInactive: false) {
    storeId
    categories { categoryId name displayOrder isActive }
    items { itemId name priceCents sku barcode plu isActive categoryId }
  }
}
```

**Auth:** JWT with `is_admin: true` or `role: "admin"`.

**Note:** No write mutations for catalog yet (`noop` placeholder only). Writes exist in `menu-service` repository layer but are not exposed via public API.

---

### 3.9 WebAuthn / Employee Override

Used for attended-mode employee interventions (voids, overrides). Ceremony endpoints at `/v1/webauthn/*` and `/v1/auth/webauthn/*`.

---

### 3.10 Update Server (OTA)

#### `GET /manifest.json`

Signed release manifest for kiosk auto-updater.

#### `POST /webhook/health`

Kiosk reports `{ kioskId, version, healthy, error? }` → `202 accepted`.

---

## 4. Authentication Overview

### 4.1 Auth methods comparison

| Method | Use case | Algorithm | Credential | State |
|--------|----------|-----------|------------|-------|
| **JWT Bearer** | All gateway `/v1/*` routes | EdDSA (primary), RS256 (fallback) | `Authorization: Bearer {token}` | Stateless |
| **Kiosk Bearer** | Sync service | Constant-time compare to `signing_key_hash` | Bearer token in header | DB-backed kiosk record |
| **Admin JWT** | Admin GraphQL | HS256 | Bearer + `is_admin` claim | Stateless |
| **Verifone HMAC** | Payment webhooks | HMAC-SHA256 | `X-Verifone-Signature` | Shared secret |
| **HMAC kiosk signing** | Documented in `api-gateway` OpenAPI only | HMAC-SHA256 | `X-Astra-Kiosk-Id`, `X-Astra-Timestamp`, `X-Astra-Signature` | **Not implemented in production gateway** |

### 4.2 JWT validation (production gateway)

| Claim / check | Value |
|---------------|-------|
| Issuer | `GATEWAY_JWT_ISSUER` (default: `astra-service`) |
| Audience | `GATEWAY_JWT_AUDIENCE` (default: `astra-gateway`) |
| Algorithms | `EdDSA`, `RS256` |
| Public keys | `GATEWAY_JWT_EDDSA_PUBLIC_KEY` or `_PATH`; `GATEWAY_JWT_RSA_PUBLIC_KEY` or `_PATH` |
| Public paths | `/health`, `/live`, `/ready`, `/metrics`, `/docs/*` |

### 4.3 Request flow (product fetch)

```
┌──────────┐         ┌─────────────┐         ┌──────────────┐         ┌──────────┐
│  Client  │         │   Gateway   │         │ menu-service │         │ Postgres │
│ (Kiosk)  │         │   :8080     │         │  gRPC+Redis  │         │  + Redis │
└────┬─────┘         └──────┬──────┘         └──────┬───────┘         └────┬─────┘
     │                      │                       │                      │
     │ GET /v1/menu         │                       │                      │
     │ Authorization: Bearer│                       │                      │
     │ ?store_id=UUID       │                       │                      │
     │─────────────────────→│                       │                      │
     │                      │ ① JWT middleware      │                      │
     │                      │ ② Rate limit (Redis)  │                      │
     │                      │ ③ gRPC GetMenu        │                      │
     │                      │──────────────────────→│                      │
     │                      │                       │ ④ Redis cache hit?   │
     │                      │                       │─────────────────────→│
     │                      │                       │ ⑤ else SQL query     │
     │                      │                       │─────────────────────→│
     │                      │←──────────────────────│                      │
     │←─────────────────────│  MenuResponse JSON    │                      │
```

---

## 5. Data Persistence

| Store | Content | Access |
|-------|---------|--------|
| PostgreSQL `items` | Product catalog (price, sku, plu, barcode) | menu-service, admin-graphql |
| PostgreSQL `categories` | Menu categories | menu-service |
| PostgreSQL `modifier_groups` / `modifier_options` | Add-on options | menu-service |
| PostgreSQL `inventory` | Stock levels per store+item | inventory-service |
| PostgreSQL `orders` | Order records | order-service |
| PostgreSQL `carts` | Active carts | cart-service |
| PostgreSQL `kiosks` | Kiosk identity, leader flag, signing key | sync-service |
| Redis | Menu cache `menu:menu:{storeId}` | menu-service |
| NATS JetStream | Domain events, outbox | All services |

**Schema:** `database/migrations/0001_init.sql`  
**Proto contracts:** `proto/proto/*.proto`  
**TS types:** `astra-service/packages/shared-types/src/`

---

## 6. Offline / P2P Architecture

Unlike Meriandes (Socket.IO + mDNS + JSON file cache), Astra uses:

- **Rust sync-daemon** — libp2p mesh, QUIC transport, Noise encryption
- **CRDT types** — PN-Counters, LWW-Registers, OR-Sets with Hybrid Logical Clocks
- **Cloud sync** — Leader kiosk uploads/downloads via `sync-service`
- **Catalog offline** — Client caches last good `GET /v1/menu` response (not P2P-shared catalog files)

**Important:** Product catalog is **not** replicated P2P between kiosks. Each kiosk fetches from gateway (or serves from local HTTP cache). P2P sync covers inventory deltas, cart merges, and transaction batches.

---

## 7. Idempotency & Concurrency

| Operation | Mechanism |
|-----------|-----------|
| Order create | `Idempotency-Key` header (HTTP + gRPC metadata) |
| Payment create | `Idempotency-Key` header |
| Cart mutations | Optimistic `version` field |
| Stock reserve | Serialized per `(store_id, item_id)` in inventory-service |
| POS terminal | `withTerminalLock` in payment-orchestrator |

---

## 8. Environment Variables Reference

| Variable | Default | Description |
|----------|---------|-------------|
| `GATEWAY_PORT` | `8080` | Gateway listen port |
| `GATEWAY_JWT_ISSUER` | `astra-service` | JWT issuer |
| `GATEWAY_JWT_AUDIENCE` | `astra-gateway` | JWT audience |
| `GATEWAY_JWT_EDDSA_PUBLIC_KEY` | — | Ed25519 public key (PEM or base64) |
| `GATEWAY_JWT_RSA_PUBLIC_KEY` | — | RSA public key (fallback) |
| `GATEWAY_RATE_LIMIT_RPS` | `50` | Rate limit per IP |
| `GATEWAY_ALLOWED_ORIGINS` | `http://localhost:5170` | CORS whitelist |
| `DATABASE_URL` | — | PostgreSQL connection string |
| `REDIS_URL` | — | Redis connection string |
| `NATS_URL` | — | NATS JetStream URL |
| `MENU_SERVICE_URL` | `http://menu-service:8085` | Gateway → menu HTTP proxy |
| `MENU_SERVICE_GRPC_ADDR` | `menu-service:50051` | Gateway → menu gRPC |
| `CACHE_TTL` | `5m` | menu-service Redis TTL |
| `VITE_API_GATEWAY_URL` | `http://localhost:8080` | Kiosk frontend API base |

Full list: `.env.example` at repo root.

---

## 9. Integration Checklist (fetching products from Astra)

1. **Start stack:** `docker compose up -d` (gateway + menu-service + postgres + redis)
2. **Obtain JWT** — issue a token with correct `iss`, `aud`, signed with EdDSA or RS256 private key matching gateway public key
3. **Know your `store_id`** — UUID from `stores` table (seed data in migrations)
4. **Call:** `GET http://localhost:8080/v1/menu?store_id={uuid}`
5. **Parse:** Use `items[]` as products; join `category_id` → `categories[]`
6. **Price:** Always use `price_cents`; do not trust client-computed totals for checkout
7. **Modifiers:** Apply `price_delta_cents` from selected `modifier_groups[].options[]`
8. **PLU/barcode lookup:** Client-side scan against loaded catalog (same pattern as kiosk produce scanner)
9. **Cache:** Respect 5m server TTL; use `If-None-Match` only if you add it — not implemented today
10. **Offline fallback:** Cache last successful response locally (Meriandes P2P cache has no direct equivalent)

---

## 10. Known Gaps & Risks

| Issue | Impact | Recommendation |
|-------|--------|----------------|
| Kiosk clients omit `store_id` on `/v1/menu` | Empty catalog / gRPC validation error | Always pass `store_id` query param |
| No public catalog write API | Admin must use DB/seeds directly | Add admin mutations or REST CRUD |
| `is_active` ≠ in-stock | UI may show unavailable items as orderable | Join inventory in menu-service or client |
| Two gateways (`gateway` vs `api-gateway`) | Confusion about auth model | Use `gateway:8080` only in production |
| OpenAPI HMAC spec ≠ production JWT | Integration docs mismatch | Treat this document + `gateway` code as source of truth |
| admin-graphql UI schema drift | Admin panel may break | Align `kiosk-admin` queries with `menus` resolver |

---

## 11. Service Port Map (docker-compose dev)

| Service | Host port | Internal role |
|---------|-----------|---------------|
| **gateway** | **8080** | **Public API entry** |
| postgres | 5432 | Database |
| redis | 6379 | Cache / rate limit |
| nats | 4222 | Events |
| menu-service | — | Catalog gRPC/REST |
| cart-service | — | Cart gRPC |
| order-service | — | Orders |
| inventory-service | — | Stock |
| payment-orchestrator | — | Payments |
| sync-service | — | Cloud sync |
| update-server | 8090 | OTA |
| kiosk (Vite) | 5180 | Frontend dev |
| ml-lane-intel | 8088 | ML profile only |
| syncd | 4499 | Rust P2P profile only |

---

*Generated from Astra-System codebase audit — gateway, menu-service, proto definitions, shared-types, and docker-compose.*
