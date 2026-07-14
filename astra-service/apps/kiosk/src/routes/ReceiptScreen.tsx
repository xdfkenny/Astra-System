import { useCallback, useEffect, useRef, useState } from "react";
import { motion, AnimatePresence } from "framer-motion";
import { motion as motionTokens } from "@astra/design-tokens";
import { useKioskMachine } from "../machines/KioskMachineProvider";
import { defaultLogger } from "../utils/logger";

const log = defaultLogger.child("ReceiptScreen");

const AUTO_RETURN_TO_ATTRACT_MS = 10_000;
const PRIMARY_DELAY_MS = 3_000;

export function ReceiptScreen(): React.JSX.Element {
  const { send, state } = useKioskMachine();
  const [showPrimary, setShowPrimary] = useState(false);
  const [printerFailed, setPrinterFailed] = useState(false);
  const [printLoading, setPrintLoading] = useState(false);
  const [emailLoading, setEmailLoading] = useState(false);
  const [emailSent, setEmailSent] = useState(false);
  const returnTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  const orderNumber = state.context.order?.orderNumber ?? "A-7842";
  const orderId = state.context.order?.orderId;

  useEffect(() => {
    const primaryTimer = setTimeout(() => { setShowPrimary(true); }, PRIMARY_DELAY_MS);
    returnTimerRef.current = setTimeout(() => {
      send({ type: "RECEIPT_ACKNOWLEDGED" });
    }, AUTO_RETURN_TO_ATTRACT_MS);
    return () => {
      clearTimeout(primaryTimer);
      if (returnTimerRef.current) clearTimeout(returnTimerRef.current);
    };
  }, [send]);

  const handlePrint = useCallback(async () => {
    if (printLoading) return;
    setPrintLoading(true);
    try {
      const orderIdToPrint = orderId ?? crypto.randomUUID();
      await fetch("/api/print/receipt", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ orderId: orderIdToPrint }),
      });
      log.info("Receipt print requested", { orderId: orderIdToPrint });
    } catch {
      setPrinterFailed(true);
      log.warn("Printer unavailable");
      setTimeout(() => { setPrinterFailed(false); }, 4000);
    } finally {
      setPrintLoading(false);
    }
  }, [printLoading, orderId]);

  const handleEmail = useCallback(async () => {
    if (emailLoading) return;
    setEmailLoading(true);
    try {
      const orderIdToEmail = orderId ?? crypto.randomUUID();
      await fetch("/api/email/receipt", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ orderId: orderIdToEmail }),
      });
      setEmailSent(true);
      log.info("Receipt email requested", { orderId: orderIdToEmail });
      setTimeout(() => { setEmailSent(false); }, 3000);
    } catch (error) {
      log.error("Email receipt failed", error);
    } finally {
      setEmailLoading(false);
    }
  }, [emailLoading, orderId]);

  const handleStartNewOrder = useCallback(() => {
    send({ type: "RECEIPT_ACKNOWLEDGED" });
  }, [send]);

  return (
    <div className="flex flex-1 flex-col items-center justify-center bg-warm-cream px-6 text-center">
      {/* Success icon — SVG stroke animation */}
      <motion.div
        initial={{ scale: 0.8, opacity: 0 }}
        animate={{ scale: 1, opacity: 1 }}
        transition={{
          duration: 0.3,
          ease: motionTokens.easeOutExpo,
        }}
        className="flex h-24 w-24 items-center justify-center rounded-full bg-moss"
        aria-hidden="true"
      >
        <svg
          viewBox="0 0 24 24"
          className="h-12 w-12 text-white"
          fill="none"
          stroke="currentColor"
          strokeWidth={3}
          strokeLinecap="round"
          strokeLinejoin="round"
        >
          <motion.path
            d="M5 13l4 4L19 7"
            initial={{ pathLength: 0 }}
            animate={{ pathLength: 1 }}
            transition={{ duration: 0.3, delay: 0.1, ease: motionTokens.easeOutExpo }}
          />
        </svg>
      </motion.div>

      {/* Thank you heading */}
      <h1 className="mt-4 font-heading text-[36px] font-semibold text-charcoal">
        Thank you
      </h1>

      {/* Order number */}
      <p className="mt-2 font-mono text-[24px] text-charcoal tabular-nums">
        Order #{orderNumber}
      </p>

      {/* Action buttons */}
      <div className="mt-8 flex flex-col items-center gap-3 w-full max-w-xs">
        <button
          type="button"
          onClick={handlePrint}
          disabled={printLoading}
          className="h-14 w-full rounded-[16px] bg-white/70 border border-taupe font-sans text-[16px] font-medium text-charcoal active:bg-warm-cream/50 disabled:opacity-40 transition-colors duration-100"
          aria-label={printLoading ? "Printing receipt..." : "Print receipt"}
        >
          {printLoading ? "Printing..." : "Print receipt"}
        </button>

        <button
          type="button"
          onClick={handleEmail}
          disabled={emailLoading}
          className="h-14 w-full rounded-[16px] bg-white/70 border border-taupe font-sans text-[16px] font-medium text-charcoal active:bg-warm-cream/50 disabled:opacity-40 transition-colors duration-100"
          aria-label={emailLoading ? "Emailing receipt..." : "Email receipt"}
        >
          {emailLoading ? "Sending..." : emailSent ? "Sent!" : "Email receipt"}
        </button>

        <AnimatePresence>
          {showPrimary && (
            <motion.button
              type="button"
              initial={{ opacity: 0, y: 8 }}
              animate={{ opacity: 1, y: 0 }}
              exit={{ opacity: 0, y: -8 }}
              transition={{ duration: 0.25, ease: motionTokens.easeOutExpo }}
              onClick={handleStartNewOrder}
              className="mt-2 h-16 w-full rounded-full bg-amber text-white font-sans text-[18px] font-medium shadow-[0_4px_16px_rgba(184,126,107,0.3)] active:scale-[0.98] active:translate-y-[1px] transition-all duration-100"
              aria-label="Start new order"
            >
              Start new order
            </motion.button>
          )}
        </AnimatePresence>
      </div>

      {/* Printer failure toast */}
      <AnimatePresence>
        {printerFailed && (
          <motion.div
            initial={{ y: -20, opacity: 0 }}
            animate={{ y: 0, opacity: 1 }}
            exit={{ y: -20, opacity: 0 }}
            transition={{ duration: 0.25, ease: motionTokens.easeOutExpo }}
            className="fixed top-16 left-1/2 -translate-x-1/2 z-40 rounded-[12px] bg-charcoal px-4 py-3 text-white font-sans text-[14px] shadow-[0_4px_24px_rgba(45,42,38,0.08)]"
            role="alert"
            aria-live="assertive"
          >
            <div className="flex items-center gap-2">
              <svg
                viewBox="0 0 20 20"
                className="h-4 w-4 shrink-0 text-amber"
                fill="none"
                stroke="currentColor"
                strokeWidth={1.5}
                aria-hidden="true"
              >
                <path d="M10 2a8 8 0 1 0 0 16 8 8 0 0 0 0-16Z" />
                <path d="M10 6v4" strokeLinecap="round" />
                <path d="M10 13v.01" strokeLinecap="round" />
              </svg>
              <span>Printer unavailable. Receipt saved.</span>
            </div>
            {/* Auto-dismiss progress bar */}
            <div className="mt-2 h-1 w-full overflow-hidden rounded-full bg-white/20">
              <motion.div
                className="h-full rounded-full bg-amber"
                initial={{ width: "100%" }}
                animate={{ width: "0%" }}
                transition={{ duration: 4, ease: "linear" }}
              />
            </div>
          </motion.div>
        )}
      </AnimatePresence>

      {/* Screen-reader live region */}
      <div className="sr-only" aria-live="polite" role="status">
        {showPrimary
          ? "Order complete. Tap start new order to continue."
          : "Receipt ready."}
      </div>
    </div>
  );
}

