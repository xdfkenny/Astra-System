import type {
  MenuResponse,
  CartResponse,
  PaymentResult,
  OrderResponse,
  KioskHeartbeatResponse,
} from "@astra/shared-types";

interface RequestConfig {
  readonly timeoutMs: number;
  readonly deduplicate: boolean;
  readonly cacheTtlMs: number;
}

const DEFAULT_CONFIG: RequestConfig = {
  timeoutMs: 10_000,
  deduplicate: false,
  cacheTtlMs: 0,
} as const;

const READ_CONFIG: RequestConfig = {
  timeoutMs: 15_000,
  deduplicate: true,
  cacheTtlMs: 30_000,
} as const;

const MUTATE_CONFIG: RequestConfig = {
  timeoutMs: 20_000,
  deduplicate: false,
  cacheTtlMs: 0,
} as const;

const HEALTH_CONFIG: RequestConfig = {
  timeoutMs: 5_000,
  deduplicate: false,
  cacheTtlMs: 0,
} as const;

const MENU_CONFIG: RequestConfig = {
  timeoutMs: 20_000,
  deduplicate: true,
  cacheTtlMs: 60_000,
} as const;

interface CacheEntry<T> {
  data: T;
  expiresAt: number;
}

const responseCache = new Map<string, CacheEntry<unknown>>();
const inflightRequests = new Map<string, Promise<unknown>>();

function cacheKey(method: string, endpoint: string, body?: string): string {
  return `${method}:${endpoint}${body ? `:${body}` : ""}`;
}

// eslint-disable-next-line @typescript-eslint/no-unnecessary-type-parameters
function getCached<T>(key: string): T | null {
  const entry = responseCache.get(key);
  if (!entry) return null;
  if (Date.now() > entry.expiresAt) {
    responseCache.delete(key);
    return null;
  }
  return entry.data as T;
}

// eslint-disable-next-line @typescript-eslint/no-unnecessary-type-parameters
function setCache<T>(key: string, data: T, ttlMs: number): void {
  if (ttlMs <= 0) return;
  responseCache.set(key, { data, expiresAt: Date.now() + ttlMs });
}

export class ApiError extends Error {
  readonly statusCode: number;
  readonly code: string;
  readonly retryAfterMs: number | undefined;

  constructor(
    message: string,
    statusCode: number,
    code: string,
    retryAfterMs?: number,
  ) {
    super(message);
    this.name = "ApiError";
    this.statusCode = statusCode;
    this.code = code;
    this.retryAfterMs = retryAfterMs;
  }
}

export class AstraApiClient {
  private baseUrl: string;
  private authToken: string | null;

  constructor() {
    const env = import.meta.env as Record<string, string | undefined>;
    this.baseUrl = env["VITE_API_GATEWAY_URL"] ?? "http://localhost:8080";
    this.authToken = env["VITE_ASTRA_JWT"] ?? null;
  }

  setAuthToken(token: string): void {
    this.authToken = token;
  }

  clearAuthToken(): void {
    this.authToken = null;
  }

  getAuthToken(): string | null {
    return this.authToken;
  }

  private async request<T>(
    endpoint: string,
    options: RequestInit = {},
    config: RequestConfig = DEFAULT_CONFIG,
  ): Promise<T> {
    const url = `${this.baseUrl}${endpoint}`;
    const method = (options.method ?? "GET").toUpperCase();
    const bodyStr = options.body as string | undefined;
    const ck = cacheKey(method, endpoint, bodyStr);

    if (method === "GET" && config.cacheTtlMs > 0) {
      const cached = getCached<T>(ck);
      if (cached) return cached;
    }

    if (config.deduplicate) {
      const inflight = inflightRequests.get(ck);
      if (inflight) return inflight as Promise<T>;
    }

    const abortController = new AbortController();
    const timeoutId = setTimeout(() => {
      abortController.abort(new DOMException("Request timed out", "TimeoutError"));
    }, config.timeoutMs);

    const headers: Record<string, string> = {
      "Content-Type": "application/json",
      "Accept-Encoding": "gzip, br",
    };

    if (this.authToken) {
      headers["Authorization"] = `Bearer ${this.authToken}`;
    }

    if (options.headers) {
      const extra = new Headers(options.headers);
      for (const [k, v] of extra.entries()) {
        headers[k] = v;
      }
    }

    const fetchPromise = (async (): Promise<T> => {
      try {
        const response = await fetch(url, {
          ...options,
          headers,
          signal: abortController.signal,
        });

        clearTimeout(timeoutId);

        if (!response.ok) {
          const errorData = await response.json().catch(() => ({})) as Record<string, unknown>;
          const code = (errorData["code"] as string | undefined) ?? "UNKNOWN_ERROR";
          const message = (errorData["message"] as string | undefined)
            ?? `API request failed: ${response.status} ${response.statusText}`;
          const retryAfterRaw = response.headers.get("Retry-After");
          const retryAfterMs = retryAfterRaw ? parseInt(retryAfterRaw, 10) * 1000 : (errorData["retryAfterMs"] as number | undefined);

          throw new ApiError(message, response.status, code, retryAfterMs);
        }

        const data = await (response.json() as Promise<T>);

        if (config.cacheTtlMs > 0) {
          setCache(ck, data, config.cacheTtlMs);
        }

        return data;
      } catch (error) {
        clearTimeout(timeoutId);
        if (error instanceof ApiError) throw error;
        if (error instanceof DOMException && error.name === "TimeoutError") {
          throw new ApiError(
            `Request timed out after ${config.timeoutMs}ms: ${method} ${endpoint}`,
            0,
            "TIMEOUT",
          );
        }
        throw error;
      } finally {
        if (config.deduplicate) {
          inflightRequests.delete(ck);
        }
      }
    })();

    if (config.deduplicate) {
      inflightRequests.set(ck, fetchPromise);
    }

    return fetchPromise;
  }

  async checkHealth(): Promise<{ status: string }> {
    return this.request<{ status: string }>("/health", {}, HEALTH_CONFIG);
  }

  async getMenuCatalog(): Promise<MenuResponse> {
    const isDev = import.meta.env.DEV || import.meta.env["VITE_ASTRA_DEV_MODE"] === "true";
    try {
      return await this.request<MenuResponse>("/v1/menu", {}, MENU_CONFIG);
    } catch (error) {
      if (!isDev) throw error;
      console.warn("API not available, using mock data for development");
      const { mockMenuResponse } = await import("../routes/mockMenuData");
      return mockMenuResponse;
    }
  }

  async getCart(cartId: string): Promise<CartResponse> {
    return this.request<CartResponse>(`/v1/carts/${cartId}`, {}, READ_CONFIG);
  }

  async createCart(
    storeId: string,
    kioskId: string,
    sessionId: string,
    customerPhone?: string,
  ): Promise<CartResponse> {
    return this.request<CartResponse>("/v1/carts", {
      method: "POST",
      body: JSON.stringify({ storeId, kioskId, sessionId, customerPhone }),
    }, MUTATE_CONFIG);
  }

  async addItemToCart(
    cartId: string,
    menuItemId: string,
    _nameSnapshot: string,
    _unitPriceCentsSnapshot: number,
    quantity: number,
    modifiers: {
      modifierId: string;
      optionId: string;
      priceDeltaCents: number;
    }[] = [],
    _notes?: string,
    _weightGrams?: number,
  ): Promise<CartResponse> {
    const protoModifiers = modifiers.map((m) => ({
      modifierOptionId: m.modifierId,
      optionId: m.optionId,
      priceDeltaCentsSnapshot: m.priceDeltaCents,
    }));
    return this.request<CartResponse>(`/v1/carts/${cartId}/items`, {
      method: "POST",
      body: JSON.stringify({
        menuItemId,
        quantity,
        modifiers: protoModifiers,
      }),
    }, MUTATE_CONFIG);
  }

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
    }, MUTATE_CONFIG);
  }

  async checkoutCart(
    cartId: string,
    method: "credit_debit" | "nfc_apple_pay" | "nfc_google_pay" | "qr_code" | "cash_recycler",
    currency = "USD",
  ): Promise<{ checkoutId: string; paymentIntentId: string }> {
    return this.request<{ checkoutId: string; paymentIntentId: string }>(
      `/v1/carts/${cartId}/checkout`,
      { method: "POST", body: JSON.stringify({ method, currency }) },
      { timeoutMs: 30_000, deduplicate: false, cacheTtlMs: 0 },
    );
  }

  async processPayment(
    cartId: string,
    paymentIntentId: string,
    amountCents: number,
    currency = "USD",
    method: "credit_debit" | "nfc_apple_pay" | "nfc_google_pay" | "qr_code" | "cash_recycler",
  ): Promise<PaymentResult> {
    return this.request<PaymentResult>("/v1/payments", {
      method: "POST",
      body: JSON.stringify({ cartId, paymentIntentId, amountCents, currency, method }),
    }, { timeoutMs: 30_000, deduplicate: false, cacheTtlMs: 0 });
  }

  async getOrder(orderId: string): Promise<OrderResponse> {
    return this.request<OrderResponse>(`/v1/orders/${orderId}`, {}, READ_CONFIG);
  }

  async createOrder(
    cartId: string,
    paymentId: string,
  ): Promise<OrderResponse> {
    return this.request<OrderResponse>("/v1/orders", {
      method: "POST",
      body: JSON.stringify({ cartId, paymentId }),
    }, MUTATE_CONFIG);
  }

  async sendKioskHeartbeat(
    kioskId: string,
    storeId: string,
    syncStatus: "online" | "offline" | "degraded" | "maintenance" = "online",
    peerCount?: number,
    queueDepth?: number,
  ): Promise<KioskHeartbeatResponse> {
    return this.request<KioskHeartbeatResponse>(`/v1/kiosks/${kioskId}/heartbeat`, {
      method: "POST",
      body: JSON.stringify({ kioskId, storeId, syncStatus, peerCount, queueDepth }),
    }, { timeoutMs: 5_000, deduplicate: true, cacheTtlMs: 0 });
  }

  async getAnnouncements(): Promise<{ announcements: {
    id: string;
    title: string;
    message: string;
    severity: "info" | "warning" | "critical";
    expiresAt: string;
  }[] }> {
    return this.request<{ announcements: {
      id: string;
      title: string;
      message: string;
      severity: "info" | "warning" | "critical";
      expiresAt: string;
    }[] }>("/v1/announcements", {}, READ_CONFIG);
  }

  async adminLogin(
    username: string,
    password: string,
  ): Promise<{ token: string; expiresAt: string }> {
    return this.request<{ token: string; expiresAt: string }>("/v1/admin/auth/login", {
      method: "POST",
      body: JSON.stringify({ username, password }),
    }, MUTATE_CONFIG);
  }

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
    }>("/v1/admin/dashboard", {}, READ_CONFIG);
  }

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
    }, MUTATE_CONFIG);
  }

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
    }>("/v1/debug/info", {}, READ_CONFIG);
  }
}

export const apiClient = new AstraApiClient();

