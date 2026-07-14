import type { CartState, CartResponse } from "@astra/shared-types";
import { uuidV7 } from "@astra/shared-types";

// Define the API client interface
interface ApiClient {
  createCart(storeId: string, kioskId: string, sessionId: string): Promise<CartResponse>;
  addItemToCart(
    cartId: string,
    menuItemId: string,
    nameSnapshot: string,
    unitPriceCentsSnapshot: number,
    quantity: number,
    modifiers: {
      modifierId: string;
      optionId: string;
      priceDeltaCents: number;
    }[],
    notes?: string,
    weightGrams?: number,
  ): Promise<CartResponse>;
  updateCart(
    cartId: string,
    request: {
      lines: {
        lineId?: string;
        menuItemId: string;
        nameSnapshot: string;
        unitPriceCentsSnapshot: number;
        quantity: number;
        modifiers: {
          modifierId: string;
          optionId: string;
          priceDeltaCents: number;
        }[],
        notes?: string;
        weightGrams?: number;
      }[],
    },
  ): Promise<CartResponse>;
  getCart(cartId: string): Promise<CartResponse>;
  checkoutCart(
    cartId: string,
    method: "credit_debit" | "nfc_apple_pay" | "nfc_google_pay" | "qr_code" | "cash_recycler",
  ): Promise<{ checkoutId: string; paymentIntentId: string }>;
}

export type CheckoutMethod =
  | "credit_debit"
  | "nfc_apple_pay"
  | "nfc_google_pay"
  | "qr_code"
  | "cash_recycler";

interface CartLineInput {
  lineId?: string;
  menuItemId: string;
  nameSnapshot: string;
  unitPriceCentsSnapshot: number;
  quantity: number;
  modifiers: {
    modifierId: string;
    optionId: string;
    priceDeltaCents: number;
  }[];
  notes?: string;
  weightGrams?: number;
}

// Resilience tuning. The external API is file-backed and flaky, so we bound
// each call with a timeout and retry only genuinely transient failures.
const READ_TIMEOUT_MS = 16_000;
const MUTATE_TIMEOUT_MS = 22_000;
const CREATE_TIMEOUT_MS = 22_000;
const CHECKOUT_TIMEOUT_MS = 35_000;

const MAX_ATTEMPTS = 3;
const BASE_BACKOFF_MS = 400;
const MAX_BACKOFF_MS = 4_000;

class TimeoutError extends Error {
  constructor(message: string) {
    super(message);
    this.name = "TimeoutError";
  }
}

/**
 * API-based cart operations that sync with the backend.
 * These operations work alongside the local cart proxy for offline-first support.
 *
 * The methods here are resilient against the external API's failure modes:
 * - every call is bounded by a timeout,
 * - transient failures (network errors, timeouts, 5xx, 429) are retried with
 *   exponential backoff,
 * - a 404/409 (cart expired or version conflict on the server) triggers a
 *   transparent cart re-create and a single retry, so a flaky backend never
 *   permanently breaks the sync loop.
 */
export class ApiCart {
  private cartId: string;
  private kioskId: string;
  private sessionId: string;
  private storeId: string;

  constructor(kioskId: string, storeId: string) {
    this.kioskId = kioskId;
    this.storeId = storeId;
    this.cartId = uuidV7();
    this.sessionId = uuidV7();
  }

  /**
   * Set the API client to use for making requests.
   */
  static setApiClient(client: ApiClient): void {
    apiClient = client;
  }

  /**
   * Create a new cart on the server.
   */
  async createCart(): Promise<CartState> {
    if (!apiClient) {
      console.warn("API client not initialized; using local cart");
      return this.createLocalCart();
    }

    try {
      const response = await ApiCart.withTimeout(
        apiClient.createCart(this.storeId, this.kioskId, this.sessionId),
        CREATE_TIMEOUT_MS,
        "createCart",
      );
      this.cartId = response.cartId;
      return this.responseToCartState(response);
    } catch (error) {
      console.warn("Failed to create cart on server; using local cart:", error);
      // Return a local cart as fallback so the kiosk keeps working offline.
      return this.createLocalCart();
    }
  }

  /**
   * Create a local cart as fallback when API is unavailable.
   */
  private createLocalCart(): CartState {
    return {
      cartId: this.cartId,
      kioskId: this.kioskId,
      sessionId: this.sessionId,
      storeId: this.storeId,
      lines: [],
      version: 0,
      currency: "USD",
      createdAtMs: Date.now(),
      updatedAtMs: Date.now(),
    };
  }

  /**
   * Convert CartResponse to CartState.
   */
  private responseToCartState(response: CartResponse): CartState {
    return {
      cartId: response.cartId,
      kioskId: response.kioskId,
      sessionId: response.sessionId,
      storeId: response.storeId,
      lines: response.lines.map(line => ({
        ...line,
        modifiers: line.modifiers.map(mod => ({
          modifierId: mod["modifierId"] as string,
          optionId: mod["optionId"] as string,
          priceDeltaCents: mod["priceDeltaCents"] as number,
        })),
      })),
      version: response.version,
      currency: "USD",
      createdAtMs: new Date(response.expiresAt).getTime(),
      updatedAtMs: new Date(response.expiresAt).getTime(),
    };
  }

  /**
   * Add an item to the cart via API. Self-heals on a missing/conflicted cart.
   */
  async addItem(
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
  ): Promise<CartState> {
    return this.execute(
      async () =>
        this.responseToCartState(
          await this.requireClient().addItemToCart(
            this.cartId,
            menuItemId,
            nameSnapshot,
            unitPriceCentsSnapshot,
            quantity,
            modifiers,
            notes,
            weightGrams,
          ),
        ),
      { timeoutMs: MUTATE_TIMEOUT_MS, label: "addItem", recreateOnMissing: true },
    );
  }

  /**
   * Update cart items via API. Self-heals on a missing/conflicted cart.
   */
  async updateCart(lines: CartLineInput[]): Promise<CartState> {
    return this.execute(
      async () =>
        this.responseToCartState(
          await this.requireClient().updateCart(this.cartId, { lines }),
        ),
      { timeoutMs: MUTATE_TIMEOUT_MS, label: "updateCart", recreateOnMissing: true },
    );
  }

  /**
   * Get the current cart from the server. Self-heals on a missing cart.
   */
  async getCart(): Promise<CartState> {
    return this.execute(
      async () => this.responseToCartState(await this.requireClient().getCart(this.cartId)),
      { timeoutMs: READ_TIMEOUT_MS, label: "getCart", recreateOnMissing: true },
    );
  }

  /**
   * Checkout the cart via API.
   */
  async checkout(method: CheckoutMethod): Promise<{ checkoutId: string; paymentIntentId: string }> {
    return this.execute(
      async () => this.requireClient().checkoutCart(this.cartId, method),
      { timeoutMs: CHECKOUT_TIMEOUT_MS, label: "checkout", recreateOnMissing: false },
    );
  }

  /**
   * Reset the cart.
   */
  reset(): void {
    this.cartId = uuidV7();
    this.sessionId = uuidV7();
  }

  /**
   * Get the current cart ID.
   */
  getCartId(): string {
    return this.cartId;
  }

  /**
   * Get the current session ID.
   */
  getSessionId(): string {
    return this.sessionId;
  }

  // --- Internal resilience helpers -----------------------------------------

  private requireClient(): ApiClient {
    if (!apiClient) {
      throw new Error("API client not initialized");
    }
    return apiClient;
  }

  private static withTimeout<T>(promise: Promise<T>, ms: number, label: string): Promise<T> {
    return new Promise<T>((resolve, reject) => {
      const timer = setTimeout(() => {
        reject(new TimeoutError(`ApiCart: ${label} timed out after ${ms}ms`));
      }, ms);
      promise.then(
        (value) => {
          clearTimeout(timer);
          resolve(value);
        },
        (error: unknown) => {
          clearTimeout(timer);
          reject(error instanceof Error ? error : new Error(String(error)));
        },
      );
    });
  }

  private static statusOf(error: unknown): number | null {
    if (error && typeof error === "object" && "statusCode" in error) {
      const statusCode = (error as { statusCode?: unknown }).statusCode;
      return typeof statusCode === "number" ? statusCode : null;
    }
    return null;
  }

  private static isRetryableNetworkError(error: unknown): boolean {
    if (error instanceof TypeError) return true;
    if (error && typeof error === "object" && "code" in error) {
      const code = (error as { code?: unknown }).code;
      return (
        code === "ECONNRESET" ||
        code === "ETIMEDOUT" ||
        code === "ENOTFOUND" ||
        code === "ECONNREFUSED"
      );
    }
    return false;
  }

  private static shouldRetry(error: unknown, status: number | null): boolean {
    if (status !== null) {
      if (status >= 500) return true;
      if (status === 429 || status === 404 || status === 409) return true;
      return false;
    }
    return ApiCart.isRetryableNetworkError(error) || error instanceof TimeoutError;
  }

  private static delay(ms: number): Promise<void> {
    return new Promise<void>((resolve) => setTimeout(resolve, ms));
  }

  /**
   * Run an API operation with a timeout, bounded retries for transient
   * failures, and transparent cart re-creation when the server reports the
   * cart as missing or conflicted.
   */
  private async execute<T>(
    op: () => Promise<T>,
    opts: { timeoutMs: number; label: string; recreateOnMissing: boolean },
  ): Promise<T> {
    let lastError: unknown;

    for (let attempt = 0; attempt < MAX_ATTEMPTS; attempt++) {
      try {
        return await ApiCart.withTimeout(op(), opts.timeoutMs, opts.label);
      } catch (error) {
        lastError = error;
        const status = ApiCart.statusOf(error);

        if (opts.recreateOnMissing && (status === 404 || status === 409) && attempt < MAX_ATTEMPTS - 1) {
          console.warn(`ApiCart: cart ${status === 409 ? "conflict" : "missing"} (${opts.label}); recreating`);
          await this.recreateCart();
          continue;
        }

        if (!ApiCart.shouldRetry(error, status) || attempt === MAX_ATTEMPTS - 1) {
          throw error;
        }

        await ApiCart.delay(Math.min(BASE_BACKOFF_MS * 2 ** attempt, MAX_BACKOFF_MS));
      }
    }

    throw lastError;
  }

  /**
   * Create a fresh server cart and adopt its id. Falls back to a local cart
   * if the server is unreachable, which is safe because callers send the full
   * line set on every update (the local cart id is replaced on the next sync).
   */
  private async recreateCart(): Promise<void> {
    try {
      const state = await this.createCart();
      this.cartId = state.cartId;
      this.sessionId = state.sessionId;
    } catch (error) {
      console.warn("ApiCart: cart recreate failed; will retry operation", error);
    }
  }
}

// This will be set by the consumer of the ApiCart class
let apiClient: ApiClient | null = null;
