/*Toast notification system for brief, non-intrusive messages.
Appears at top center, below status bar.
Auto-dismiss after 4 seconds with progress bar.
*/
import { useEffect, useState } from "react";
import { AnimatePresence, motion } from "framer-motion";
import { motion as motionTokens } from "@astra/design-tokens";
import { cn } from "@/utils/cn";

export interface ToastProps {
  message: string;
  type?: "info" | "success" | "warning" | "error";
  duration?: number;
  onClose?: () => void;
  className?: string;
}

export function Toast({ message, type = "info", duration = 4000, onClose, className }: ToastProps) {
  const [isVisible, setIsVisible] = useState(true);
  const [progress, setProgress] = useState(100);

  useEffect(() => {
    if (duration === Infinity) return;

    const interval = setInterval(() => {
      setProgress((prev) => {
        if (prev <= 0) {
          clearInterval(interval);
          setIsVisible(false);
          onClose?.();
          return 0;
        }
        return prev - (100 * (100 / duration));
      });
    }, 50);

    return () => { clearInterval(interval); };
  }, [duration, onClose]);

  const typeClasses = {
    info: "bg-charcoal text-white",
    success: "bg-moss text-white",
    warning: "bg-amber text-white",
    error: "bg-softRose text-white",
  };

  return (
    <AnimatePresence>
      {isVisible && (
        <motion.div
          initial={{ y: -20, opacity: 0 }}
          animate={{ y: 0, opacity: 1 }}
          exit={{ y: -20, opacity: 0 }}
          transition={{ duration: 0.25, ease: motionTokens.easeOutExpo }}
          className={cn(
            "relative z-40 mx-auto mt-2 max-w-sm rounded-[12px] px-4 py-3 shadow-lg",
            "font-sans text-[14px] font-medium",
            "border border-charcoal/10 backdrop-blur-[8px]",
            typeClasses[type],
            className
          )}
          role="alert"
          aria-live="polite"
        >
          <div className="flex items-center justify-between">
            <span className="pr-4">{message}</span>
          </div>
          {duration !== Infinity && (
            <motion.div
              className="absolute bottom-0 left-0 h-1 rounded-b-[12px] bg-white/20"
              style={{ width: `${progress}%` }}
              transition={{ ease: "linear", duration: 0.3 }}
              initial={{ width: "100%" }}
              exit={{ width: "0%" }}
            />
          )}
        </motion.div>
      )}
    </AnimatePresence>
  );
}
