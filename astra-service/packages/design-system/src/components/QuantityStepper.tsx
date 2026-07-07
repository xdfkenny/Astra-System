import { useCallback, useEffect, useRef, useState } from "react";
import type { HTMLAttributes } from "react";
import { cn } from "../utils/cn";
import { haptic } from "../utils/haptics";

export interface QuantityStepperProps
  extends Omit<HTMLAttributes<HTMLDivElement>, "onChange"> {
  value: number;
  min?: number;
  max?: number;
  onChange: (value: number) => void;
}

const LONG_PRESS_DELAY_MS = 500;
const REPEAT_INTERVAL_MS = 100;

export function QuantityStepper({
  value,
  min = 0,
  max = Number.MAX_SAFE_INTEGER,
  onChange,
  className,
  ...rest
}: QuantityStepperProps) {
  const [decrementing, setDecrementing] = useState(false);
  const [incrementing, setIncrementing] = useState(false);
  const intervalRef = useRef<ReturnType<typeof setInterval> | null>(null);
  const timeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  const clearTimers = useCallback(() => {
    if (intervalRef.current) {
      clearInterval(intervalRef.current);
      intervalRef.current = null;
    }
    if (timeoutRef.current) {
      clearTimeout(timeoutRef.current);
      timeoutRef.current = null;
    }
  }, []);

  useEffect(() => {
    return () => {
      clearTimers();
    };
  }, [clearTimers]);

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

  const startRepeating = useCallback(
    (action: () => void) => {
      clearTimers();
      action();
      timeoutRef.current = setTimeout(() => {
        intervalRef.current = setInterval(action, REPEAT_INTERVAL_MS);
      }, LONG_PRESS_DELAY_MS);
    },
    [clearTimers],
  );

  const handlePointerUp = useCallback(() => {
    clearTimers();
    setDecrementing(false);
    setIncrementing(false);
  }, [clearTimers]);

  const atMin = value <= min;
  const atMax = value >= max;

  return (
    <div
      className={cn("inline-flex items-center gap-2", className)}
      onPointerUp={handlePointerUp}
      onPointerLeave={handlePointerUp}
      {...rest}
    >
      <StepButton
        label="Decrease quantity"
        disabled={atMin}
        onClick={decrement}
        onPointerDown={() => {
          setDecrementing(true);
          startRepeating(decrement);
        }}
        pressed={decrementing}
      >
        −
      </StepButton>
      <span
        className="min-w-12 text-center text-[20px] font-semibold tabular-nums text-ink"
        aria-live="polite"
        aria-atomic="true"
      >
        {value}
      </span>
      <StepButton
        label="Increase quantity"
        disabled={atMax}
        onClick={increment}
        onPointerDown={() => {
          setIncrementing(true);
          startRepeating(increment);
        }}
        pressed={incrementing}
      >
        +
      </StepButton>
    </div>
  );
}

interface StepButtonProps {
  label: string;
  disabled: boolean;
  onClick: () => void;
  onPointerDown: () => void;
  pressed: boolean;
  children: React.ReactNode;
}

function StepButton({
  label,
  disabled,
  onClick,
  onPointerDown,
  pressed,
  children,
}: StepButtonProps) {
  return (
    <button
      type="button"
      aria-label={label}
      disabled={disabled}
      onClick={onClick}
      onPointerDown={onPointerDown}
      className={cn(
        "flex h-12 w-12 items-center justify-center rounded-full border border-taupe bg-linen text-xl text-charcoal transition-transform duration-150 ease-out-expo",
        "hover:bg-warm-cream",
        "active:scale-95",
        "focus-visible:ring-2 focus-visible:ring-moss focus-visible:ring-offset-2",
        "disabled:opacity-40 disabled:grayscale-[0.5]",
        pressed && "scale-95 bg-warm-cream",
      )}
    >
      {children}
    </button>
  );
}
