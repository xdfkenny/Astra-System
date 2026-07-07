import { motion } from "framer-motion";
import { motion as motionTokens } from "@astra/design-tokens";
import { useSessionStore, resetCart } from "@astra/kiosk-state";
import { uuidV7 } from "@astra/shared-types";

/**
 * Attract Loop — idle state. Full-bleed rotating promo content would be
 * federated in from a CMS-driven remote in production; this ships the
 * structural shell + the "Tap to Start" primary CTA which begins a session.
 */
export function AttractScreen(): React.JSX.Element {
  const startSession = useSessionStore((s) => s.startSession);
  const kioskId =
    // eslint-disable-next-line @typescript-eslint/dot-notation
    (import.meta.env["VITE_KIOSK_ID"] as string | undefined) ?? "unknown-kiosk";

  const handleStart = (): void => {
    const sessionId = uuidV7();
    resetCart(kioskId);
    startSession(sessionId);
  };

  return (
    <div className="relative flex flex-1 flex-col items-center justify-end bg-gradient-to-b from-primary/90 to-primary p-6 text-white">
      <div className="mb-12 text-center">
        <h1 className="font-heading text-5xl font-bold tracking-tight">Astra-Service</h1>
        <p className="mt-3 text-lg text-white/80">Fast, contactless self-checkout</p>
      </div>
      <motion.button
        type="button"
        onClick={handleStart}
        whileTap={{ scale: 0.97 }}
        transition={{ duration: motionTokens.durationInstant }}
        className="mb-16 flex h-[var(--touch-primary-action)] w-full items-center justify-center rounded-lg bg-accent font-heading text-2xl font-bold text-ink shadow-lg active:bg-accent-hover"
        aria-label="Tap to start shopping"
      >
        Tap to Start
      </motion.button>
    </div>
  );
}
