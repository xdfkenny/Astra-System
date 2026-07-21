# Plugin Architecture

## Overview

Astra-System supports extensibility through several mechanisms that allow independently-developed components to integrate without modifying core code.

## Extension Mechanisms

| Mechanism | Language | Scope | Example |
|-----------|----------|-------|---------|
| Module Federation | TypeScript | Frontend MFEs | kiosk-menu, kiosk-cart, kiosk-payment |
| gRPC Service | Go | Backend services | legacy-pos-adapter |
| Rust FFI | Rust | Payment hardware | verifone-ffi bridge |
| Web Workers | TypeScript | Off-main-thread | CRDT worker, totals worker |
| Service Worker | TypeScript | Caching, sync | Background sync queue |
| OTA Remote Modules | TypeScript | Dynamic module loading | remote-modules.ts |
| Strangler Fig Adapter | Go | Legacy migration | legacy-pos-adapter |

## Module Federation Remotes

Frontend micro-frontends can be independently developed, deployed, and versioned.

**File:** `apps/kiosk/src/updater/remote-modules.ts`

```typescript
interface RemoteDefinition {
  name: string;           // Unique module name
  baseUrl: string;        // Remote entry URL
  version: string;        // Semver version
  timeoutMs: number;      // Load timeout (default 30s)
  fallbackUrl?: string;   // Fallback if primary fails
  sharedDeps: string[];   // Shared dependency names
}
```

**Adding a new MFE:**
1. Create new app in `astra-service/apps/`
2. Configure as Module Federation remote in `vite.config.ts`
3. Add RemoteDefinition to `remote-modules.ts`
4. Add route/component in the host shell
5. Deploy independently

## gRPC Service Extensions

New backend services follow the standard Go service pattern:

1. Define proto service in `proto/proto/`
2. Generate Go code with `buf generate`
3. Create service in `astra-service/services/`
4. Register with the gateway (or access directly)
5. Add Dockerfile in `infra/docker/`

## Rust FFI Bridge

For hardware integration (payment terminals, printers, scanners):

1. Create Rust crate in `astra-service/packages/` or `astra-service/daemons/`
2. Wrap vendor C SDK with safe Rust bindings
3. Expose via local HTTP/gRPC interface
4. Consumed by kiosk apps or Go services

## Web Workers

Heavy computations offloaded to Web Workers:

```typescript
// In main thread
const worker = new Worker(
  new URL('./workers/crdtWorker.ts', import.meta.url),
  { type: 'module' }
)
worker.postMessage({ type: 'MERGE_CART_OPS', payload: delta })
worker.onmessage = (event) => { /* handle result */ }
```

## OTA Remote Modules

Dynamic module loading for A/B testing, feature flags, and hotfixes without full deployment.

See [Module Federation](../frontend/module-federation.md) for details.
