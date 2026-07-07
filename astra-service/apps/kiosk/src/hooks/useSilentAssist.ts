import { useEffect } from "react";
import { useSessionStore } from "@astra/kiosk-state";
import { useKioskMachine } from "../machines/KioskMachineProvider";

/** Time stalled on an active screen before the next logical action is highlighted. */
export const SILENT_ASSIST_STALL_MS = 45_000;

const ELIGIBLE_STAGES = new Set<string>([
  "MENU_BROWSE",
  "CART_REVIEW",
]);

/**
 * Silent Assist: if a customer stalls on menu or cart, arm a subtle highlight
 * on the next logical action. Interruptions increase abandonment; subtle
 * affordance nudges do not.
 */
export function useSilentAssist(): void {
  const { state } = useKioskMachine();

  useEffect(() => {
    const armSilentAssist = useSessionStore.getState().armSilentAssist;
    const recordInteraction = useSessionStore.getState().recordInteraction;

    const interval = window.setInterval(() => {
      const store = useSessionStore.getState();
      const eligible = ELIGIBLE_STAGES.has(state.value as string);
      const stalled = Date.now() - store.lastInteractionAtMs > SILENT_ASSIST_STALL_MS;
      if (eligible && stalled && !store.silentAssistArmed) {
        armSilentAssist(true);
      }
    }, 2000);

    return () => {
      window.clearInterval(interval);
      recordInteraction();
    };
  }, [state.value]);
}
