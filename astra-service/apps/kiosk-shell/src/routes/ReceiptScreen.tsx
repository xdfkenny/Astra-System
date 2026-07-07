import { useEffect } from "react";
import { motion } from "framer-motion";
import { motion as motionTokens } from "@astra/design-tokens";
import { useSessionStore } from "@astra/kiosk-state";

const AUTO_RETURN_TO_ATTRACT_MS = 12_000;

/**
 * Receipt — terminal state of a successful transaction. Auto-returns to
 * Attract after a short dwell so the lane frees up for the next customer
 * without requiring an extra tap (kiosks in high-throughput retail can't
 * rely on customers remembering to "finish" a flow).
 */
export function ReceiptScreen(): React.JSX.Element {
  const endSession = useSessionStore((s) => s.endSession);

  useEffect(() => {
    const timeout = window.setTimeout(() => { endSession(); }, AUTO_RETURN_TO_ATTRACT_MS);
    return () => { window.clearTimeout(timeout); };
  }, [endSession]);

  return (
    <div className="flex flex-1 flex-col items-center justify-center gap-6 bg-surface p-6 text-center">
      <motion.div
        initial={{ scale: 0.8, opacity: 0 }}
        animate={{ scale: 1, opacity: 1 }}
        transition={{ duration: motionTokens.durationSlow, ease: motionTokens.easeEmphasized }}
        className="flex h-24 w-24 items-center justify-center rounded-full bg-success text-white"
        aria-hidden="true"
      >
        <svg viewBox="0 0 24 24" fill="none" className="h-12 w-12" stroke="currentColor" strokeWidth={3}>
          <path d="M5 13l4 4L19 7" strokeLinecap="round" strokeLinejoin="round" />
        </svg>
      </motion.div>
      <h2 className="font-heading text-3xl font-bold text-ink">Thank you!</h2>
      <p className="text-lg text-ink-muted">Your receipt is printing now.</p>
      <button
        type="button"
        onClick={() => { endSession(); }}
        className="mt-8 h-[var(--touch-comfortable)] rounded-lg border border-border-strong px-8 font-medium text-ink"
      >
        Start New Order
      </button>
    </div>
  );
}
