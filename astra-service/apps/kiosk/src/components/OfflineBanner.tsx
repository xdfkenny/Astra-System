import { useEffect, useState } from "react";
import { motion, AnimatePresence } from "framer-motion";
import { useKioskMachine } from "../machines/KioskMachineProvider";

export function OfflineBanner() {
  const { state } = useKioskMachine();
  const [showBanner, setShowBanner] = useState(false);

  const isOffline = state.context.isOfflineMode || state.context.apiStatus === "offline";

  useEffect(() => {
    if (isOffline) {
      setShowBanner(true);
      const timer = setTimeout(() => {
        setShowBanner(false);
      }, 5000);
      return () => { clearTimeout(timer); };
    } else {
      setShowBanner(false);
       return () => { /* empty cleanup */ };
    }
  }, [isOffline]);

  if (!showBanner) return null;

  return (
    <AnimatePresence>
      {isOffline && (
        <motion.div
          initial={{ y: -60, opacity: 0 }}
          animate={{ y: 0, opacity: 1 }}
          exit={{ y: -60, opacity: 0 }}
          transition={{ duration: 0.3, ease: "easeInOut" }}
          className="fixed top-0 left-0 right-0 z-40 bg-pale-mint border-b border-moss/20 px-4 py-3"
          role="alert"
          aria-live="assertive"
        >
          <div className="flex items-center justify-center gap-2">
            <svg
              viewBox="0 0 20 20"
              className="h-5 w-5 shrink-0 text-moss"
              fill="none"
              stroke="currentColor"
              strokeWidth={1.5}
              aria-hidden="true"
            >
              <path d="M10 2a8 8 0 1 0 0 16 8 8 0 0 0 0-16Z" />
              <path d="M10 6v4" strokeLinecap="round" />
              <path d="M10 13v.01" strokeLinecap="round" />
            </svg>
            <span className="font-sans text-[14px] font-medium text-moss">
              Working offline. Your cart is secure.
            </span>
          </div>
        </motion.div>
      )}
    </AnimatePresence>
  );
}