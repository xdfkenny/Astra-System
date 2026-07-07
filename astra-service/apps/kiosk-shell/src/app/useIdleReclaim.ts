import { useEffect } from "react";
import { SESSION_IDLE_TIMEOUT_MS, useSessionStore } from "@astra/kiosk-state";

/**
 * Attract-loop reclaim: any active session with no touch interaction for
 * SESSION_IDLE_TIMEOUT_MS is forcibly ended and the kiosk returns to the
 * Attract screen. This prevents a walked-away customer from blocking the
 * lane indefinitely and is a PCI hygiene control (no cart/payment context
 * lingers on screen unattended).
 */
export function useIdleReclaim(): void {
  useEffect(() => {
    const interval = window.setInterval(() => {
      const { stage, lastInteractionAtMs, endSession } = useSessionStore.getState();
      if (stage === "attract" || stage === "payment_auth") {
        // Never auto-reclaim mid-payment — a stalled UI during card auth must
        // not silently abandon a transaction the terminal may still complete.
        return;
      }
      if (Date.now() - lastInteractionAtMs > SESSION_IDLE_TIMEOUT_MS) {
        endSession();
      }
    }, 1000);
    return () => { window.clearInterval(interval); };
  }, []);
}
