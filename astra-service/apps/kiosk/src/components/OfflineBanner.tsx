import { useEffect, useState } from "react";
import { motion, AnimatePresence } from "framer-motion";
import { useSessionStore } from "@astra/kiosk-state";

export interface OfflineBannerProps {
  /** For testing override — reads from session store by default */
  onlineOverride?: boolean;
}

export function OfflineBanner({ onlineOverride }: OfflineBannerProps) {
  const online = onlineOverride ?? useSessionStore((s) => s.network.online);
  const [dismissed, setDismissed] = useState(false);
  const [canDismiss, setCanDismiss] = useState(false);

  useEffect(() => {
    if (!online) {
      setDismissed(false);
      setCanDismiss(false);
      const timer = setTimeout(() => {
        setCanDismiss(true);
      }, 3000);
      return () => clearTimeout(timer);
    }
    return undefined;
  }, [online]);

  const show = !online && !dismissed;

  return (
    <AnimatePresence>
      {show && (
        <motion.div
          key="offline-banner"
          initial={{ height: 0, opacity: 0 }}
          animate={{ height: canDismiss ? 40 : 40, opacity: 1 }}
          exit={{ height: 0, opacity: 0 }}
          transition={{ duration: 0.25, ease: [0.16, 1, 0.3, 1] }}
          className="relative z-40 flex w-full items-center justify-center gap-2 overflow-hidden bg-pale-mint px-3 border-b border-moss/20"
          role="alert"
          aria-live="assertive"
        >
          <svg
            width="16"
            height="16"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            strokeWidth="2"
            strokeLinecap="round"
            strokeLinejoin="round"
            className="shrink-0 text-moss"
            aria-hidden="true"
          >
            <path d="M22.61 16.95A5 5 0 0 0 18 10h-1.26a8 8 0 0 0-7.05-6M5 5a8 8 0 0 0 4 15h9a5 5 0 0 0 1.7-.3" />
            <line x1="1" y1="1" x2="23" y2="23" />
          </svg>
          <span className="font-ui text-[14px] text-moss">
            Working offline. Your cart is secure.
          </span>
          {canDismiss && (
            <button
              type="button"
              onClick={() => setDismissed(true)}
              className="ml-2 flex h-6 w-6 items-center justify-center rounded-full text-moss/60 hover:text-moss focus-visible:ring-2 focus-visible:ring-moss focus-visible:ring-offset-2"
              aria-label="Dismiss offline banner"
            >
              <svg
                width="14"
                height="14"
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                strokeWidth="2"
                strokeLinecap="round"
                strokeLinejoin="round"
                aria-hidden="true"
              >
                <line x1="18" y1="6" x2="6" y2="18" />
                <line x1="6" y1="6" x2="18" y2="18" />
              </svg>
            </button>
          )}
        </motion.div>
      )}
    </AnimatePresence>
  );
}
