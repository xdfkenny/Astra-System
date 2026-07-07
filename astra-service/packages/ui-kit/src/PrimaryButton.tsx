import { motion } from "framer-motion";
import { motion as motionTokens } from "@astra/design-tokens";
import type { ButtonHTMLAttributes } from "react";
import type { HTMLMotionProps } from "framer-motion";

export interface PrimaryButtonProps extends ButtonHTMLAttributes<HTMLButtonElement> {
  readonly variant?: "primary" | "accent" | "danger" | "ghost";
  readonly assistHighlight?: boolean;
}

/**
 * Shared primary CTA button — 88px thumb-zone height, hardware-accelerated
 * tap feedback only (scale transform), and an optional "Silent Assist"
 * highlight pulse (deep-improvement #4) driven by a CSS animation rather
 * than a JS timer loop so it costs nothing when not armed.
 */
export function PrimaryButton({
  variant = "primary",
  assistHighlight = false,
  className = "",
  children,
  ...rest
}: PrimaryButtonProps): React.JSX.Element {
  const variantClasses: Record<NonNullable<PrimaryButtonProps["variant"]>, string> = {
    primary: "bg-primary text-white active:bg-primary-pressed",
    accent: "bg-accent text-ink active:bg-accent-hover",
    danger: "bg-error text-white active:brightness-90",
    ghost: "bg-transparent text-ink border border-border-strong",
  };

  return (
    <motion.button
      type="button"
      whileTap={{ scale: 0.97 }}
      transition={{ duration: motionTokens.durationInstant }}
      className={`flex h-[var(--touch-primary-action)] min-w-[var(--touch-min)] items-center justify-center rounded-lg font-heading text-xl font-semibold shadow-md transition-shadow ${variantClasses[variant]} ${assistHighlight ? "astra-assist-pulse" : ""} ${className}`}
      {...(rest as unknown as HTMLMotionProps<"button">)}
    >
      {children}
    </motion.button>
  );
}
