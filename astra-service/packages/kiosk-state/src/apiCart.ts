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

// This will be set by the consumer of the ApiCart class
let apiClient: ApiClient | null = null;

/**
 * API-based cart operations that sync with the backend.
 * These operations work alongside the local cart proxy for offline-first support.
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
      console.error("API client not initialized");
      return this.createLocalCart();
    }
    
    try {
      const response = await apiClient.createCart(
        this.storeId,
        this.kioskId,
        this.sessionId,
      );
      this.cartId = response.cartId;
      return this.responseToCartState(response);
    } catch (error) {
      console.error("Failed to create cart on server:", error);
      // Return a local cart as fallback
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
          priceDeltaCents: mod["priceDeltaCents"] as number
        }))
      })),
      version: response.version,
      currency: "USD",
      createdAtMs: new Date(response.expiresAt).getTime(),
      updatedAtMs: new Date(response.expiresAt).getTime(),
    };
  }

   /**
   * Add an item to the cart via API.
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
    if (!apiClient) {
      console.error("API client not initialized");
      throw new Error("API client not initialized");
    }
    
    try {
      const response = await apiClient.addItemToCart(
        this.cartId,
        menuItemId,
        nameSnapshot,
        unitPriceCentsSnapshot,
        quantity,
        modifiers,
        notes,
        weightGrams,
      );
      
      return this.responseToCartState(response);
    } catch (error) {
      console.error("Failed to add item to cart via API:", error);
      throw error;
    }
  }

   /**
   * Update cart items via API.
   */
  async updateCart(
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
      }[];
      notes?: string;
      weightGrams?: number;
    }[],
  ): Promise<CartState> {
    if (!apiClient) {
      console.error("API client not initialized");
      throw new Error("API client not initialized");
    }
    
    try {
      const response = await apiClient.updateCart(this.cartId, {
        lines,
      });
      
      return this.responseToCartState(response);
    } catch (error) {
      console.error("Failed to update cart via API:", error);
      throw error;
    }
  }

   /**
   * Get the current cart from the server.
   */
  async getCart(): Promise<CartState> {
    if (!apiClient) {
      console.error("API client not initialized");
      throw new Error("API client not initialized");
    }
    
    try {
      const response = await apiClient.getCart(this.cartId);
      
      return this.responseToCartState(response);
    } catch (error) {
      console.error("Failed to get cart from server:", error);
      throw error;
    }
  }

  /**
   * Checkout the cart via API.
   */
  async checkout(
    method: "credit_debit" | "nfc_apple_pay" | "nfc_google_pay" | "qr_code" | "cash_recycler",
  ): Promise<{ checkoutId: string; paymentIntentId: string }> {
    if (!apiClient) {
      console.error("API client not initialized");
      throw new Error("API client not initialized");
    }
    
    try {
      return await apiClient.checkoutCart(this.cartId, method);
    } catch (error) {
      console.error("Failed to checkout cart via API:", error);
      throw error;
    }
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
}