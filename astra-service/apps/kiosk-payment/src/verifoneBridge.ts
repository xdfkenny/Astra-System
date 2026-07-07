import type {
  OfflinePaymentToken,
  PaymentAuthorizationResult,
  PaymentMethod,
} from "@astra/shared-types";
import { uuidV7 } from "@astra/shared-types";

/**
 * Browser-side bridge to the local Verifone payment sidecar.
 *
 * The actual Verifone SDK is native C/C++ wrapped by our Rust FFI layer
 * (daemons/payment-ffi) and exposed as a loopback-only HTTP+WebSocket
 * sidecar process (`astra-payment-sidecar`, co-located on the kiosk, never
 * reachable off-device — enforced by binding to 127.0.0.1 and a Unix domain
 * socket fallback). The browser NEVER talks to the terminal directly: all
 * PCI-scoped communication happens in native code, and the browser only
 * exchanges opaque tokens/status over this bridge. This satisfies "payment
 * data never touches the kiosk [browser] storage."
 */
const SIDECAR_BASE_URL = "http://127.0.0.1:8963";

export interface InitiatePaymentRequest {
  readonly cartId: string;
  readonly amountCents: number;
  readonly method: PaymentMethod;
  readonly idempotencyKey: string; // UUIDv7, generated once per payment attempt
}

export async function initiatePayment(
  req: InitiatePaymentRequest,
): Promise<PaymentAuthorizationResult> {
  try {
    const res = await fetch(`${SIDECAR_BASE_URL}/v1/payments/initiate`, {
      method: "POST",
      headers: { "Content-Type": "application/json", "Idempotency-Key": req.idempotencyKey },
      body: JSON.stringify(req),
      signal: AbortSignal.timeout(30_000), // Verifone terminal interactions can legitimately take ~20s (tap/insert/PIN)
    });

    if (!res.ok) {
      throw new PaymentBridgeError(`Sidecar returned ${String(res.status)}`);
    }
    return (await res.json()) as PaymentAuthorizationResult;
  } catch (err) {
    // Sidecar unreachable (native process crashed, terminal cable unplugged,
    // or genuinely offline) — this is NOT a decline, it's a network failure.
    // The caller queues an offline token instead of telling the customer "declined".
    if (err instanceof PaymentBridgeError) throw err;
    throw new PaymentBridgeError("Verifone sidecar unreachable", { cause: err });
  }
}

export class PaymentBridgeError extends Error {
  constructor(message: string, options?: ErrorOptions) {
    super(message, options);
    this.name = "PaymentBridgeError";
  }
}

/**
 * Builds a locally-queued offline payment token when the sidecar is
 * reachable (terminal-side auth still happens natively) but the cloud
 * settlement path is unavailable. Signed with HMAC-SHA256 using the
 * per-kiosk sync key so a compromised kiosk cannot forge tokens that the
 * leader-election settlement pipeline would accept.
 */
export async function buildOfflineToken(params: {
  cartId: string;
  kioskId: string;
  amountCents: number;
  method: PaymentMethod;
  verifoneOpaqueToken: string;
  syncKey: CryptoKey;
}): Promise<OfflinePaymentToken> {
  const tokenId = uuidV7();
  const createdAtMs = Date.now();
  const expiresAtMs = createdAtMs + 48 * 60 * 60 * 1000; // matches the 48h offline SLA

  const canonical = `${tokenId}|${params.cartId}|${String(params.amountCents)}|${params.verifoneOpaqueToken}`;
  const signature = await hmacSha256Hex(params.syncKey, canonical);

  return {
    tokenId,
    kioskId: params.kioskId,
    cartId: params.cartId,
    amountCents: params.amountCents,
    currency: "USD",
    method: params.method,
    verifoneOpaqueToken: params.verifoneOpaqueToken,
    createdAtMs,
    expiresAtMs,
    hmacSignature: signature,
    synced: false,
  };
}

async function hmacSha256Hex(key: CryptoKey, message: string): Promise<string> {
  const encoder = new TextEncoder();
  const signatureBuffer = await crypto.subtle.sign("HMAC", key, encoder.encode(message));
  return Array.from(new Uint8Array(signatureBuffer))
    .map((b) => b.toString(16).padStart(2, "0"))
    .join("");
}
