import { uuidv7 } from "uuidv7";
import { z } from "zod";

/**
 * UUID v7 (RFC 9562) helpers. UUID v7 is time-sortable, which keeps Postgres
 * primary-key indexes insert-monotonic and makes event logs naturally ordered.
 */

const UUID_V7_REGEX =
  /^[0-9a-f]{8}-[0-9a-f]{4}-7[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/i;

/** Zod schema accepting a UUID v7 string. */
export const UuidSchema = z.string().regex(UUID_V7_REGEX, {
  message: "Expected UUID v7",
});

/**
 * Generates a new UUID v7 string using the current wall-clock time.
 */
export function generateId(): string {
  return uuidv7();
}

/**
 * Validates a raw string and returns it when it is a well-formed UUID v7.
 * Throws otherwise.
 */
export function idFromString(value: string): string {
  if (!isValidId(value)) {
    throw new Error(`Invalid UUID v7: "${String(value)}"`);
  }
  return value;
}

/**
 * Type guard returning true when the value is a string matching the UUID v7
 * variant (version nibble == 7, variant == 10xx).
 */
export function isValidId(value: unknown): value is string {
  return typeof value === "string" && UUID_V7_REGEX.test(value);
}

/**
 * Extracts the embedded millisecond timestamp from a UUID v7. Throws for
 * invalid identifiers.
 */
export function extractTimestampFromId(id: string): Date {
  if (!isValidId(id)) {
    throw new Error(`Invalid UUID v7: "${String(id)}"`);
  }
  const hex = id.replace(/-/g, "").slice(0, 12);
  const ms = Number.parseInt(hex, 16);
  return new Date(ms);
}
