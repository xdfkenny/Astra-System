import { describe, expect, it } from "vitest";
import { envSchema, parseEnv } from "../env";

describe("env schema", () => {
  it("parses a minimal valid environment", () => {
    const env = parseEnv({});
    expect(["development", "production", "test"]).toContain(env.NODE_ENV);
    expect(env.ASTRA_API_URL).toBe("http://localhost:3000");
    expect(env.ASTRA_WEBSOCKET_URL).toBe("ws://localhost:3001");
    expect(env.ASTRA_OFFLINE_TOKEN_TTL_HOURS).toBe(48);
    expect(env.ASTRA_CURRENCY).toBe("USD");
  });

  it("falls back to the default NODE_ENV when not present", () => {
    const env = parseEnv({ NODE_ENV: undefined });
    expect(env.NODE_ENV).toBe("development");
  });

  it("overrides defaults with provided values", () => {
    const env = parseEnv({
      NODE_ENV: "production",
      ASTRA_API_URL: "https://api.astra.internal",
      ASTRA_WEBSOCKET_URL: "wss://sync.astra.internal",
      ASTRA_SERVICE_NAME: "kiosk-edge",
      ASTRA_CURRENCY: "CAD",
    });
    expect(env.NODE_ENV).toBe("production");
    expect(env.ASTRA_API_URL).toBe("https://api.astra.internal");
    expect(env.ASTRA_WEBSOCKET_URL).toBe("wss://sync.astra.internal");
    expect(env.ASTRA_SERVICE_NAME).toBe("kiosk-edge");
    expect(env.ASTRA_CURRENCY).toBe("CAD");
  });

  it("coerces numeric values from strings", () => {
    const env = parseEnv({
      ASTRA_OFFLINE_TOKEN_TTL_HOURS: "24",
      ASTRA_TRACE_SAMPLE_RATIO: "0.5",
    });
    expect(env.ASTRA_OFFLINE_TOKEN_TTL_HOURS).toBe(24);
    expect(env.ASTRA_TRACE_SAMPLE_RATIO).toBe(0.5);
  });

  it("rejects an invalid URL", () => {
    expect(() => parseEnv({ ASTRA_API_URL: "not-a-url" })).toThrow();
  });

  it("rejects an offline token TTL outside the allowed range", () => {
    expect(() => parseEnv({ ASTRA_OFFLINE_TOKEN_TTL_HOURS: 96 })).toThrow();
    expect(() => parseEnv({ ASTRA_OFFLINE_TOKEN_TTL_HOURS: 0 })).toThrow();
  });

  it("rejects an invalid UUID", () => {
    expect(() => parseEnv({ ASTRA_KIOSK_ID: "not-a-uuid" })).toThrow();
  });

  it("validates the schema independently", () => {
    const result = envSchema.safeParse({
      NODE_ENV: "test",
      ASTRA_ENVIRONMENT: "test",
      ASTRA_API_URL: "http://localhost:3000",
    });
    expect(result.success).toBe(true);
  });
});
