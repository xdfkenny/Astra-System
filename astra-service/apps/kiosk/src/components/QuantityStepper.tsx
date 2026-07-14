// Quantity stepper for cart items and item detail customization.
// Horizontal layout: - button | value | + button
// Long-press on + or - accelerates after 500ms
import { useEffect, useCallback, useRef, useState } from "react";
import { cn } from "@/utils/cn";

export interface QuantityStepperProps {
  value: number;
  onChange: (value: number) => void;
  min?: number;
  max?: number;
  disabled?: boolean;
  size?: "sm" | "md" | "lg";
  className?: string;
}

export function QuantityStepper({
  value,
  onChange,
  min = 0,
  max = 99,
  disabled = false,
  size = "md",
  className,
}: QuantityStepperProps) {
  const [, setPressCount] = useState(0);
  const [, setAcceleration] = useState(1);
  const pressTimerRef = useRef<ReturnType<typeof setTimeout> | undefined>(undefined);
  const intervalRef = useRef<ReturnType<typeof setTimeout> | undefined>(undefined);
  const lastChangeTimeRef = useRef<number>(0);

  const sizeClasses = {
    sm: {
      button: "w-10 h-10",
      count: "w-10 h-10 text-base",
      icon: "w-4 h-4",
    },
    md: {
      button: "w-12 h-12",
      count: "w-12 h-12 text-lg",
      icon: "w-5 h-5",
    },
    lg: {
      button: "w-14 h-14",
      count: "w-14 h-14 text-xl",
      icon: "w-6 h-6",
    },
  };

  const handleDecrement = useCallback(() => {
    if (value > min) {
      onChange(value - 1);
    }
  }, [value, min, onChange]);

  const handleIncrement = useCallback(() => {
    if (value < max) {
      onChange(value + 1);
    }
  }, [value, max, onChange]);

  const handlePress = useCallback(
    (type: "decrement" | "increment") => {
      if (disabled) return;

      const now = Date.now();
      const timeSinceLastChange = now - lastChangeTimeRef.current;

      if (timeSinceLastChange < 500) {
        setAcceleration((acc) => Math.min(acc * 1.5, 4));
      } else {
        setAcceleration(1);
      }

      lastChangeTimeRef.current = now;

      if (type === "decrement") {
        handleDecrement();
      } else {
        handleIncrement();
      }
    },
    [disabled, handleDecrement, handleIncrement]
  );

  const handleMouseDown = useCallback(
    (type: "decrement" | "increment") => {
      handlePress(type);

      pressTimerRef.current = setTimeout(() => {
        setPressCount(1);
        intervalRef.current = setInterval(() => {
          handlePress(type);
          setPressCount((c) => c + 1);
        }, 100);
      }, 500);
    },
    [handlePress]
  );

  const handleMouseUp = useCallback(() => {
    if (pressTimerRef.current) {
      clearTimeout(pressTimerRef.current);
      pressTimerRef.current = undefined;
    }
    if (intervalRef.current) {
      clearInterval(intervalRef.current);
      intervalRef.current = undefined;
    }
    setPressCount(0);
    setAcceleration(1);
  }, []);

  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if (disabled) return;

      if (e.key === "ArrowDown" || e.key === "a") {
        e.preventDefault();
        handleDecrement();
      } else if (e.key === "ArrowUp" || e.key === "d") {
        e.preventDefault();
        handleIncrement();
      }
    };

    window.addEventListener("keydown", handleKeyDown);
    return () => { window.removeEventListener("keydown", handleKeyDown); };
  }, [disabled, handleDecrement, handleIncrement]);

  useEffect(() => {
    return () => {
      if (pressTimerRef.current) clearTimeout(pressTimerRef.current);
      if (intervalRef.current) clearInterval(intervalRef.current);
    };
  }, []);

  return (
    <div className={cn("inline-flex items-center rounded-full bg-linen border border-taupe", className)}>
      <button
        type="button"
        className={cn(
          "flex items-center justify-center transition-colors rounded-l-full hover:bg-stone/10 active:bg-stone/20",
          sizeClasses[size].button,
          value <= min
            ? "text-stone/40 cursor-not-allowed"
            : "text-charcoal hover:text-amber"
        )}
        onMouseDown={() => {
          handleMouseDown("decrement");
        }}
        onMouseUp={handleMouseUp}
        onMouseLeave={handleMouseUp}
        onTouchStart={(e) => {
          const touch = e.touches[0];
          if (touch) {
            const rect = e.currentTarget.getBoundingClientRect();
            const x = touch.clientX - rect.left;
            if (x < rect.width / 2) {
              handleMouseDown("decrement");
            }
          }
        }}
        onTouchEnd={handleMouseUp}
        disabled={value <= min || disabled}
        aria-label="Decrease quantity"
      >
        <span className={sizeClasses[size].icon}>
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5">
            <path d="M20 12H4" strokeLinecap="round" />
          </svg>
        </span>
      </button>

      <div className={cn(
        "flex items-center justify-center font-medium tabular-nums select-none",
        sizeClasses[size].count
      )}>
        {value}
      </div>

      <button
        type="button"
        className={cn(
          "flex items-center justify-center transition-colors rounded-r-full hover:bg-stone/10 active:bg-stone/20",
          sizeClasses[size].button,
          value >= max
            ? "text-stone/40 cursor-not-allowed"
            : "text-charcoal hover:text-amber"
        )}
        onMouseDown={() => {
          handleMouseDown("increment");
        }}
        onMouseUp={handleMouseUp}
        onMouseLeave={handleMouseUp}
        onTouchStart={(e) => {
          const touch = e.touches[0];
          if (touch) {
            const rect = e.currentTarget.getBoundingClientRect();
            const x = touch.clientX - rect.left;
            if (x >= rect.width / 2) {
              handleMouseDown("increment");
            }
          }
        }}
        onTouchEnd={handleMouseUp}
        disabled={value >= max || disabled}
        aria-label="Increase quantity"
      >
        <span className={sizeClasses[size].icon}>
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5">
            <path d="M12 4v16M20 12H4" strokeLinecap="round" />
          </svg>
        </span>
      </button>
    </div>
  );
}
