import { useEffect, useRef } from "react";
import { useKioskMachine } from "../machines/KioskMachineProvider";

/** Time with no interaction before the kiosk reclaims an active session. */
export const IDLE_TIMEOUT_MS = 90_000;

const IMMUNE_STAGES = new Set<string>([
  "ATTRACT",
  "IDLE_TIMEOUT",
  "PAYMENT_AUTH",
  "PROCESSING",
  "RECEIPT",
  "RESET",
]);

/**
 * Attract-loop reclaim: any active session with no touch/keyboard interaction
 * for IDLE_TIMEOUT_MS is ended and the kiosk returns to the Attract screen.
 * Payment phases are immune so a stalled UI mid-auth is never silently abandoned.
 */
export function useIdleReclaim(): void {
  const { state, send } = useKioskMachine();
  const lastInteractionAtMs = useRef(Date.now());

  useEffect(() => {
    const recordInteraction = (): void => {
      lastInteractionAtMs.current = Date.now();
    };

    window.addEventListener("pointerdown", recordInteraction);
    window.addEventListener("keydown", recordInteraction);

    const interval = window.setInterval(() => {
      const stage = state.value as string;
      if (IMMUNE_STAGES.has(stage)) return;
      if (Date.now() - lastInteractionAtMs.current > IDLE_TIMEOUT_MS) {
        send({ type: "IDLE_TIMEOUT" });
      }
    }, 1000);

    return () => {
      window.removeEventListener("pointerdown", recordInteraction);
      window.removeEventListener("keydown", recordInteraction);
      window.clearInterval(interval);
    };
  }, [state.value, send]);
}
