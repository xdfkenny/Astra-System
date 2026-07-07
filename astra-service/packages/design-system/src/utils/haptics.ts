export type HapticPattern = number | number[] | "light" | "medium" | "heavy";

const patterns: Record<"light" | "medium" | "heavy", number[]> = {
  light: [10],
  medium: [20],
  heavy: [30],
};

export function haptic(pattern: HapticPattern = "medium"): boolean {
  if (
    typeof navigator === "undefined" ||
    !("vibrate" in navigator) ||
    typeof navigator.vibrate !== "function"
  ) {
    return false;
  }

  const resolved =
    typeof pattern === "string"
      ? patterns[pattern]
      : Array.isArray(pattern)
        ? pattern
        : [pattern];

  return navigator.vibrate(resolved);
}
