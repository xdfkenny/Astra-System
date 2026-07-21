# Micro-Frontend Architecture

## Overview

Astra-System uses **Module Federation** (via `@module-federation/vite`) to compose independently-developed micro-frontends into a unified kiosk experience. Each MFE is independently built, versioned, deployed, and updated.

## Architecture

```
                           ┌──────────────────┐
                           │   kiosk (Host)   │
                           │  React 19 + Vite │
                           │  ModuleFederation │
                           └──┬───┬───┬───┬───┘
                              │   │   │   │
              ┌───────────────┘   │   │   └───────────────┐
              │                   │   │                   │
     ┌────────┴────────┐  ┌──────┴───┴──────┐  ┌─────────┴──────────┐
     │  kiosk-menu     │  │  kiosk-cart     │  │  kiosk-payment     │
     │  (Remote: Menu) │  │  (Remote: Cart) │  │  (Remote: Payment) │
     │  Item grid      │  │  Cart view      │  │  Payment UI        │
     │  Search/filter  │  │  Totals         │  │  Method selection  │
     └─────────────────┘  └─────────────────┘  └────────────────────┘

     ┌──────────────────────────────────────────────────────────────┐
     │  kiosk-admin (Separate app, not federated)                   │
     │  React 19 + Vite + Apollo Client (GraphQL) + React Router   │
     └──────────────────────────────────────────────────────────────┘
```

## Applications

| App | Type | Purpose | Port |
|-----|------|---------|------|
| `kiosk` | Host shell | Main kiosk UI, workflow orchestration | 5180 |
| `kiosk-menu` | Remote | Menu browsing, search, item detail | Dev-only |
| `kiosk-cart` | Remote | Shopping cart view and management | Dev-only |
| `kiosk-payment` | Remote | Payment flow UI | Dev-only |
| `kiosk-shell` | Remote | Shell skeleton (WIP) | Dev-only |
| `kiosk-admin` | Standalone | Admin dashboard (GraphQL) | 5173 |
| `docs` | Standalone | VitePress documentation site | Dev-only |

## Module Federation Configuration

**File:** `apps/kiosk/vite.config.ts`

```typescript
federation({
  name: 'kiosk',
  remotes: {
    'astra_menu': 'http://localhost:5181/assets/remoteEntry.js',
    'astra_cart': 'http://localhost:5182/assets/remoteEntry.js',
    'astra_payment': 'http://localhost:5183/assets/remoteEntry.js',
  },
  shared: {
    react: { singleton: true, requiredVersion: '^19.0.0' },
    'react-dom': { singleton: true },
  },
})
```

**Shared Dependencies:**
- `react` (singleton, version 19)
- `react-dom` (singleton)
- `@xstate/react` (singleton)
- `zustand` (singleton)
- `valtio` (singleton)
- `shared-types` (via workspace package)

## Remote Module Loading

**File:** `apps/kiosk/src/updater/remote-modules.ts`

Dynamic script injection for loading remote modules with:
- **Timeout handling:** Configurable timeout (default 30s)
- **Fallback URLs:** Alternate URL if primary fails
- **Version pinning:** Each remote tracked by version
- **Atomic rollback:** Preserve previous version on failure

```typescript
interface RemoteDefinition {
  name: string;
  baseUrl: string;
  version: string;
  timeoutMs: number;
  fallbackUrl?: string;
  sharedDeps: string[];
}
```

## Versioning & Deployment

- Each MFE independently versioned (package.json + changesets)
- Versioned `remoteEntry.js` files cache-busted by content hash
- OTA updater manages remote module URL updates
- Atomic swap: load new version, health check, switch traffic

## Cross-MFE Communication

| Mechanism | Purpose |
|-----------|---------|
| React Context (shared singleton) | Shared providers (Theme, i18n, Auth) |
| Zustand store | Session state, network status |
| Valtio proxy | Cart state (shared across MFEs) |
| XState machine | Workflow stage transitions |
| Events | Custom DOM events for MFE-boundary communication |

## Styling Isolation

- **Tailwind CSS v4** for apps (with `important` prefix strategy)
- **Tailwind CSS v3** for `design-system` package
- **CSS Modules** for component-scoped styles
- Design tokens shared via `packages/design-tokens/`
