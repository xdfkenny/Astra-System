import { useEffect } from "react";
import { SILENT_ASSIST_STALL_MS, useSessionStore } from "@astra/kiosk-state";

/**
 * Deep-improvement #4: "Silent Assist" mode.
 *
 * If a customer stalls for >45s on an active (non-attract, non-payment)
 * screen, we arm a subtle visual highlight on the next logical action
 * (handled declaratively by consuming components via `silentAssistArmed`)
 * rather than an interruptive modal/popup — interruptions measurably
 * increase abandonment in kiosk UX research, subtle affordance nudges do not.
 */
export function useSilentAssist(): void {
  useEffect(() => {
    const interval = window.setInterval(() => {
      const { stage, lastInteractionAtMs, silentAssistArmed, armSilentAssist } =
        useSessionStore.getState();
      const eligible = stage === "menu" || stage === "cart_review";
      const stalled = Date.now() - lastInteractionAtMs > SILENT_ASSIST_STALL_MS;
      if (eligible && stalled && !silentAssistArmed) {
        armSilentAssist(true);
      }
    }, 2000);
    return () => { window.clearInterval(interval); };
  }, []);
}
