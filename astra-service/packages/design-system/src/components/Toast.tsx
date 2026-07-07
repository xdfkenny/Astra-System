import { Children, useEffect, useRef } from "react";
import type { ReactNode } from "react";
import { createPortal } from "react-dom";
import { cn } from "../utils/cn";
import { announce } from "../utils/a11y";

export type ToastVariant = "info" | "success" | "warning" | "error";

const VARIANTS: Record<ToastVariant, string> = {
  info: "bg-charcoal text-white",
  success: "bg-charcoal text-white",
  warning: "bg-charcoal text-white",
  error: "bg-charcoal text-white",
};

export interface ToastProps {
  open: boolean;
  message: ReactNode;
  variant?: ToastVariant;
  onClose?: () => void;
  duration?: number;
}

function toPlainText(node: ReactNode): string {
  return Children.toArray(node)
    .map((child) =>
      typeof child === "string" || typeof child === "number"
        ? String(child)
        : "",
    )
    .join(" ")
    .trim();
}

export function Toast({
  open,
  message,
  variant = "info",
  onClose,
  duration = 4000,
}: ToastProps) {
  const progressRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (!open) {
      return;
    }
    announce(toPlainText(message), "polite");
    if (!onClose || duration <= 0) {
      return;
    }
    const id = setTimeout(onClose, duration);
    return () => clearTimeout(id);
  }, [open, message, variant, duration, onClose]);

  if (!open) {
    return null;
  }

  return createPortal(
    <div
      role="status"
      aria-live="polite"
      aria-atomic="true"
      className={cn(
        "fixed left-1/2 top-16 z-40 min-h-14 max-w-[90%] -translate-x-1/2 rounded-[12px] px-6 py-3 shadow-lg transition-all duration-base ease-out-expo",
        VARIANTS[variant],
      )}
      style={{
        animation: "toast-enter 250ms cubic-bezier(0.16, 1, 0.3, 1) forwards",
      }}
    >
      <div className="flex items-center gap-2">{message}</div>
      {duration > 0 && (
        <div className="absolute bottom-0 left-0 h-0.5 w-full overflow-hidden rounded-b-[12px]">
          <div
            ref={progressRef}
            className="h-full bg-amber"
            style={{
              animation: `toast-progress ${duration}ms linear forwards`,
            }}
          />
        </div>
      )}
    </div>,
    document.body,
  );
}
