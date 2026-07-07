import { forwardRef, useCallback } from "react";
import type { ButtonHTMLAttributes, MouseEvent, ReactNode } from "react";
import { cn } from "../utils/cn";
import { haptic } from "../utils/haptics";
import { Spinner } from "./Spinner";

export type ButtonVariant = "primary" | "cta" | "secondary" | "ghost";

export interface ButtonProps extends ButtonHTMLAttributes<HTMLButtonElement> {
  variant?: ButtonVariant;
  loading?: boolean;
  children?: ReactNode;
}

const variantStyles: Record<ButtonVariant, string> = {
  cta: [
    "h-16 w-full max-w-md rounded-full bg-cta text-white shadow-[0_4px_16px_rgba(184,126,107,0.3)]",
    "hover:brightness-110 hover:scale-[1.01]",
    "active:scale-[0.98]",
  ].join(" "),
  primary: [
    "h-14 rounded-lg bg-primary text-white",
    "hover:bg-primary-hover",
    "active:scale-[0.98]",
  ].join(" "),
  secondary: [
    "h-14 rounded-[16px] border border-taupe bg-white/70 text-ink",
    "hover:bg-warm-cream/50",
    "active:scale-[0.98]",
  ].join(" "),
  ghost: [
    "h-14 bg-transparent text-denim",
    "hover:bg-warm-cream/50",
    "active:scale-[0.98]",
  ].join(" "),
};

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
        "inline-flex items-center justify-center font-ui text-base font-medium transition-all duration-150 ease-out-expo px-6",
        "focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-offset-2 focus-visible:ring-moss",
        "disabled:cursor-not-allowed disabled:opacity-40 disabled:grayscale-[0.5]",
        variantStyles[variant],
        className,
      )}
      {...rest}
    >
      {loading ? (
        <Spinner size="sm" className="mr-2 text-moss" aria-hidden="true" />
      ) : null}
      {children}
    </button>
  );
});
