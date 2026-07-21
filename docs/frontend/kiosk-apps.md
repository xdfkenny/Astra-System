# Kiosk Applications

## Kiosk Host Shell

**Location:** `apps/kiosk/`

The main kiosk application that serves as the Module Federation host. It orchestrates the workflow, manages global state, and composes micro-frontends.

### Entry Point

**File:** `apps/kiosk/src/main.tsx`

```typescript
createRoot(document.getElementById('root')!).render(
  <KioskProvider>
    <KioskShell />
  </KioskProvider>
)
```

### Key Files

| File | Purpose |
|------|---------|
| `src/main.tsx` | Application entry, React root |
| `src/App.tsx` | KioskShell component composition |
| `src/machines/kioskMachine.ts` | XState v5 workflow machine (9 stages) |
| `src/routes/WorkflowRouter.tsx` | Stage-to-screen router with federated lazy-loads |
| `src/state/queryClient.ts` | TanStack Query configuration |
| `src/state/apiClient.ts` | HTTP API client with caching, dedup, retry |
| `src/state/cartService.ts` | Cart service with debounced sync (800ms) |
| `src/workers/crdtWorker.ts` | CRDT Web Worker (WASM) |
| `src/workers/service-worker.ts` | Workbox service worker |
| `src/hooks/useNetworkMonitor.ts` | Network monitoring (5s poll) |
| `src/hooks/useIdleReclaim.ts` | Idle reclaim (90s timeout) |
| `src/hooks/useSilentAssist.ts` | Silent assist (45s stall detection) |
| `src/updater/updater.ts` | OTA updater with Ed25519 verification |
| `src/updater/remote-modules.ts` | Dynamic MFE remote loader |
| `src/ghost-cart/qrSignaling.ts` | QR-based ghost cart WebRTC signaling |
| `src/ghost-cart/dataChannel.ts` | WebRTC data channel for cart transfer |
| `src/ghost-cart/nfcFallback.ts` | NFC fallback for ghost cart |
| `src/webauthn/employeeAuth.ts` | Employee WebAuthn/FIDO2 auth |
| `src/produce/useProduceScanner.ts` | Camera-based produce recognition |
| `src/i18n/index.ts` | i18n with 17 locales, RTL support |
| `src/utils/logger.ts` | Structured JSON logging |

### Configuration

**File:** `vite.config.ts`

- Module Federation (host configuration)
- PWA support (Workbox)
- Chunk splitting for optimal loading
- Development server on port 5180

## Kiosk Menu (Remote)

**Location:** `apps/kiosk-menu/`

Virtualized item grid with search, filtering, and category browsing.

### Key Components

- **MenuApp.tsx** - Main menu micro-frontend
- Virtualized grid via TanStack Virtual
- Category filtering with animated transitions
- Item search with debounced input
- Item detail view with modifier selection

## Kiosk Cart (Remote)

**Location:** `apps/kiosk-cart/`

Shopping cart view with line item management, totals, and modifiers.

### Key Components

- **CartApp.tsx** - Main cart micro-frontend
- Reads cart state from shared Valtio proxy
- Displays line items, quantities, totals
- Modifier selection per line item
- Quantity increment/decrement controls

## Kiosk Payment (Remote)

**Location:** `apps/kiosk-payment/`

Payment flow UI with 4 payment methods and Verifone terminal integration.

### Key Components

- **PaymentApp.tsx** - Main payment micro-frontend
- 4 payment methods: credit/debit, cash, mobile wallet, gift card
- 4 payment phases: method selection → processing → result → receipt
- **verifoneBridge.ts** - Local sidecar communication (`127.0.0.1:8963`)
- Offline token generation when network unavailable

## Kiosk Admin (Standalone)

**Location:** `apps/kiosk-admin/`

Store management dashboard with GraphQL API and RBAC.

### Key Features

- **RouteGuard** RBAC for all routes (5 roles × resource CRUD matrix)
- 10 route pages: Dashboard, Locations, Lanes, Kiosks, Menu, Inventory, Orders, Payments/Refunds, Employees/Roles, Audit Logs
- Fleet health monitoring (3s polling)
- Mesh topology visualization
- Circuit breaker status dashboard

### Key Files

| File | Purpose |
|------|---------|
| `src/App.tsx` | Router with 10 RBAC-guarded routes |
| `src/main.tsx` | Application entry |
| `src/graphql/queries.ts` | 8 GraphQL queries |
| `src/lib/roles.ts` | RBAC permissions definition |
| `src/hooks/useFleetHealth.ts` | Fleet health polling |
| `src/components/RouteGuard.tsx` | Route authorization guard |
| `src/components/MeshTopologyGraph.tsx` | P2P mesh visualization |
| `src/components/CircuitBreakerList.tsx` | Circuit breaker status |
| `src/routes/*.tsx` | Individual route pages |
