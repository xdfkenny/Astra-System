// Secondary button for supporting actions throughout the kiosk.
// Height: 56px. Outlined surface, used for back/cancel actions.
import { forwardRef, type ButtonHTMLAttributes, type ReactNode } from "react";
import { cn } from "@/utils/cn";

type ButtonVariant = "default" | "outline" | "secondary" | "ghost" | "destructive";
type ButtonSize = "default" | "sm" | "lg" | "icon";

const BASE =
  "inline-flex items-center justify-center whitespace-nowrap font-sans font-medium text-[16px] rounded-[16px] transition-all duration-100 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-moss focus-visible:ring-offset-2 disabled:pointer-events-none disabled:opacity-50";

const VARIANTS: Record<ButtonVariant, string> = {
  default: "bg-white/70 border border-taupe text-charcoal hover:bg-warm-cream/50",
  outline: "border border-taupe bg-transparent text-charcoal hover:bg-warm-cream/50",
  secondary: "bg-white border border-taupe text-charcoal hover:bg-warm-cream/50",
  ghost: "text-stone hover:text-charcoal hover:bg-warm-cream/50",
  destructive: "bg-softRose text-white hover:bg-softRose/90",
};

const SIZES: Record<ButtonSize, string> = {
  default: "h-14 px-6 py-3",
  sm: "h-12 px-4",
  lg: "h-16 px-8",
  icon: "h-10 w-10",
};

export interface SecondaryButtonProps extends ButtonHTMLAttributes<HTMLButtonElement> {
  readonly variant?: ButtonVariant;
  readonly size?: ButtonSize;
  readonly leftIcon?: ReactNode;
  readonly rightIcon?: ReactNode;
}

export const SecondaryButton = forwardRef<HTMLButtonElement, SecondaryButtonProps>(
  (
    {
      className,
      variant = "default",
      size = "default",
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
        {...props}
      >
        {leftIcon && <span className="mr-2">{leftIcon}</span>}
        <span className="truncate">{children}</span>
        {rightIcon && <span className="ml-2">{rightIcon}</span>}
      </button>
    );
  },
);

SecondaryButton.displayName = "SecondaryButton";
