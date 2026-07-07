import { describe, expect, it } from "vitest";
import { generateId, idFromString, isValidId, extractTimestampFromId } from "../ids";

describe("ids", () => {
  it("generates valid UUID v7 strings", () => {
    const id = generateId();
    expect(isValidId(id)).toBe(true);
    expect(id).toMatch(/^[0-9a-f]{8}-[0-9a-f]{4}-7[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/i);
  });

  it("generates unique identifiers", () => {
    const ids = new Set(Array.from({ length: 100 }, generateId));
    expect(ids.size).toBe(100);
  });

  it("idFromString returns valid ids", () => {
    const id = generateId();
    expect(idFromString(id)).toBe(id);
  });

  it("idFromString throws for invalid ids", () => {
    expect(() => idFromString("not-a-uuid")).toThrow();
  });

  it("rejects v4 UUIDs", () => {
    const v4 = "550e8400-e29b-41d4-a716-446655440000";
    expect(isValidId(v4)).toBe(false);
  });

  it("rejects non-string values", () => {
    expect(isValidId(null)).toBe(false);
    expect(isValidId(123)).toBe(false);
    expect(isValidId({})).toBe(false);
  });

  it("extracts the embedded timestamp", () => {
    const now = Date.now();
    const id = generateId();
    const extracted = extractTimestampFromId(id);
    expect(extracted.getTime()).toBeLessThanOrEqual(now + 1000);
    expect(extracted.getTime()).toBeGreaterThanOrEqual(now - 1000);
  });
});
