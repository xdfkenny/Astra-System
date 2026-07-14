import { useState, useEffect, useCallback, useRef } from "react";
import { motion } from "framer-motion";
import { motion as motionTokens } from "@astra/design-tokens";
import { useKioskMachine } from "../machines/KioskMachineProvider";

interface ProcessingStage {
  readonly label: string;
  readonly durationMs: number;
}

const STAGES: readonly ProcessingStage[] = [
  { label: "Connecting to terminal...", durationMs: 1000 },
  { label: "Waiting for card...", durationMs: 2000 },
  { label: "Authorizing...", durationMs: 2500 },
  { label: "Finalizing...", durationMs: 1500 },
];

export function ProcessingScreen(): React.JSX.Element {
  const { send, state } = useKioskMachine();
  const [stageIndex, setStageIndex] = useState(0);
  const [dotsFilled, setDotsFilled] = useState(0);
  const mountedRef = useRef(true);

  useEffect(() => {
    return () => { mountedRef.current = false; };
  }, []);

  // The XState machine drives the real flow (finalizeOrder actor).
  // These stages are purely cosmetic feedback. The machine transition
  // will override this screen when the order is finalized or fails.
  useEffect(() => {
    if (stageIndex >= STAGES.length) return;
    const timer = setTimeout(() => {
      if (!mountedRef.current) return;
      setStageIndex((prev) => prev + 1);
      setDotsFilled((prev) => Math.min(prev + 1, STAGES.length));
    }, STAGES[stageIndex]?.durationMs ?? 1500);
    return () => { clearTimeout(timer); };
  }, [stageIndex]);

  const currentStage: ProcessingStage =
    STAGES[Math.min(stageIndex, STAGES.length - 1)] ??
    { label: "Processing...", durationMs: 1000 };

  const handleCancel = useCallback(() => {
    send({ type: "CANCEL_PAYMENT" });
  }, [send]);

  const showCancel = state.can({ type: "CANCEL_PAYMENT" });

  return (
    <div
      className="fixed inset-0 z-50 flex flex-col items-center justify-center bg-linen/90 backdrop-blur-[4px]"
      role="status"
      aria-live="polite"
      aria-label="Processing payment"
    >
      {/* Animated organic blob */}
      <motion.div
        className="h-48 w-48 rounded-full bg-moss opacity-[0.08]"
        animate={{
          borderRadius: [
            "60% 40% 30% 70% / 60% 30% 70% 40%",
            "30% 60% 70% 40% / 50% 60% 30% 60%",
            "50% 60% 30% 60% / 30% 40% 70% 50%",
            "60% 40% 30% 70% / 60% 30% 70% 40%",
          ],
          rotate: [0, 5, -5, 0],
        }}
        transition={{
          duration: 8,
          repeat: Infinity,
          ease: "easeInOut",
        }}
        aria-hidden="true"
      />

      {/* Processing text */}
      <p className="mt-4 font-sans text-[18px] text-stone">
        Processing payment...
      </p>

      {/* Terminal status */}
      <p className="mt-2 font-mono text-[14px] text-stone">
        Terminal: {currentStage.label}
      </p>

      {/* Progress dots — 4 dots that fill sequentially */}
      <div className="mt-6 flex items-center gap-3" aria-hidden="true">
        {Array.from({ length: STAGES.length }, (_, i) => (
          <motion.div
            key={i}
            className="h-3 w-3 rounded-full"
            animate={{
              backgroundColor: i < dotsFilled ? "#5A7A5C" : "#C4B8A8",
              scale: i === dotsFilled ? 1.3 : 1,
            }}
            transition={{ duration: 0.3, ease: motionTokens.easeInOutSoft }}
          />
        ))}
      </div>

      {/* Cancel button — shown only when the machine allows it */}
      {showCancel && (
        <button
          type="button"
          onClick={handleCancel}
          className="absolute bottom-10 left-1/2 -translate-x-1/2 h-14 rounded-[16px] bg-white/70 border border-taupe px-6 font-sans text-[16px] font-medium text-charcoal active:bg-warm-cream/50 transition-colors duration-100"
          aria-label="Cancel payment"
        >
          Cancel
        </button>
      )}
    </div>
  );
}

