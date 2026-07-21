# Module Federation

## Overview

Module Federation enables independently-developed micro-frontends to compose into a single runtime application. Each MFE is a separate Vite/React app that exposes components for dynamic loading.

## Architecture

```
Host (kiosk)                    Remote 1 (kiosk-menu)
┌──────────────────┐           ┌──────────────────┐
│  vite.config.ts  │──────────▶│  ModuleFederation │
│  remotes: {      │           │  exposes: {       │
│    astra_menu,   │           │    './MenuApp'    │
│    astra_cart,   │           │  }                │
│    astra_payment │           └──────────────────┘
│  }               │
│  shared: {       │           Remote 2 (kiosk-cart)
│    react,        │           ┌──────────────────┐
│    react-dom,    │──────────▶│  ModuleFederation │
│    zustand,      │           │  exposes: {       │
│    valtio        │           │    './CartApp'    │
│  }               │           │  }                │
└──────────────────┘           └──────────────────┘
```

## Dynamic Remote Loading

**File:** `apps/kiosk/src/updater/remote-modules.ts`

```typescript
interface RemoteDefinition {
  name: string;
  baseUrl: string;
  version: string;
  timeoutMs: number;
  fallbackUrl?: string;
  sharedDeps: string[];
}

const remotes: RemoteDefinition[] = [
  {
    name: 'astra_menu',
    baseUrl: 'https://cdn.astra.io/menu/v2.1.0',
    version: '2.1.0',
    timeoutMs: 30000,
    fallbackUrl: 'https://cdn.astra.io/menu/v2.0.0',
    sharedDeps: ['react', 'react-dom', 'zustand', 'valtio'],
  },
  // ...
]
```

## Shared Dependencies

Libraries marked as `shared` are loaded as singletons:
- `react` (^19.0.0)
- `react-dom` (^19.0.0)
- `@xstate/react`
- `zustand`
- `valtio`
- `shared-types` (workspace package)

## Versioning & Rollback

- Each MFE versioned independently (package.json + changesets)
- Versioned `remoteEntry.js` files cache-busted by content hash
- OTA updater manages remote URL updates
- Atomic swap: load new version → health check → switch traffic
- Automatic rollback on failure (preserves previous version)
