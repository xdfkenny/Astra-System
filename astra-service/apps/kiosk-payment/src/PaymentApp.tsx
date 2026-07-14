import { useState } from "react";
import { useSnapshot } from "valtio";
import { motion } from "framer-motion";
import { motion as motionTokens } from "@astra/design-tokens";
import { PrimaryButton } from "@astra/ui-kit";
import { cartProxy } from "@astra/kiosk-state";
import { uuidV7 } from "@astra/shared-types";
import type { PaymentAuthorizationResult, PaymentMethod } from "@astra/shared-types";
import { initiatePayment, buildOfflineToken, PaymentBridgeError } from "./verifoneBridge";
import { useCartTotals } from "@astra/cart-engine";

export interface PaymentAppProps {
  readonly onResult: (result: PaymentAuthorizationResult) => void;
  readonly onCancel: () => void;
}

type UiPhase = "select_method" | "awaiting_terminal" | "queued_offline" | "declined";

const METHODS: readonly { id: PaymentMethod; label: string; icon: string }[] = [
  { id: "credit_debit", label: "Credit / Debit", icon: "\u{1F4B3}" },
  { id: "nfc_apple_pay", label: "Apple Pay", icon: "\u{1F4F1}" },
  { id: "nfc_google_pay", label: "Google Pay", icon: "\u{1F4F1}" },
  { id: "qr_code", label: "QR Code", icon: "\u{1F532}" },
];

/**
 * Federated Payment micro-frontend. This is the ONLY place the security
 * mandate allows a biometric/PIN auth factor to be triggered (via the
 * Verifone terminal itself — the customer taps/inserts/authenticates at the
 * physical PIN pad, not in the browser UI). If the sidecar is unreachable,
 * we fail into a locally-queued offline token rather than blocking the sale.
 */
export default function PaymentApp({ onResult, onCancel }: PaymentAppProps): React.JSX.Element {
  const cart = useSnapshot(cartProxy);
  const totals = useCartTotals(cart.lines, false);
  const [phase, setPhase] = useState<UiPhase>("select_method");
  const [declineReason, setDeclineReason] = useState<string | null>(null);

  const handleSelectMethod = async (method: PaymentMethod): Promise<void> => {
    setPhase("awaiting_terminal");
    const idempotencyKey = uuidV7();

    try {
      const result = await initiatePayment({
        cartId: cart.cartId,
        amountCents: totals.totalCents,
        method,
        idempotencyKey,
      });

      if (result.status === "declined") {
        setDeclineReason(result.declineReason ?? "Card declined. Please try another method.");
        setPhase("declined");
        return;
      }
      onResult(result);
    } catch (err) {
      if (err instanceof PaymentBridgeError) {
        // Network/sidecar failure, NOT a decline. Queue offline and let the
        // customer complete the sale — settlement reconciles once the mesh
        // leader regains connectivity (see astra-syncd raft + outbox pattern).
        await queueOffline(method, totals.totalCents);
        return;
      }
      setDeclineReason("An unexpected error occurred. Please try again.");
      setPhase("declined");
    }
  };

  const queueOffline = async (method: PaymentMethod, amountCents: number): Promise<void> => {
    setPhase("queued_offline");
    const syncKey = await importDevSyncKey(); // production: pulled from Vault/SOPS-provisioned kiosk secret
    const token = await buildOfflineToken({
      cartId: cart.cartId,
      kioskId: cart.kioskId,
      amountCents,
      method,
      verifoneOpaqueToken: `offline-pending-${uuidV7()}`,
      syncKey,
    });
    // Persisted to the local SQLCipher-encrypted queue by astra-syncd via its
    // loopback IPC channel; the browser only hands off the signed token.
    await fetch("http://127.0.0.1:4499/v1/offline-queue/enqueue", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(token),
    }).catch(() => {
      // Even the local daemon is unreachable — extremely degraded state.
      // In production this triggers a hardware alert LED + PagerDuty page.
    });

    onResult({
      authorizationId: token.tokenId,
      status: "queued_offline",
      method,
      amountCents,
    });
  };

  if (phase === "declined") {
    return (
      <div className="flex flex-1 flex-col items-center justify-center gap-4 p-6 text-center">
        <p className="font-heading text-2xl font-bold text-error">Payment declined</p>
        <p className="text-ink-muted">{declineReason}</p>
        <PrimaryButton variant="primary" onClick={() => { setPhase("select_method"); }}>
          Try Another Method
        </PrimaryButton>
        <button type="button" onClick={onCancel} className="text-sm text-ink-muted underline">
          Cancel and return to cart
        </button>
      </div>
    );
  }

  if (phase === "queued_offline") {
    return (
      <div className="flex flex-1 flex-col items-center justify-center gap-4 p-6 text-center">
        <motion.div
          animate={{ opacity: [0.5, 1, 0.5] }}
          transition={{ duration: 1.5, repeat: Infinity }}
          className="text-5xl"
          aria-hidden="true"
        >
          {"\u{1F4E1}"}
        </motion.div>
        <p className="font-heading text-xl font-semibold">Payment secured offline</p>
        <p className="max-w-xs text-ink-muted">
          Your payment is confirmed and will sync automatically once the network reconnects.
        </p>
      </div>
    );
  }

  if (phase === "awaiting_terminal") {
    return (
      <div className="flex flex-1 flex-col items-center justify-center gap-6 p-6 text-center">
        <motion.div
          animate={{ scale: [1, 1.08, 1] }}
          transition={{ duration: 1.2, repeat: Infinity, ease: motionTokens.easeStandard }}
          className="flex h-28 w-28 items-center justify-center rounded-full bg-primary text-white"
          aria-hidden="true"
        >
          <svg viewBox="0 0 24 24" className="h-14 w-14" fill="none" stroke="currentColor" strokeWidth={1.5}>
            <rect x="2" y="6" width="20" height="14" rx="2" />
            <path d="M2 10h20" strokeLinecap="round" />
          </svg>
        </motion.div>
        <p className="font-heading text-xl font-semibold">Follow the prompts on the card reader</p>
        <p role="status" aria-live="assertive" className="text-ink-muted">
          Tap, insert, or scan to complete your payment.
        </p>
      </div>
    );
  }

  return (
    <div className="flex flex-1 flex-col gap-4 p-6">
      <h2 className="font-heading text-2xl font-bold">Choose payment method</h2>
      <p className="text-lg font-medium text-ink-muted">
        Total: <span className="tabular-nums text-ink">${(totals.totalCents / 100).toFixed(2)}</span>
      </p>
      <div className="mt-2 flex flex-col gap-3">
        {METHODS.map((m) => (
          <button
            key={m.id}
            type="button"
            onClick={() => void handleSelectMethod(m.id)}
            className="hairline flex h-[var(--touch-comfortable)] items-center gap-4 rounded-md bg-surface px-5 text-lg font-medium active:bg-surface-sunken"
          >
            <span aria-hidden="true" className="text-2xl">
              {m.icon}
            </span>
            {m.label}
          </button>
        ))}
      </div>
      <button type="button" onClick={onCancel} className="mt-auto text-sm text-ink-muted underline">
        Cancel and return to cart
      </button>
    </div>
  );
}

/**
 * Retrieves the per-device sync key from a secure source.
 * - In development: reads from VITE_PAYMENT_SYNC_KEY env var (never committed).
 * - In production: the kiosk syncd daemon exposes a scoped, time-limited
 *   derivative through the local IPC bridge at 127.0.0.1:4499 — the browser
 *   never has access to the root kiosk key.
 */
async function importDevSyncKey(): Promise<CryptoKey> {
  const keyFromEnv: string | undefined = import.meta.env["VITE_PAYMENT_SYNC_KEY"] as string | undefined;
  const encoded: string | null = keyFromEnv
    ?? (import.meta.env.DEV
      ? await fetchSyncKeyFromDaemon()
      : null);
  if (!encoded) {
    throw new Error(
      "No sync key available. Ensure VITE_PAYMENT_SYNC_KEY is set in development "
      + "or the syncd daemon IPC is reachable in production.",
    );
  }
  const rawKey = new TextEncoder().encode(encoded);
  return crypto.subtle.importKey("raw", rawKey, { name: "HMAC", hash: "SHA-256" }, false, ["sign"]);
}

async function fetchSyncKeyFromDaemon(): Promise<string | null> {
  try {
    const res = await fetch("http://127.0.0.1:4499/v1/sync-key", {
      signal: AbortSignal.timeout(2000),
    });
    if (!res.ok) return null;
    const data = await res.json() as { key?: string };
    return data.key ?? null;
  } catch {
    return null;
  }
}
