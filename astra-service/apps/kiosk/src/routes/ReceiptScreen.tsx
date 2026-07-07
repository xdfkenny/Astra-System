import { useEffect } from "react";
import { motion } from "framer-motion";
import { PrimaryButton } from "@astra/ui-kit";
import { motion as motionTokens } from "@astra/design-tokens";
import { useKioskMachine } from "../machines/KioskMachineProvider";

const AUTO_RETURN_TO_ATTRACT_MS = 8_000;

/**
 * Receipt — terminal state of a successful transaction. Auto-returns to Attract
 * after a short dwell so the lane frees up for the next customer.
 */
export function ReceiptScreen(): React.JSX.Element {
  const { send } = useKioskMachine();

  useEffect(() => {
    const timeout = window.setTimeout(() => {
      send({ type: "RECEIPT_ACKNOWLEDGED" });
    }, AUTO_RETURN_TO_ATTRACT_MS);
    return () => {
      window.clearTimeout(timeout);
    };
  }, [send]);

  return (
    <div className="flex flex-1 flex-col items-center justify-center gap-6 bg-surface p-6 text-center">
      <motion.div
        initial={{ scale: 0.8, opacity: 0 }}
        animate={{ scale: 1, opacity: 1 }}
        transition={{
          duration: motionTokens.durationSlow,
          ease: motionTokens.easeEmphasized,
        }}
        className="flex h-24 w-24 items-center justify-center rounded-full bg-success text-white"
        aria-hidden="true"
      >
        <svg viewBox="0 0 24 24" fill="none" className="h-12 w-12" stroke="currentColor" strokeWidth={3}>
          <path d="M5 13l4 4L19 7" strokeLinecap="round" strokeLinejoin="round" />
        </svg>
      </motion.div>
      <h2 className="font-heading text-3xl font-bold text-ink">Thank you!</h2>
      <p className="text-lg text-ink-muted">Your receipt is printing now.</p>
      <PrimaryButton
        variant="ghost"
        onClick={() => {
          send({ type: "RECEIPT_ACKNOWLEDGED" });
        }}
        aria-label="Start new order"
      >
        Start New Order
      </PrimaryButton>
    </div>
  );
}
