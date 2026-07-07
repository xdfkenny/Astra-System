import { proxy, subscribe } from "valtio";
import type { CartLineItem, CartState } from "@astra/shared-types";
import { uuidV7 } from "@astra/shared-types";

/**
 * Cart reactive state (Valtio proxy).
 *
 * WHY Valtio here specifically: every mutation is a plain JS object mutation
 * (`cartProxy.lines.push(...)`), which we mirror 1:1 into CRDT operations
 * sent to the astra-syncd WASM module (see workers/crdtWorker.ts). Valtio's
 * proxy-diffing gives us "what changed" for free via subscribe(), so we never
 * hand-roll a change-detection layer on top of an immutable store.
 */
export const cartProxy = proxy<CartState>({
  cartId: uuidV7(),
  kioskId:
    // eslint-disable-next-line @typescript-eslint/dot-notation
    import.meta.env["VITE_KIOSK_ID"] ?? "unknown-kiosk",
  sessionId: uuidV7(),
  lines: [],
  version: 0,
  currency: "USD",
  createdAtMs: Date.now(),
  updatedAtMs: Date.now(),
});

export function addLineItem(item: Omit<CartLineItem, "lineId" | "addedAtMs">): void {
  cartProxy.lines.push({
    ...item,
    lineId: uuidV7(),
    addedAtMs: Date.now(),
  });
  cartProxy.version += 1;
  cartProxy.updatedAtMs = Date.now();
}

export function updateLineQuantity(lineId: string, quantity: number): void {
  const line = cartProxy.lines.find((l) => l.lineId === lineId);
  if (!line) return;
  if (quantity <= 0) {
    removeLineItem(lineId);
    return;
  }
  line.quantity = quantity;
  cartProxy.version += 1;
  cartProxy.updatedAtMs = Date.now();
}

export function removeLineItem(lineId: string): void {
  const idx = cartProxy.lines.findIndex((l) => l.lineId === lineId);
  if (idx === -1) return;
  cartProxy.lines.splice(idx, 1);
  cartProxy.version += 1;
  cartProxy.updatedAtMs = Date.now();
}

export function resetCart(kioskId: string): void {
  cartProxy.cartId = uuidV7();
  cartProxy.sessionId = uuidV7();
  cartProxy.kioskId = kioskId;
  cartProxy.lines = [];
  cartProxy.version = 0;
  cartProxy.createdAtMs = Date.now();
  cartProxy.updatedAtMs = Date.now();
}

/** Derived, memoized item count — recomputes only when `lines` changes. */
export const derivedCart = proxy<{ itemCount: number; isEmpty: boolean }>({
  itemCount: 0,
  isEmpty: true,
});

subscribe(cartProxy, () => {
  derivedCart.itemCount = cartProxy.lines.reduce((sum, l) => sum + l.quantity, 0);
  derivedCart.isEmpty = cartProxy.lines.length === 0;
});

/**
 * Bridges every cart mutation into the CRDT worker as an append-only op log.
 * This is the seam where local Valtio state becomes durable, sync-able state.
 * Call once at app boot.
 */
export function bridgeCartToCrdtWorker(postOp: (op: CrdtCartOp) => void): () => void {
  return subscribe(cartProxy, (ops) => {
    for (const op of ops) {
      const opType = op[0];
      postOp({
        kind: "cart_mutation",
        path: op[1].map(String),
        opType,
        cartId: cartProxy.cartId,
        version: cartProxy.version,
        timestampMs: Date.now(),
      });
    }
  });
}

export interface CrdtCartOp {
  readonly kind: "cart_mutation";
  readonly path: readonly string[];
  readonly opType: "set" | "delete";
  readonly cartId: string;
  readonly version: number;
  readonly timestampMs: number;
}
