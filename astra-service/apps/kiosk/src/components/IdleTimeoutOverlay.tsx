import { motion } from "framer-motion";
import { motion as motionTokens } from "@astra/design-tokens";
import { useTranslation } from "../i18n";

export interface IdleTimeoutOverlayProps {
  onContinue: () => void;
  onReset: () => void;
}

export function IdleTimeoutOverlay({
  onContinue,
  onReset,
}: IdleTimeoutOverlayProps): React.JSX.Element {
  const { t } = useTranslation();
  return (
    <div
      className="absolute inset-0 z-30 flex items-center justify-center bg-charcoal/20 p-6"
      role="dialog"
      aria-modal="true"
      aria-label={t("idle.ariaLabel")}
    >
      <motion.div
        initial={{ opacity: 0, scale: 0.95 }}
        animate={{ opacity: 1, scale: 1 }}
        transition={{
          duration: 0.25,
          ease: motionTokens.easeOutExpo,
        }}
        className="w-full max-w-lg rounded-[24px] bg-white p-8 text-center shadow-[0_8px_32px_rgba(45,42,38,0.12)]"
      >
        <h2 className="font-heading text-[28px] font-semibold text-charcoal">
          {t("idle.warning")}
        </h2>
        <p className="mt-2 font-sans text-[18px] text-stone">
          {t("idle.message")}
        </p>
        <div className="mt-8 flex flex-col gap-3">
          <button
            type="button"
            onClick={onContinue}
            className="h-14 w-full rounded-full bg-moss text-white font-sans text-[18px] font-medium shadow-[0_4px_16px_rgba(90,122,92,0.3)] active:scale-[0.98] active:translate-y-[1px] transition-all duration-100"
            aria-label={t("idle.continueLabel")}
          >
            {t("idle.continue")}
          </button>
          <button
            type="button"
            onClick={onReset}
            className="h-14 w-full rounded-[16px] bg-white/70 border border-taupe font-sans text-[16px] font-medium text-charcoal active:bg-warm-cream/50 transition-colors duration-100"
            aria-label={t("idle.endSessionLabel")}
          >
            {t("idle.endSession")}
          </button>
        </div>
      </motion.div>
    </div>
  );
}

