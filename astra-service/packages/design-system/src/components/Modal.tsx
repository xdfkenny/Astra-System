import { forwardRef, useEffect, useId, useRef } from "react";
import type { HTMLAttributes, ReactNode } from "react";
import { createPortal } from "react-dom";
import { cn } from "../utils/cn";
import { createFocusTrap } from "../utils/a11y";
import { IconButton } from "./IconButton";

export interface ModalProps extends HTMLAttributes<HTMLDivElement> {
  open: boolean;
  onClose: () => void;
  title: string;
  description?: string;
  children?: ReactNode;
}

export const Modal = forwardRef<HTMLDivElement, ModalProps>(function Modal(
  { open, onClose, title, description, children, className, ...rest },
  ref,
) {
  const contentRef = useRef<HTMLDivElement>(null);
  const trapRef = useRef<ReturnType<typeof createFocusTrap> | null>(null);
  const titleId = useId();
  const descId = useId();

  useEffect(() => {
    if (!open) {
      return;
    }
    const content = contentRef.current;
    if (!content) {
      return;
    }
    trapRef.current = createFocusTrap(content, { onEscape: onClose });
    trapRef.current.activate();
    return () => {
      trapRef.current?.deactivate();
      trapRef.current = null;
    };
  }, [open, onClose]);

  if (!open) {
    return null;
  }

  return createPortal(
    <div
      ref={ref}
      className="fixed inset-0 z-50 flex items-center justify-center bg-slate-900/50 p-4"
      onClick={(event) => {
        if (event.target === event.currentTarget) {
          onClose();
        }
      }}
      {...rest}
    >
      <div
        ref={contentRef}
        role="dialog"
        aria-modal="true"
        aria-labelledby={titleId}
        aria-describedby={description ? descId : undefined}
        className={cn(
          "w-full max-w-md rounded-lg border-[0.5px] border-slate-900/10 bg-surface p-6 shadow-xl",
          className,
        )}
      >
        <div className="mb-4 flex items-start justify-between gap-4">
          <div>
            <h2
              id={titleId}
              className="text-xl font-semibold text-text-primary"
            >
              {title}
            </h2>
            {description ? (
              <p id={descId} className="mt-1 text-sm text-text-secondary">
                {description}
              </p>
            ) : null}
          </div>
          <IconButton label="Close dialog" onClick={onClose}>
            ✕
          </IconButton>
        </div>
        {children}
      </div>
    </div>,
    document.body,
  );
});
