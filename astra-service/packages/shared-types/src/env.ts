import { z } from "zod";

/**
 * Runtime environment schema shared by browser (via `window.__ENV__`) and Node
 * (via `process.env`). Every variable is documented below; callers should
 * import the typed `env` object after ensuring the environment is parsed.
 */

export const envSchema = z.object({
  /**
   * Node execution environment. Controls logging verbosity, stack traces, and
   * safe defaults for unset variables.
   */
  NODE_ENV: z.enum(["development", "production", "test"]).default("development"),

  /**
   * Logical Astra deployment environment (e.g. "kiosk-edge", "staging",
   * "production"). Used for telemetry resource attributes and feature flags.
   */
  ASTRA_ENVIRONMENT: z.string().min(1).default("development"),

  /**
   * Service name reported to OpenTelemetry and structured logs.
   */
  ASTRA_SERVICE_NAME: z.string().min(1).default("astra-service"),

  /**
   * Base URL for the Astra REST API. Required in production; defaults to the
   * local development gateway otherwise.
   */
  ASTRA_API_URL: z.string().url().default("http://localhost:3000"),

  /**
   * Base URL for the Astra WebSocket/sync gateway. Required in production;
   * defaults to the local development gateway otherwise.
   */
  ASTRA_WEBSOCKET_URL: z.string().url().default("ws://localhost:3001"),

  /**
   * OpenTelemetry collector endpoint (e.g.
   * "https://otel.astra-service.internal/v1/traces"). Optional; when omitted
   * telemetry is sampled to zero and spans are discarded.
   */
  ASTRA_OTLP_ENDPOINT: z.string().url().optional(),

  /**
   * UUID v7 identifier of the current kiosk. Required in kiosk-edge runtime;
   * optional for cloud services.
   */
  ASTRA_KIOSK_ID: z.string().uuid().optional(),

  /**
   * UUID v7 identifier of the store the kiosk belongs to.
   */
  ASTRA_STORE_ID: z.string().uuid().optional(),

  /**
   * UUID v7 identifier of the lane/queue the kiosk serves.
   */
  ASTRA_LANE_ID: z.string().uuid().optional(),

  /**
   * UUID v7 identifier of the tenant. Used for multi-tenant routing and
   * telemetry.
   */
  ASTRA_TENANT_ID: z.string().uuid().optional(),

  /**
   * Offline payment token TTL in hours. Must be between 1 and 72 hours. The
   * 48-hour default matches the offline-resilience target in ARCHITECTURE.md.
   */
  ASTRA_OFFLINE_TOKEN_TTL_HOURS: z.coerce.number().int().min(1).max(72).default(48),

  /**
   * ISO 4217 currency code (e.g. "USD") used as the default when a store
   * setting is unavailable.
   */
  ASTRA_CURRENCY: z.string().length(3).default("USD"),

  /**
   * Ratio between 0 and 1 for head-based trace sampling in the browser
   * telemetry SDK. Defaults to 1 (sample all) in development and 0.1 in
   * production.
   */
  ASTRA_TRACE_SAMPLE_RATIO: z.coerce.number().min(0).max(1).default(1),
});

export type Env = z.infer<typeof envSchema>;

interface BrowserEnvWindow {
  readonly __ENV__?: Record<string, unknown>;
}

function readBrowserEnv(): Record<string, unknown> | undefined {
  if (typeof window === "undefined") return undefined;
  const win = window as unknown as BrowserEnvWindow;
  return win.__ENV__;
}

function readNodeEnv(): Record<string, unknown> {
  const globalProcess = (globalThis as Record<string, unknown>)["process"];
  if (
    typeof globalProcess !== "object" ||
    globalProcess === null ||
    !("env" in globalProcess)
  ) {
    return {};
  }
  return (globalProcess as Record<string, unknown>)["env"] as Record<string, unknown>;
}

/**
 * Parses a raw environment object against the Astra environment schema.
 * Browser-injected `window.__ENV__` values take precedence over Node-style
 * `process.env` values, allowing runtime configuration of built static assets.
 */
export function parseEnv(raw: Record<string, unknown>): Env {
  const browserEnv = readBrowserEnv();
  const merged: Record<string, unknown> = { ...readNodeEnv(), ...browserEnv, ...raw };
  return envSchema.parse(merged);
}

/**
 * Typed environment object for the current runtime. Throws at import time if
 * required variables are missing or invalid.
 */
export const env: Env = parseEnv({});
