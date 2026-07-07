import { forwardRef } from "react";
import type { HTMLAttributes, ReactNode } from "react";
import { cn } from "../utils/cn";

export interface CardProps extends HTMLAttributes<HTMLDivElement> {
  children?: ReactNode;
}

export const Card = forwardRef<HTMLDivElement, CardProps>(function Card(
  { children, className, ...rest },
  ref,
) {
  return (
    <div
      ref={ref}
      className={cn(
        "relative rounded-lg bg-white/85 shadow-sm transition-transform duration-100 ease-out-expo",
        "active:scale-[0.98]",
        "after:pointer-events-none after:absolute after:inset-[5px] after:rounded-[inherit] after:border after:border-dashed after:border-[rgba(61,58,54,0.12)]",
        className,
      )}
      {...rest}
    >
      {children}
    </div>
  );
});
