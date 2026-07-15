import type {
  MenuResponse,
  CartResponse,
  PaymentResult,
  OrderResponse,
  KioskHeartbeatResponse,
} from "@astra/shared-types";

/**
 * API client for the Meriandes Self-Service Cafeteria Kiosk system.
 * Handles authentication, API requests, and error handling.
 */
export class AstraApiClient {
  private baseUrl: string;
  private authToken: string | null;

  constructor() {
    this.baseUrl = import.meta.env.VITE_API_GATEWAY_URL ?? "http://localhost:8080";
    this.authToken = null;
  }

  /**
   * Set the authentication token for subsequent requests.
   */
  setAuthToken(token: string): void {
    this.authToken = token;
  }

  /**
   * Clear the authentication token.
   */
  clearAuthToken(): void {
    this.authToken = null;
  }

  /**
   * Get the current authentication token.
   */
  getAuthToken(): string | null {
    return this.authToken;
  }

  /**
   * Make an authenticated API request.
   */
  private async request<T>(endpoint: string, options: RequestInit = {}): Promise<T> {
    const url = `${this.baseUrl}${endpoint}`;
    const headers = new Headers({
      "Content-Type": "application/json",
    });

    if (this.authToken) {
      headers.set("Authorization", `Bearer ${this.authToken}`);
    }

    if (options.headers) {
      new Headers(options.headers).forEach((value, key) => {
        headers.set(key, value);
      });
    }

    try {
      const response = await fetch(url, {
        ...options,
        headers,
      });

      if (!response.ok) {
        const errorData = (await response.json().catch(() => ({}))) as unknown;
        throw new Error(
          (errorData as { message?: string }).message ??
            `API request failed: ${response.status} ${response.statusText}`,
        );
      }

      return (await response.json()) as T;
    } catch (error) {
      console.error(`API request failed for ${endpoint}:`, error);
      throw error;
    }
  }

  /**
   * Health Check - GET /health
   * Check the health status of the API gateway.
   */
  async checkHealth(): Promise<{ status: string }> {
    return this.request<{ status: string }>("/health");
  }

  /**
   * Get Menu Catalog - GET /v1/menu
   * Fetch the complete menu catalog for the kiosk.
   */
  async getMenuCatalog(): Promise<MenuResponse> {
    // Force mock data when API is not available for development
    try {
      return await this.request<MenuResponse>("/v1/menu");
    } catch {
      console.warn("API not available, using mock data for development");
      // Import mock data dynamically to avoid circular dependency
      const { mockMenuResponse } = await import("../routes/mockMenuData");
      return mockMenuResponse;
    }
  }

  /**
   * Get Cart - GET /v1/carts/{cartId}
   * Retrieve an existing cart.
   */
  async getCart(cartId: string): Promise<CartResponse> {
    return this.request<CartResponse>(`/v1/carts/${cartId}`);
  }

  /**
   * Create Cart - POST /v1/carts
   * Create a new cart for the current session.
   */
  async createCart(
    storeId: string,
    kioskId: string,
    sessionId: string,
    customerPhone?: string,
  ): Promise<CartResponse> {
    return this.request<CartResponse>("/v1/carts", {
      method: "POST",
      body: JSON.stringify({
        storeId,
        kioskId,
        sessionId,
        customerPhone,
      }),
    });
  }

  /**
   * Add Item to Cart - POST /v1/carts/{cartId}/items
   * Add an item to the cart with optional modifiers.
   */
  async addItemToCart(
    cartId: string,
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
  ): Promise<CartResponse> {
    return this.request<CartResponse>(`/v1/carts/${cartId}/items`, {
      method: "POST",
      body: JSON.stringify({
        menuItemId,
        nameSnapshot,
        unitPriceCentsSnapshot,
        quantity,
        modifiers,
        notes,
        weightGrams,
      }),
    });
  }

  /**
   * Update Cart - PUT /v1/carts/{cartId}
   * Update cart details or line items.
   */
  async updateCart(
    cartId: string,
    updates: {
      sessionId?: string;
      customerPhone?: string;
      lines?: {
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
      }[];
      version?: number;
    },
  ): Promise<CartResponse> {
    return this.request<CartResponse>(`/v1/carts/${cartId}`, {
      method: "PUT",
      body: JSON.stringify(updates),
    });
  }

  /**
   * Checkout Cart - POST /v1/carts/{cartId}/checkout
   * Initiate the checkout process for a cart.
   */
  async checkoutCart(
    cartId: string,
    method: "credit_debit" | "nfc_apple_pay" | "nfc_google_pay" | "qr_code" | "cash_recycler",
    currency = "USD",
  ): Promise<{ checkoutId: string; paymentIntentId: string }> {
    return this.request<{ checkoutId: string; paymentIntentId: string }>(
      `/v1/carts/${cartId}/checkout`,
      {
        method: "POST",
        body: JSON.stringify({ method, currency }),
      },
    );
  }

  /**
   * Process Payment - POST /v1/payments
   * Process a payment for a checkout.
   */
  async processPayment(
    cartId: string,
    paymentIntentId: string,
    amountCents: number,
    currency = "USD",
    method: "credit_debit" | "nfc_apple_pay" | "nfc_google_pay" | "qr_code" | "cash_recycler",
  ): Promise<PaymentResult> {
    return this.request<PaymentResult>("/v1/payments", {
      method: "POST",
      body: JSON.stringify({
        cartId,
        paymentIntentId,
        amountCents,
        currency,
        method,
      }),
    });
  }

  /**
   * Get Order - GET /v1/orders/{orderId}
   * Retrieve order details.
   */
  async getOrder(orderId: string): Promise<OrderResponse> {
    return this.request<OrderResponse>(`/v1/orders/${orderId}`);
  }

  /**
   * Create Order - POST /v1/orders
   * Create an order from a completed payment.
   */
  async createOrder(cartId: string, paymentId: string): Promise<OrderResponse> {
    return this.request<OrderResponse>("/v1/orders", {
      method: "POST",
      body: JSON.stringify({ cartId, paymentId }),
    });
  }

  /**
   * Send Kiosk Heartbeat - POST /v1/kiosks/{kioskId}/heartbeat
   * Send a heartbeat to indicate kiosk is active.
   */
  async sendKioskHeartbeat(
    kioskId: string,
    storeId: string,
    syncStatus: "online" | "offline" | "degraded" | "maintenance" = "online",
    peerCount?: number,
    queueDepth?: number,
  ): Promise<KioskHeartbeatResponse> {
    return this.request<KioskHeartbeatResponse>(`/v1/kiosks/${kioskId}/heartbeat`, {
      method: "POST",
      body: JSON.stringify({
        kioskId,
        storeId,
        syncStatus,
        peerCount,
        queueDepth,
      }),
    });
  }

  /**
   * Get Announcements - GET /v1/announcements
   * Retrieve system announcements.
   */
  async getAnnouncements(): Promise<{
    announcements: {
      id: string;
      title: string;
      message: string;
      severity: "info" | "warning" | "critical";
      expiresAt: string;
    }[];
  }> {
    return this.request<{
      announcements: {
        id: string;
        title: string;
        message: string;
        severity: "info" | "warning" | "critical";
        expiresAt: string;
      }[];
    }>("/v1/announcements");
  }

  /**
   * Admin Login - POST /v1/admin/auth/login
   * Authenticate as an admin user.
   */
  async adminLogin(
    username: string,
    password: string,
  ): Promise<{ token: string; expiresAt: string }> {
    return this.request<{ token: string; expiresAt: string }>("/v1/admin/auth/login", {
      method: "POST",
      body: JSON.stringify({ username, password }),
    });
  }

  /**
   * Admin Dashboard - GET /v1/admin/dashboard
   * Get admin dashboard data.
   */
  async getAdminDashboard(): Promise<{
    kioskCount: number;
    activeOrders: number;
    revenueToday: number;
    syncStatus: Record<string, "online" | "offline" | "degraded">;
  }> {
    return this.request<{
      kioskCount: number;
      activeOrders: number;
      revenueToday: number;
      syncStatus: Record<string, "online" | "offline" | "degraded">;
    }>("/v1/admin/dashboard");
  }

  /**
   * P2P Sync - POST /v1/p2p/sync
   * Sync data with peer kiosks.
   */
  async p2pSync(
    kioskId: string,
    storeId: string,
    events: Record<string, unknown>[],
    vectorClock: Record<string, number>,
  ): Promise<{ accepted: boolean; conflicts: number; vectorClock: Record<string, number> }> {
    return this.request<{
      accepted: boolean;
      conflicts: number;
      vectorClock: Record<string, number>;
    }>("/v1/p2p/sync", {
      method: "POST",
      body: JSON.stringify({ kioskId, storeId, events, vectorClock }),
    });
  }

  /**
   * Debug Info - GET /v1/debug/info
   * Get debug information about the system.
   */
  async getDebugInfo(): Promise<{
    version: string;
    uptime: number;
    memoryUsage: number;
    activeConnections: number;
    databaseStatus: "healthy" | "degraded" | "unhealthy";
  }> {
    return this.request<{
      version: string;
      uptime: number;
      memoryUsage: number;
      activeConnections: number;
      databaseStatus: "healthy" | "degraded" | "unhealthy";
    }>("/v1/debug/info");
  }
}

/**
 * Singleton instance of the API client.
 */
export const apiClient = new AstraApiClient();
