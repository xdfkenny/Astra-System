# State Management

## Overview

Astra-System uses a **multi-store state architecture** with four state management libraries, each serving a specific purpose. This avoids the limitations of a single global store while maintaining consistency.

## Store Architecture

```
┌──────────────────────────────────────────────────────────────┐
│                       Kiosk Application                       │
│                                                               │
│  ┌────────────────────────────────────────────────────────┐  │
│  │                    XState v5 Machine                     │  │
│  │  LANGUAGE_SELECT → ATTRACT → MENU → CART → PAYMENT →   │  │
│  │              PROCESSING → RECEIPT → ADMIN               │  │
│  │            (Workflow orchestration)                     │  │
│  └────────────────────────┬───────────────────────────────┘  │
│                           │                                    │
│  ┌────────────────────────┴───────────────────────────────┐  │
│  │                    Zustand Store                        │  │
│  │  Session state │ Network status │ Lane mode │ i18n     │  │
│  │            (Ephemeral, non-persisted)                  │  │
│  └────────────────────────┬───────────────────────────────┘  │
│                           │                                    │
│  ┌────────────────────────┴───────────────────────────────┐  │
│  │                    Valtio Proxy                         │  │
│  │     Cart state (reactive, CRDT-backed)                 │  │
│  │     → subscribe() → CRDT Worker → IndexedDB           │  │
│  └────────────────────────┬───────────────────────────────┘  │
│                           │                                    │
│  ┌────────────────────────┴───────────────────────────────┐  │
│  │               TanStack Query (React Query)               │  │
│  │       Server cache │ Offline-first │ Background sync    │  │
│  │       Menu data │ Inventory │ Orders │ Payments         │  │
│  └────────────────────────────────────────────────────────┘  │
└──────────────────────────────────────────────────────────────┘
```

## 1. XState v5 - Workflow State Machine

**File:** `apps/kiosk/src/machines/kioskMachine.ts`

The kiosk workflow is modeled as a single state machine with 9 stages:

```typescript
const kioskMachine = setup({
  types: {
    context: {} as KioskContext,
    events: {} as KioskEvents,
  },
}).createMachine({
  id: 'kiosk',
  initial: 'LANGUAGE_SELECT',
  states: {
    LANGUAGE_SELECT: { on: { LANGUAGE_SELECTED: 'ATTRACT' } },
    ATTRACT: { on: { START_ORDER: 'MENU', ADMIN_LOGIN: 'ADMIN' } },
    MENU: { on: { ITEM_SELECTED: 'ITEM_DETAIL', VIEW_CART: 'CART' } },
    ITEM_DETAIL: { on: { ADD_TO_CART: 'MENU', BACK: 'MENU' } },
    CART: { on: { CHECKOUT: 'PAYMENT', CONTINUE_SHOPPING: 'MENU' } },
    PAYMENT: { on: { PAYMENT_COMPLETE: 'PROCESSING', PAYMENT_FAILED: 'CART' } },
    PROCESSING: { on: { ORDER_COMPLETE: 'RECEIPT' } },
    RECEIPT: { on: { NEW_ORDER: 'ATTRACT', TIMEOUT: 'ATTRACT' } },
    ADMIN: { on: { ADMIN_LOGOUT: 'ATTRACT' } },
  },
})
```

**KioskContext:** 13 properties including cartId, storeId, kioskId, selectedItem, currentOrder, paymentIntent, etc.

**Events:** 20+ typed events including `LANGUAGE_SELECTED`, `START_ORDER`, `ITEM_SELECTED`, `ADD_TO_CART`, `CHECKOUT`, `PAYMENT_COMPLETE`, `ADMIN_OVERRIDE`, `IDLE_TIMEOUT`.

**Actors:** Async actors via `fromPromise` for API calls.

## 2. Zustand - Session State

**File:** `packages/kiosk-state/src/sessionStore.ts`

```typescript
interface SessionState {
  workflowStage: WorkflowStage;
  laneMode: LaneMode;
  networkStatus: NetworkStatus;
  employeeAuth: AuthState | null;
  paymentState: PaymentState;
  setWorkflowStage: (stage: WorkflowStage) => void;
  setLaneMode: (mode: LaneMode) => void;
  setNetworkStatus: (status: NetworkStatus) => void;
}
```

**Purpose:** Ephemeral session state (no persistence). Used for:
- Current workflow stage
- Lane mode (attended/unattended)
- Network connectivity status
- Employee authentication state
- Payment flow state

**Transition Guards:** `ALLOWED_TRANSITIONS` map prevents invalid stage transitions.

## 3. Valtio - Cart State (Reactive Proxy)

**File:** `packages/kiosk-state/src/cartProxy.ts`

```typescript
const cartState = proxy<CartState>({
  items: [],
  totals: { subtotal: 0, tax: 0, total: 0 },
  version: 0,
  status: 'active',
  storeId: null,
  kioskId: null,
})
```

**Reactivity:** Components that access `cartState` automatically re-render on mutation.

**CRDT Integration:**
1. Valtio proxy mutation triggers `subscribe()` callback
2. Changes forwarded to CRDT Web Worker
3. Worker merges with local IndexedDB state
4. Debounced (800ms) API sync when online

**Mutations:**
- `addLineItem(item, quantity, modifiers)`
- `updateLineQuantity(lineId, quantity)`
- `removeLineItem(lineId)`
- `clearCart()`
- `applyGhostCart(cartSnapshot)`

## 4. TanStack Query - Server Cache

**File:** `apps/kiosk/src/state/queryClient.ts`

```typescript
const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      staleTime: 60_000,     // 60s before refetch
      gcTime: 1_800_000,     // 30min garbage collection
      retry: 3,              // Max retries
      retryDelay: (attempt) => Math.min(1000 * 2 ** attempt, 10000),
      networkMode: 'offlineFirst',
    },
  },
})
```

**Offline-First Mode:** TanStack Query's `networkMode: 'offlineFirst'` allows stale data to be shown immediately while background refetch happens.

**Key Query Keys:**
- `['menu', storeId]` - Store menu
- `['item', itemId]` - Single item
- `['inventory', storeId, itemId]` - Stock level
- `['order', orderId]` - Order details
- `['categories', storeId]` - Menu categories

## State Flow Summary

| Concern | Store | Persistence | Update Trigger |
|---------|-------|-------------|----------------|
| UI workflow | XState | None | User actions, events |
| Session | Zustand | None | Auth, network changes |
| Cart data | Valtio | IndexedDB (via CRDT worker) | User adds/removes items |
| Server data | TanStack Query | Memory + IndexedDB cache | API responses, mutations |
| Language/i18n | Zustand | localStorage | User selection |
| Admin state | React Context | Memory | Auth, navigation |
