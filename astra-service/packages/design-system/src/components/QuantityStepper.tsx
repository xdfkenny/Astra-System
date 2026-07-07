import { useCallback } from "react";
import type { HTMLAttributes } from "react";
import { cn } from "../utils/cn";
import { haptic } from "../utils/haptics";
import { IconButton } from "./IconButton";

export interface QuantityStepperProps
  extends Omit<HTMLAttributes<HTMLDivElement>, "onChange"> {
  value: number;
  min?: number;
  max?: number;
  onChange: (value: number) => void;
}

export function QuantityStepper({
  value,
  min = 0,
  max = Number.MAX_SAFE_INTEGER,
  onChange,
  className,
  ...rest
}: QuantityStepperProps) {
  const decrement = useCallback(() => {
    if (value > min) {
      haptic("light");
      onChange(value - 1);
    }
  }, [value, min, onChange]);

  const increment = useCallback(() => {
    if (value < max) {
      haptic("light");
      onChange(value + 1);
    }
  }, [value, max, onChange]);

  return (
    <div className={cn("inline-flex items-center gap-2", className)} {...rest}>
      <IconButton
        label="Decrease quantity"
        onClick={decrement}
        disabled={value <= min}
      >
        −
      </IconButton>
      <span
        className="min-w-14 text-center text-lg font-semibold tabular-nums"
        aria-live="polite"
        aria-atomic="true"
      >
        {value}
      </span>
      <IconButton
        label="Increase quantity"
        onClick={increment}
        disabled={value >= max}
      >
        +
      </IconButton>
    </div>
  );
}
