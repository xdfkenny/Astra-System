import { useState, useEffect, useCallback } from "react";
import { motion } from "framer-motion";
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

  // Progress through stages
  useEffect(() => {
    if (stageIndex >= STAGES.length) return;
    const timer = setTimeout(() => {
      setStageIndex((prev) => prev + 1);
      setDotsFilled((prev) => Math.min(prev + 1, STAGES.length));
    }, STAGES[stageIndex]?.durationMs ?? 0);
    return () => { clearTimeout(timer); };
  }, [stageIndex]);

  const currentStage = STAGES[Math.min(stageIndex, STAGES.length - 1)];

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
    <div className="fixed inset-0 z-50 flex flex-col items-center justify-center bg-linen/90 backdrop-blur-[4px]">
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
      <p
        className="mt-4 font-sans text-[18px] text-stone"
        role="status"
        aria-live="polite"
      >
        Processing payment...
      </p>

      {/* Terminal status */}
      <p className="mt-2 font-mono text-caption text-stone">
        Terminal: {currentStage.label}
      </p>

      {/* Progress dots */}
      <div className="mt-6 flex items-center gap-3" aria-hidden="true">
        {Array.from({ length: STAGES.length }, (_, i) => (
          <motion.div
            key={i}
            className="h-3 w-3 rounded-full"
            animate={{
              backgroundColor: i < dotsFilled ? "#5A7A5C" : "#C4B8A8",
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
        className="absolute bottom-10 left-1/2 -translate-x-1/2 h-14 rounded-[16px] bg-white/70 border border-taupe px-6 font-sans text-[16px] font-medium text-charcoal"
        aria-label="Cancel payment"
      >
        Cancel
      </button>
    </div>
  );
}
