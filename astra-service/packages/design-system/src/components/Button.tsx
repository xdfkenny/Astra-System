import { forwardRef, useCallback } from "react";
import type { ButtonHTMLAttributes, MouseEvent, ReactNode } from "react";
import { cn } from "../utils/cn";
import { haptic } from "../utils/haptics";
import { Spinner } from "./Spinner";

export type ButtonVariant = "primary" | "secondary" | "ghost";

export interface ButtonProps extends ButtonHTMLAttributes<HTMLButtonElement> {
  variant?: ButtonVariant;
  loading?: boolean;
  children?: ReactNode;
}

export const Button = forwardRef<HTMLButtonElement, ButtonProps>(function Button(
  { variant = "primary", loading = false, disabled, children, className, onClick, ...rest },
  ref,
) {
  const handleClick = useCallback(
    (event: MouseEvent<HTMLButtonElement>) => {
      haptic("medium");
      onClick?.(event);
    },
    [onClick],
  );

  const isDisabled = disabled === true || loading;

  return (
    <button
      ref={ref}
      type="button"
      disabled={isDisabled}
      aria-busy={loading || undefined}
      onClick={handleClick}
      className={cn(
        "inline-flex min-h-14 min-w-14 items-center justify-center rounded-md px-6 py-3 text-base font-semibold transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-offset-2 motion-safe:duration-200 disabled:cursor-not-allowed",
        variant === "primary" &&
          "bg-primary text-white hover:bg-primary-hover focus-visible:ring-primary",
        variant === "secondary" &&
          "border-[0.5px] border-slate-900/10 bg-slate-100 text-text-primary hover:bg-slate-200 focus-visible:ring-slate-400",
        variant === "ghost" &&
          "bg-transparent text-text-secondary hover:bg-slate-100 focus-visible:ring-slate-400",
        isDisabled && "opacity-50",
        className,
      )}
      {...rest}
    >
      {loading ? <Spinner size="sm" className="mr-2" aria-hidden="true" /> : null}
      {children}
    </button>
  );
});
