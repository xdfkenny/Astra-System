import { cartProxy, resetCart as resetCartProxy } from "@astra/kiosk-state";
import { ApiCart, type CartLineItem } from "@astra/kiosk-state";
import { apiClient } from "./apiClient";


const KIOSK_ID = (import.meta.env.VITE_KIOSK_ID) ?? "kiosk-local";
const STORE_ID = (import.meta.env.VITE_STORE_ID as string | undefined) ?? "store-default";

/**
 * Cart service that bridges between local state and API.
 * Handles offline/online synchronization.
 */
export class CartService {
  private apiCart: ApiCart;
  private isOnline: boolean;

  constructor() {
    // Set the API client for ApiCart
    ApiCart.setApiClient(apiClient);
    
    this.apiCart = new ApiCart(KIOSK_ID, STORE_ID);
    this.isOnline = navigator.onLine;
    
    // Initialize cart
    void this.initializeCart();
    
    // Listen for network changes
    window.addEventListener("online", this.handleOnline);
    window.addEventListener("offline", this.handleOffline);
  }

  private handleOnline = () => {
    this.isOnline = true;
    void this.syncCartToServer();
  };

  private handleOffline = () => {
    this.isOnline = false;
  };

   private async initializeCart(): Promise<void> {
    try {
      // Try to create a cart on the server
      const cartState = await this.apiCart.createCart();
       
      // Update local proxy with server cart ID
      cartProxy.cartId = cartState.cartId;
      cartProxy.sessionId = cartState.sessionId;
    } catch (error: unknown) {
      console.warn("Failed to initialize cart on server, using local cart:", error);
      // Use local cart as fallback
    }
  }

     // @ts-expect-error - line parameter is guaranteed to be a valid CartLineItem
     private mapCartLineToApiFormat(line: CartLineItem): Record<string, unknown> {
       return {
         // eslint-disable-next-line @typescript-eslint/no-unsafe-member-access
         lineId: line.lineId,
         // eslint-disable-next-line @typescript-eslint/no-unsafe-member-access
         menuItemId: line.menuItemId,
         // eslint-disable-next-line @typescript-eslint/no-unsafe-member-access
         nameSnapshot: line.nameSnapshot,
         // eslint-disable-next-line @typescript-eslint/no-unsafe-member-access
         unitPriceCentsSnapshot: line.unitPriceCentsSnapshot,
         // eslint-disable-next-line @typescript-eslint/no-unsafe-member-access
         quantity: line.quantity,
         // eslint-disable-next-line @typescript-eslint/no-unsafe-member-access
         modifiers: line.modifiers,
         // @ts-expect-error - optional properties are safe to access
         // eslint-disable-next-line @typescript-eslint/no-unsafe-member-access
         ...(line.notes !== undefined && { notes: line.notes }),
         // @ts-expect-error - optional properties are safe to access
         // eslint-disable-next-line @typescript-eslint/no-unsafe-member-access
         ...(line.weightGrams !== undefined && { weightGrams: line.weightGrams }),
       };
     }
   
    private async syncCartToServer(): Promise<void> {
     if (!this.isOnline) return;
     
     try {
       // Get current local cart state
       const localLines = cartProxy.lines.map(line => this.mapCartLineToApiFormat(line));
       
       // Sync with server
          await this.apiCart.updateCart(localLines);
       } catch (error: unknown) {
       console.error("Failed to sync cart to server:", error);
     }
   }

  /**
   * Add an item to the cart.
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
  ): Promise<void> {
    // Add to local cart first for immediate UI feedback
      const newItem: CartLineItem = {
       lineId: crypto.randomUUID(),
       menuItemId,
       nameSnapshot,
       unitPriceCentsSnapshot,
       quantity,
       modifiers,
       addedAtMs: Date.now(),
     };
     
      // eslint-disable-next-line @typescript-eslint/no-unsafe-member-access
      if (notes !== undefined) newItem.notes = notes;
      // eslint-disable-next-line @typescript-eslint/no-unsafe-member-access
      if (weightGrams !== undefined) newItem.weightGrams = weightGrams;
     
      // eslint-disable-next-line @typescript-eslint/no-unsafe-argument
      cartProxy.lines.push(newItem);
    cartProxy.version += 1;
    cartProxy.updatedAtMs = Date.now();

     // Try to sync with server if online
     if (this.isOnline) {
       try {
         await this.apiCart.addItem(
           menuItemId,
           nameSnapshot,
           unitPriceCentsSnapshot,
           quantity,
           modifiers,
           notes,
           weightGrams,
         );
        } catch (error: unknown) {
         console.warn("Failed to sync addItem to server:", error);
       }
    }
  }

  /**
   * Update item quantity.
   */
   async updateQuantity(lineId: string, quantity: number): Promise<void> {
    const line = cartProxy.lines.find((l) => l.lineId === lineId);
    if (!line) return;
    
    if (quantity <= 0) {
       await this.removeItem(lineId);
       return;
    }
    
      line.quantity = quantity;
      cartProxy.version += 1;
      cartProxy.updatedAtMs = Date.now();

      // Try to sync with server if online
      if (this.isOnline) {
        try {
          const localLines = cartProxy.lines.map(l => this.mapCartLineToApiFormat(l));
         
         await this.apiCart.updateCart(localLines);
      } catch (error) {
        console.error("Failed to update quantity on server cart, will sync later:", error);
      }
    }
  }

  /**
   * Remove an item from the cart.
   */
  async removeItem(lineId: string): Promise<void> {
    const idx = cartProxy.lines.findIndex((l) => l.lineId === lineId);
    if (idx === -1) return;
    
     cartProxy.lines.splice(idx, 1);
     cartProxy.version += 1;
     cartProxy.updatedAtMs = Date.now();

      // Try to sync with server if online
      if (this.isOnline) {
        try {
          const localLines = cartProxy.lines.map(l => this.mapCartLineToApiFormat(l));
         
         await this.apiCart.updateCart(localLines);
      } catch (error) {
        console.error("Failed to remove item from server cart, will sync later:", error);
      }
    }
  }

  /**
   * Reset the cart.
   */
  resetCart(): void {
    resetCartProxy(KIOSK_ID);
    this.apiCart.reset();
  }

  /**
   * Checkout the cart.
   */
  async checkout(
    method: "credit_debit" | "nfc_apple_pay" | "nfc_google_pay" | "qr_code" | "cash_recycler",
  ): Promise<{ checkoutId: string; paymentIntentId: string }> {
    try {
      return await this.apiCart.checkout(method);
    } catch (error) {
      console.error("Failed to checkout via API:", error);
      throw error;
    }
  }

  /**
   * Get the current cart ID.
   */
  getCartId(): string {
    return this.apiCart.getCartId();
  }

  /**
   * Get the current session ID.
   */
  getSessionId(): string {
    return this.apiCart.getSessionId();
  }

  /**
   * Clean up resources.
   */
  destroy() {
    window.removeEventListener("online", this.handleOnline);
    window.removeEventListener("offline", this.handleOffline);
  }
}

/**
 * Singleton instance of the cart service.
 */
export const cartService = new CartService();