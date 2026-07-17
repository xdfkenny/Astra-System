/*Offline mode banner for when the kiosk is offline.
Appears at top of screen, automatically dismisses after 3 seconds.
*/
import { useEffect, useState } from "react";
import { motion, AnimatePresence } from "framer-motion";
import { motion as motionTokens } from "@astra/design-tokens";
import { useKioskMachine } from "../machines/KioskMachineProvider";
import { useTranslation } from "../i18n";

const AUTO_DISMISS_MS = 3_000;

export function OfflineBanner() {
  const { t } = useTranslation();
  const { state } = useKioskMachine();
  const [dismissed, setDismissed] = useState(false);

  const isOffline = state.context.isOfflineMode || state.context.apiStatus === "offline";

  useEffect(() => {
    if (!isOffline) {
      setDismissed(false);
      return;
    }

    const timer = setTimeout(() => {
      setDismissed(true);
    }, AUTO_DISMISS_MS);

    return () => { clearTimeout(timer); };
  }, [isOffline]);

  const isVisible = isOffline && !dismissed;

  return (
    <AnimatePresence>
      {isVisible && (
        <motion.div
          initial={{ y: -60, opacity: 0 }}
          animate={{ y: 0, opacity: 1 }}
          exit={{ y: -60, opacity: 0 }}
          transition={{ duration: 0.25, ease: motionTokens.easeOutExpo }}
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
              {t("offline.workingOffline")}
            </span>
            <button
              type="button"
              onClick={() => { setDismissed(true); }}
              className="ml-auto shrink-0 rounded-full p-1 text-moss/60 hover:text-moss touch-target"
              aria-label={t("offline.dismiss")}
            >
              <svg viewBox="0 0 16 16" className="h-4 w-4" fill="none" stroke="currentColor" strokeWidth={2}>
                <path d="M4 4l8 8M12 4l-8 8" strokeLinecap="round" />
              </svg>
            </button>
          </div>
        </motion.div>
      )}
    </AnimatePresence>
  );
}
