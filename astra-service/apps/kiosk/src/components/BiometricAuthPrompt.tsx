import { motion } from "framer-motion";
import { cn } from "@/utils/cn";

export interface BiometricAuthPromptProps {
  readonly isOpen: boolean;
  readonly onAuthenticate: () => void;
  readonly onCancel: () => void;
  readonly className?: string;
}

export function BiometricAuthPrompt({
  isOpen,
  onAuthenticate,
  onCancel,
  className,
}: BiometricAuthPromptProps): React.JSX.Element {
  if (!isOpen) {
    return <></>;
  }

  return (
    <div
      className={cn(
        "fixed inset-0 z-50 flex items-center justify-center bg-charcoal/40 p-6 backdrop-blur-[4px]",
        className,
      )}
      role="dialog"
      aria-modal="true"
      aria-label="Verify to complete payment"
    >
      <motion.div
        initial={{ opacity: 0, scale: 0.95 }}
        animate={{ opacity: 1, scale: 1 }}
        transition={{ duration: 0.2, ease: "easeOut" }}
        className="w-full max-w-sm rounded-[24px] bg-white p-6 text-center shadow-[0_8px_32px_rgba(45,42,38,0.12)]"
      >
        <h2 className="font-heading text-[28px] font-semibold text-charcoal">
          Verify to complete
        </h2>
        <p className="mt-3 font-sans text-[16px] text-stone">
          Please use the PIN pad or present your card to the terminal.
        </p>
        <div className="my-6 flex justify-center">
          <motion.div
            className="h-16 w-16 rounded-full bg-moss/10"
            animate={{ opacity: [0.6, 1, 0.6] }}
            transition={{ duration: 2, repeat: Infinity, ease: "easeInOut" }}
            aria-hidden="true"
          />
        </div>
        <div className="flex gap-3">
          <button
            type="button"
            onClick={onCancel}
            className="flex-1 rounded-[16px] border border-taupe px-4 py-3 font-sans text-[16px] font-medium text-charcoal transition-all duration-100 active:scale-[0.98]"
          >
            Cancel
          </button>
          <button
            type="button"
            onClick={onAuthenticate}
            className="flex-1 rounded-[16px] bg-moss px-4 py-3 font-sans text-[16px] font-medium text-white transition-all duration-100 active:scale-[0.98]"
          >
            Verify
          </button>
        </div>
      </motion.div>
    </div>
  );
}
