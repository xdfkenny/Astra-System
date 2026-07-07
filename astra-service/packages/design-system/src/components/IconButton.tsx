import { forwardRef, useCallback } from "react";
import type { ButtonHTMLAttributes, MouseEvent, ReactNode } from "react";
import { cn } from "../utils/cn";
import { haptic } from "../utils/haptics";

export interface IconButtonProps
  extends ButtonHTMLAttributes<HTMLButtonElement> {
  children: ReactNode;
  label: string;
}

export const IconButton = forwardRef<HTMLButtonElement, IconButtonProps>(
  function IconButton(
    { label, className, onClick, children, ...rest },
    ref,
  ) {
    const handleClick = useCallback(
      (event: MouseEvent<HTMLButtonElement>) => {
        haptic("light");
        onClick?.(event);
      },
      [onClick],
    );

    return (
      <button
        ref={ref}
        type="button"
        aria-label={label}
        onClick={handleClick}
        className={cn(
          "inline-flex h-14 w-14 items-center justify-center rounded-full border-[0.5px] border-slate-900/10 bg-surface text-text-primary shadow-sm transition-colors hover:bg-slate-50 active:bg-slate-100 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary motion-safe:duration-200 disabled:cursor-not-allowed disabled:opacity-50",
          className,
        )}
        {...rest}
      >
        {children}
      </button>
    );
  },
);
