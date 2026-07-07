import { motion } from "framer-motion";
import { PrimaryButton } from "@astra/ui-kit";
import { motion as motionTokens } from "@astra/design-tokens";

export interface IdleTimeoutOverlayProps {
  onContinue: () => void;
  onReset: () => void;
}

/**
 * Idle timeout overlay. Arms when the customer walks away; tapping "I'm still
 * here" resumes the session, otherwise the kiosk auto-resets to Attract.
 */
export function IdleTimeoutOverlay({
  onContinue,
  onReset,
}: IdleTimeoutOverlayProps): React.JSX.Element {
  return (
    <div className="absolute inset-0 z-modal flex items-center justify-center bg-overlay p-6">
      <motion.div
        initial={{ opacity: 0, scale: 0.95 }}
        animate={{ opacity: 1, scale: 1 }}
        transition={{
          duration: motionTokens.durationBase,
          ease: motionTokens.easeStandard,
        }}
        className="w-full max-w-lg rounded-xl bg-surface p-8 text-center shadow-xl"
      >
        <h2 className="font-heading text-3xl font-bold text-ink">Still shopping?</h2>
        <p className="mt-2 text-ink-muted">Tap below to continue, or this kiosk will reset for the next customer.</p>
        <div className="mt-8 flex flex-col gap-3">
          <PrimaryButton
            variant="primary"
            className="w-full"
            onClick={onContinue}
          >
            I&apos;m Still Here
          </PrimaryButton>
          <PrimaryButton
            variant="ghost"
            className="w-full"
            onClick={onReset}
          >
            Start Over
          </PrimaryButton>
        </div>
      </motion.div>
    </div>
  );
}
