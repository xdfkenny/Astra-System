// Primary button for main actions throughout the kiosk.
// Height: 64px minimum for accessibility. Full-width on mobile.
import { forwardRef, type ButtonHTMLAttributes, type ReactNode } from "react";
import { cn } from "@/utils/cn";

type ButtonVariant =
  | "default"
  | "destructive"
  | "outline"
  | "secondary"
  | "ghost"
  | "link";
type ButtonSize = "default" | "sm" | "lg" | "icon";

const BASE =
  "inline-flex items-center justify-center whitespace-nowrap font-sans font-medium text-[18px] rounded-full transition-all duration-100 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-moss focus-visible:ring-offset-2 disabled:pointer-events-none disabled:opacity-50";

const VARIANTS: Record<ButtonVariant, string> = {
  default:
    "bg-amber text-white shadow-[0_4px_16px_rgba(184,126,107,0.3)] hover:brightness-110 active:scale-[0.98] active:translate-y-[1px]",
  destructive: "bg-softRose text-white hover:bg-softRose/90 active:scale-[0.98]",
  outline: "border border-taupe bg-white text-charcoal hover:bg-warm-cream/50",
  secondary: "bg-white/70 border border-taupe text-charcoal hover:bg-warm-cream/50",
  ghost: "text-charcoal hover:bg-warm-cream/50",
  link: "text-amber underline-offset-4 hover:underline",
};

const SIZES: Record<ButtonSize, string> = {
  default: "h-16 px-6 py-3",
  sm: "h-12 rounded-[12px] px-4 text-[16px]",
  lg: "h-20 px-8 text-[20px]",
  icon: "h-10 w-10 rounded-[12px]",
};

export interface PrimaryButtonProps extends ButtonHTMLAttributes<HTMLButtonElement> {
  readonly variant?: ButtonVariant;
  readonly size?: ButtonSize;
  readonly isLoading?: boolean;
  readonly loadingText?: string;
  readonly leftIcon?: ReactNode;
  readonly rightIcon?: ReactNode;
}

export const PrimaryButton = forwardRef<HTMLButtonElement, PrimaryButtonProps>(
  (
    {
      className,
      variant = "default",
      size = "default",
      disabled = false,
      isLoading = false,
      loadingText,
      leftIcon,
      rightIcon,
      children,
      type = "button",
      ...props
    },
    ref,
  ) => {
    return (
      <button
        ref={ref}
        type={type}
        className={cn(BASE, VARIANTS[variant], SIZES[size], className)}
        disabled={disabled || isLoading}
        {...props}
      >
        {isLoading && (
          <span
            className="mr-2 inline-block h-4 w-4 animate-spin rounded-full border-2 border-current border-t-transparent"
            aria-hidden="true"
          />
        )}
        {leftIcon && !isLoading && <span className="mr-2">{leftIcon}</span>}
        <span className="truncate">
          {isLoading && loadingText ? loadingText : children}
        </span>
        {rightIcon && !isLoading && <span className="ml-2">{rightIcon}</span>}
      </button>
    );
  },
);

PrimaryButton.displayName = "PrimaryButton";
