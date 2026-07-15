import { useState, useEffect, useCallback } from "react";
import { motion } from "framer-motion";
import { haptic } from "@astra/design-system";
import { useKioskMachine } from "../machines/KioskMachineProvider";
import { apiClient } from "../state/apiClient";

interface ProcessingStage {
  readonly label: string;
  readonly durationMs: number;
}

const STAGES: readonly ProcessingStage[] = [
  { label: "Connecting to terminal...", durationMs: 1500 },
  { label: "Waiting for card...", durationMs: 2000 },
  { label: "Authorizing...", durationMs: 2500 },
  { label: "Finalizing...", durationMs: 1500 },
];

export function ProcessingScreen(): React.JSX.Element {
  const { send } = useKioskMachine();
  const [stageIndex, setStageIndex] = useState(0);
  const [dotsFilled, setDotsFilled] = useState(0);

  // Subtle haptic pulse on each stage transition (if supported by hardware).
  useEffect(() => {
    haptic("light");
  }, [stageIndex]);

  // Progress through stages
  useEffect(() => {
    if (stageIndex >= STAGES.length) return;
    const timer = setTimeout(() => {
      setStageIndex((prev) => prev + 1);
      setDotsFilled((prev) => Math.min(prev + 1, STAGES.length));
    }, STAGES[stageIndex]?.durationMs ?? 0);
    return () => {
      clearTimeout(timer);
    };
  }, [stageIndex]);

  const currentStage = STAGES[Math.min(stageIndex, STAGES.length - 1)] ?? {
    label: "Processing...",
    durationMs: 0,
  };

  // When processing is complete, create the order
  useEffect(() => {
    if (stageIndex >= STAGES.length) {
      const processOrder = async () => {
        try {
          // In a real implementation, we would get the payment ID from the payment result
          // For now, we'll simulate this
          const paymentId = crypto.randomUUID();
          const cartId = "current-cart-id"; // This should come from state

          const order = await apiClient.createOrder(cartId, paymentId);

          // Send the order to the state machine
          send({ type: "ORDER_FINALIZED", order });
        } catch (error) {
          console.error("Failed to create order:", error);
          send({ type: "PAYMENT_FAILED", message: "Failed to finalize order" });
        }
      };

      void processOrder();
    }
  }, [stageIndex, send]);

  const handleCancel = useCallback(() => {
    send({ type: "CANCEL_PAYMENT" });
  }, [send]);

  return (
    <div className="bg-linen/90 fixed inset-0 z-50 flex flex-col items-center justify-center backdrop-blur-[4px]">
      {/* Animated organic blob */}
      <motion.div
        className="bg-moss h-48 w-48 rounded-full opacity-[0.08]"
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
      <p className="text-stone mt-4 font-sans text-[18px]" role="status" aria-live="polite">
        Processing payment...
      </p>

      {/* Terminal status */}
      <p className="text-caption text-stone mt-2 font-mono">Terminal: {currentStage.label}</p>

      {/* Progress dots */}
      <div className="mt-6 flex items-center gap-3" aria-hidden="true">
        {Array.from({ length: STAGES.length }, (_, i) => (
          <motion.div
            key={i}
            className="h-3 w-3 rounded-full"
            animate={{
              backgroundColor:
                i < dotsFilled
                  ? "var(--color-moss)"
                  : "var(--color-taupe)",
              scale: i === dotsFilled ? 1.3 : 1,
            }}
            transition={{ duration: 0.3 }}
          />
        ))}
      </div>

      {/* Cancel button */}
      <button
        type="button"
        onClick={handleCancel}
        className="border-taupe text-charcoal absolute bottom-10 left-1/2 h-14 -translate-x-1/2 rounded-[16px] border bg-white/70 px-6 font-sans text-[16px] font-medium"
        aria-label="Cancel payment"
      >
        Cancel
      </button>
    </div>
  );
}
