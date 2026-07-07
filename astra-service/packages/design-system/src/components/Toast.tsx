import { Children, useEffect } from "react";
import type { ReactNode } from "react";
import { createPortal } from "react-dom";
import { cn } from "../utils/cn";
import { announce } from "../utils/a11y";

export type ToastVariant = "info" | "success" | "warning" | "error";

const VARIANTS: Record<ToastVariant, string> = {
  info: "bg-slate-800 text-white",
  success: "bg-emerald-700 text-white",
  warning: "bg-amber-600 text-white",
  error: "bg-rose-600 text-white",
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
  duration = 5000,
}: ToastProps) {
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
        "fixed bottom-4 left-1/2 z-50 min-h-14 -translate-x-1/2 rounded-lg px-6 py-3 shadow-lg transition-opacity motion-safe:duration-300",
        VARIANTS[variant],
      )}
    >
      {message}
    </div>,
    document.body,
  );
}
