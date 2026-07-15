import { cartProxy, resetCart as resetCartProxy } from "@astra/kiosk-state";
import { ApiCart } from "@astra/kiosk-state";
import type { CartLineItem } from "@astra/shared-types";
import { apiClient } from "./apiClient";

const KIOSK_ID = (import.meta.env as Record<string, string | undefined>)["VITE_KIOSK_ID"] ?? "kiosk-local";
const STORE_ID = (import.meta.env as Record<string, string | undefined>)["VITE_STORE_ID"] ?? "store-default";

const DEBOUNCE_MS = 800;
const MAX_RETRY_DELAY_MS = 30_000;
const BASE_RETRY_DELAY_MS = 1_000;

export class CartService {
  private apiCart: ApiCart;
  private isOnline: boolean;
  private syncDebounceTimer: ReturnType<typeof setTimeout> | null;
  private pendingSyncVersion: number;
  private retryDelay: number;
  private retryTimer: ReturnType<typeof setTimeout> | null;
  private destroyed: boolean;

  constructor() {
    ApiCart.setApiClient(apiClient);

    this.apiCart = new ApiCart(KIOSK_ID, STORE_ID);
    this.isOnline = navigator.onLine;
    this.syncDebounceTimer = null;
    this.pendingSyncVersion = 0;
    this.retryDelay = BASE_RETRY_DELAY_MS;
    this.retryTimer = null;
    this.destroyed = false;

    void this.initializeCart();

    window.addEventListener("online", this.handleOnline);
    window.addEventListener("offline", this.handleOffline);
  }

  private handleOnline = (): void => {
    this.isOnline = true;
    this.retryDelay = BASE_RETRY_DELAY_MS;
    void this.flushSync();
  };

  private handleOffline = (): void => {
    this.isOnline = false;
    this.cancelSync();
  };

  private scheduleSync(): void {
    if (this.syncDebounceTimer) {
      clearTimeout(this.syncDebounceTimer);
    }
    this.syncDebounceTimer = setTimeout(() => {
      void this.flushSync();
    }, DEBOUNCE_MS);
  }

  private cancelSync(): void {
    if (this.syncDebounceTimer) {
      clearTimeout(this.syncDebounceTimer);
      this.syncDebounceTimer = null;
    }
    if (this.retryTimer) {
      clearTimeout(this.retryTimer);
      this.retryTimer = null;
    }
  }

  private async flushSync(): Promise<void> {
    if (!this.isOnline || this.destroyed) return;
    if (this.syncDebounceTimer) {
      clearTimeout(this.syncDebounceTimer);
      this.syncDebounceTimer = null;
    }

    const version = ++this.pendingSyncVersion;
    const localLines = cartProxy.lines.map((line) => this.mapCartLineToApiFormat(line));

    try {
      await this.apiCart.updateCart(localLines);
      this.retryDelay = BASE_RETRY_DELAY_MS;
    } catch (error) {
      if (version !== this.pendingSyncVersion) return;
      console.warn("Cart sync failed, will retry:", error);
      this.scheduleRetry();
    }
  }

  private scheduleRetry(): void {
    if (this.destroyed || this.retryTimer) return;
    this.retryTimer = setTimeout(() => {
      this.retryTimer = null;
      void this.flushSync();
    }, this.retryDelay);
    this.retryDelay = Math.min(this.retryDelay * 2, MAX_RETRY_DELAY_MS);
  }

  private async initializeCart(): Promise<void> {
    try {
      const cartState = await this.apiCart.createCart();
      cartProxy.cartId = cartState.cartId;
      cartProxy.sessionId = cartState.sessionId;
    } catch (error: unknown) {
      console.warn("Failed to initialize cart on server, using local cart:", error);
    }
  }

  private mapCartLineToApiFormat(line: CartLineItem): {
    lineId?: string;
    menuItemId: string;
    nameSnapshot: string;
    unitPriceCentsSnapshot: number;
    quantity: number;
    modifiers: { modifierId: string; optionId: string; priceDeltaCents: number }[];
    notes?: string;
    weightGrams?: number;
  } {
    return {
      lineId: line.lineId,
      menuItemId: line.menuItemId,
      nameSnapshot: line.nameSnapshot,
      unitPriceCentsSnapshot: line.unitPriceCentsSnapshot,
      quantity: line.quantity,
      modifiers: line.modifiers,
      ...(line.notes !== undefined && { notes: line.notes }),
      ...(line.weightGrams !== undefined && { weightGrams: line.weightGrams }),
    };
  }

  private markDirty(): void {
    cartProxy.version += 1;
    cartProxy.updatedAtMs = Date.now();
    this.scheduleSync();
  }

  addItem(
    menuItemId: string,
    nameSnapshot: string,
    unitPriceCentsSnapshot: number,
    quantity: number,
    modifiers: {
      modifierId: string;
      optionId: string;
      priceDeltaCents: number;
    }[] = [],
    notes?: string,
    weightGrams?: number,
  ): void {
    const newItem: CartLineItem = {
      lineId: crypto.randomUUID(),
      menuItemId,
      nameSnapshot,
      unitPriceCentsSnapshot,
      quantity,
      modifiers,
      addedAtMs: Date.now(),
      ...(notes !== undefined && { notes }),
      ...(weightGrams !== undefined && { weightGrams }),
    };

    cartProxy.lines.push(newItem);
    this.markDirty();
  }

  updateQuantity(lineId: string, quantity: number): void {
    const line = cartProxy.lines.find((l) => l.lineId === lineId);
    if (!line) return;

    if (quantity <= 0) {
      this.removeItem(lineId);
      return;
    }

    line.quantity = quantity;
    this.markDirty();
  }

  removeItem(lineId: string): void {
    const idx = cartProxy.lines.findIndex((l) => l.lineId === lineId);
    if (idx === -1) return;

    cartProxy.lines.splice(idx, 1);
    this.markDirty();
  }

  resetCart(): void {
    this.cancelSync();
    resetCartProxy(KIOSK_ID);
    this.apiCart.reset();
  }

  async checkout(
    method: "credit_debit" | "nfc_apple_pay" | "nfc_google_pay" | "qr_code" | "cash_recycler",
  ): Promise<{ checkoutId: string; paymentIntentId: string }> {
    await this.flushSync();
    return this.apiCart.checkout(method);
  }

  getCartId(): string {
    return this.apiCart.getCartId();
  }

  getSessionId(): string {
    return this.apiCart.getSessionId();
  }

  destroy(): void {
    this.destroyed = true;
    this.cancelSync();
    window.removeEventListener("online", this.handleOnline);
    window.removeEventListener("offline", this.handleOffline);
  }
}

export const cartService = new CartService();

