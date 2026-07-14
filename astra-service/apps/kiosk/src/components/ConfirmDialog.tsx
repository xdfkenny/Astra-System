/*Modal/ConfirmDialog for critical actions like payment confirmations and employee overrides.
Dismissible on backdrop click, escape key, or explicit close.
*/
import { useEffect, useCallback } from "react";
import { motion, AnimatePresence } from "framer-motion";
import { motion as motionTokens } from "@astra/design-tokens";
import { cn } from "@/utils/cn";

export interface ModalProps {
  open: boolean;
  onClose: () => void;
  onConfirm: () => void;
  title?: string;
  message?: string;
  confirmText?: string;
  cancelText?: string;
  type?: "default" | "warning" | "success";
  className?: string;
}

export function Modal({
  open,
  onClose,
  onConfirm,
  title = "Confirm Action",
  message = "Are you sure you want to proceed?",
  confirmText = "Confirm",
  cancelText = "Cancel",
  type = "default",
  className,
}: ModalProps) {
  const handleBackdropClick = useCallback((e: React.MouseEvent) => {
    if (e.target === e.currentTarget) {
      onClose();
    }
  }, [onClose]);

  const handleKeyDown = useCallback((e: KeyboardEvent) => {
    if (e.key === "Escape") {
      onClose();
    } else if (e.key === "Enter") {
      onConfirm();
    }
  }, [onClose, onConfirm]);

  useEffect(() => {
    if (open) {
      document.addEventListener("keydown", handleKeyDown);
      document.body.style.overflow = "hidden";
    } else {
      document.body.style.overflow = "unset";
    }
    return () => {
      document.removeEventListener("keydown", handleKeyDown);
      document.body.style.overflow = "unset";
    };
  }, [open, handleKeyDown]);

  const typeStyles = {
    default: {
      backdrop: "bg-charcoal/20 backdrop-blur-[4px]",
      content: "bg-white/95 border-taupe/20",
      title: "text-charcoal",
      message: "text-stone",
    },
    warning: {
      backdrop: "bg-charcoal/20 backdrop-blur-[4px]",
      content: "bg-white/95 border-amber/20",
      title: "text-amber",
      message: "text-stone",
    },
    success: {
      backdrop: "bg-charcoal/20 backdrop-blur-[4px]",
      content: "bg-white/95 border-moss/20",
      title: "text-moss",
      message: "text-stone",
    },
  };

  const currentStyles = typeStyles[type];

  return (
    <AnimatePresence>
      {open && (
        <div
          className={cn(
            "fixed inset-0 z-50 flex items-center justify-center",
            currentStyles.backdrop
          )}
          onClick={handleBackdropClick}
          role="dialog"
          aria-modal="true"
          aria-labelledby="modal-title"
          aria-describedby="modal-description"
        >
          <motion.div
            initial={{ opacity: 0, scale: 0.95 }}
            animate={{ opacity: 1, scale: 1 }}
            exit={{ opacity: 0, scale: 0.95 }}
            transition={{ duration: 0.25, ease: motionTokens.easeOutExpo }}
            className={cn(
              "w-full max-w-md rounded-[24px] p-6 shadow-[0_8px_32px_rgba(45,42,38,0.12)]",
              "border",
              currentStyles.content,
              className
            )}
          >
            {title && (
              <h2
                id="modal-title"
                className={cn("font-heading text-[24px] font-semibold", currentStyles.title)}
              >
                {title}
              </h2>
            )}
            {message && (
              <p
                id="modal-description"
                className={cn("mt-3 font-sans text-[18px] leading-relaxed", currentStyles.message)}
              >
                {message}
              </p>
            )}

            <div
              className="mt-8 flex gap-4"
              role="group"
              aria-label="Modal actions"
            >
              <button
                type="button"
                onClick={onClose}
                className={cn(
                  "flex-1 h-14 rounded-[16px] border border-taupe font-sans text-[16px] font-medium",
                  "transition-all duration-100",
                  "hover:bg-taupe/10 active:scale-[0.98]"
                )}
                aria-label="Cancel action"
              >
                {cancelText}
              </button>
              <button
                type="button"
                onClick={onConfirm}
                className={cn(
                  "flex-1 h-14 rounded-full font-sans text-[16px] font-medium shadow-[0_4px_16px_rgba(184,126,107,0.3)]",
                  "transition-all duration-100",
                  type === "warning" && "bg-amber text-white hover:brightness-110 active:scale-[0.98]",
                  type === "success" && "bg-moss text-white hover:brightness-110 active:scale-[0.98]",
                  type === "default" && "bg-denim text-white hover:brightness-110 active:scale-[0.98]"
                )}
                aria-label="Confirm action"
              >
                {confirmText}
              </button>
            </div>
          </motion.div>
        </div>
      )}
    </AnimatePresence>
  );
}
