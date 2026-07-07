import { motion } from "framer-motion";
import { PrimaryButton } from "@astra/ui-kit";
import { motion as motionTokens } from "@astra/design-tokens";
import { useKioskMachine } from "../machines/KioskMachineProvider";
import { uuidV7 } from "@astra/shared-types";
import { resetCart } from "@astra/kiosk-state";

const KIOSK_ID = import.meta.env.VITE_KIOSK_ID ?? "kiosk-local";

/**
 * Attract Loop — idle state. Full-bleed branded screen with the primary
 * "Tap to Start" CTA that begins a new session.
 */
export function AttractScreen(): React.JSX.Element {
  const { send } = useKioskMachine();

  const handleStart = (): void => {
    resetCart(KIOSK_ID);
    send({ type: "START_SESSION", sessionId: uuidV7() });
  };

  return (
    <div className="relative flex flex-1 flex-col items-center justify-end bg-gradient-to-b from-primary to-primary-pressed p-6 text-white">
      <div className="mb-12 text-center">
        <h1 className="font-heading text-5xl font-bold tracking-tight">Astra-Service</h1>
        <p className="mt-3 text-lg text-white/80">Fast, contactless self-checkout</p>
      </div>
      <motion.div
        whileTap={{ scale: 0.97 }}
        transition={{ duration: motionTokens.durationInstant }}
        className="mb-16 w-full"
      >
        <PrimaryButton
          variant="accent"
          className="w-full text-ink"
          style={{ minHeight: "88px", fontSize: "1.5rem" }}
          onClick={handleStart}
          aria-label="Tap to start shopping"
        >
          Tap to Start
        </PrimaryButton>
      </motion.div>
    </div>
  );
}
