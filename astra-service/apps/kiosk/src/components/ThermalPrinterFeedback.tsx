/*Thermal printer feedback for receipt printing.
Small toast or status banner with printer icon.
*/
import { useEffect, useState } from "react";
import { motion, AnimatePresence } from "framer-motion";
import { motion as motionTokens } from "@astra/design-tokens";
import { cn } from "@/utils/cn";

export interface ThermalPrinterFeedbackProps {
  isVisible: boolean;
  message?: string;
  status?: "printing" | "success" | "error";
}

export function ThermalPrinterFeedback({
  isVisible,
  message = "Printing receipt...",
  status = "printing",
}: ThermalPrinterFeedbackProps) {
  const [shouldShow, setShouldShow] = useState(false);

  useEffect(() => {
    if (!isVisible) {
      const timeout = setTimeout(() => {
        setShouldShow(false);
      }, 2000);
      return () => { clearTimeout(timeout); };
    }
    setShouldShow(true);
    return undefined;
  }, [isVisible]);

  const statusClasses = {
    printing: {
      bg: "bg-white/95",
      border: "border-taupe/20",
      icon: "text-moss",
      iconColor: "#5A7A5C",
      pulse: true,
    },
    success: {
      bg: "bg-pale-mint/90",
      border: "border-moss/20",
      icon: "text-moss",
      iconColor: "#5A7A5C",
      pulse: false,
    },
    error: {
      bg: "bg-softRose/90",
      border: "border-softRose/20",
      icon: "text-softRose",
      iconColor: "#C4A4A4",
      pulse: false,
    },
  };

  const currentStatus = statusClasses[status];

  return (
    <AnimatePresence>
      {shouldShow && (
        <motion.div
          initial={{ opacity: 0, y: -20, scale: 0.95 }}
          animate={{ opacity: 1, y: 0, scale: 1 }}
          exit={{ opacity: 0, y: -20, scale: 0.95 }}
          transition={{ duration: 0.25, ease: motionTokens.easeOutExpo }}
          className="fixed top-20 left-1/2 z-40 -translate-x-1/2"
        >
          <div
            className={cn(
              "rounded-[12px] border px-4 py-3 shadow-lg backdrop-blur-[8px]",
              currentStatus.bg,
              currentStatus.border
            )}
          >
            <div className="flex items-center gap-2">
              <svg
                viewBox="0 0 20 20"
                className={cn("h-5 w-5", currentStatus.icon)}
                fill="none"
                stroke="currentColor"
                strokeWidth="1.5"
                style={{ color: currentStatus.iconColor }}
              >
                <path d="M4 4h16v2H4V4zm0 4h8v8H4v-8zm12 0h2v8h-2V8z" />
                <path d="M4 10h12v2H4z" />
                {currentStatus.pulse && (
                  <motion.circle
                    cx="10"
                    cy="10"
                    r="4"
                    animate={{ r: [4, 6, 4] }}
                    transition={{ duration: 1.5, repeat: Infinity, ease: "easeInOut" }}
                  />
                )}
              </svg>
              <span
                className={cn(
                  "font-sans text-[14px] font-medium",
                  status === "success" && "text-moss",
                  status === "error" && "text-softRose",
                  status === "printing" && "text-charcoal"
                )}
              >
                {message}
              </span>
            </div>
          </div>
        </motion.div>
      )}
    </AnimatePresence>
  );
}