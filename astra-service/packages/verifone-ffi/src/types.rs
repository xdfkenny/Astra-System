/**
 * TypeScript type declarations mirroring the safe Rust API exposed by
 * `astra-verifone-ffi`.
 *
 * These types are intended for frontend or Node.js consumers that interact
 * with the Verifone SDK through a TypeScript façade.
 */

/** Numeric status code returned by the Verifone SDK. */
export type VerifoneStatus = number;

/** Discriminated union of all Verifone error conditions. */
export type VerifoneError =
  | { kind: 'NotInitialized'; message: string }
  | { kind: 'InvalidParameter'; message: string }
  | { kind: 'Timeout'; message: string }
  | { kind: 'CardReadFailed'; message: string }
  | { kind: 'Processing'; message: string }
  | { kind: 'Network'; message: string }
  | { kind: 'Canceled'; message: string }
  | { kind: 'Closed'; message: string }
  | { kind: 'Unknown'; code: VerifoneStatus; message: string };

/** High-level Verifone SDK operations exposed to TypeScript callers. */
export interface VerifoneFFI {
  /** Initialise the terminal driver. */
  initTerminal(): Promise<void>;

  /**
   * Start a new transaction.
   * @param amount   Amount in the smallest currency unit (e.g. cents).
   * @param currency Three-letter ISO-4217 currency code (e.g. "USD").
   */
  startTransaction(amount: number, currency: string): Promise<void>;

  /** Wait for the cardholder to present a card. */
  waitForCard(): Promise<void>;

  /**
   * Process the active transaction and return the transaction identifier
   * assigned by the SDK.
   */
  processPayment(): Promise<string>;

  /** Refund a previously completed transaction. */
  refund(transactionId: string): Promise<void>;

  /** Close the terminal driver and release resources. */
  closeTerminal(): Promise<void>;
}
