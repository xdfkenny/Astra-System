export const duration = {
  0: "0ms",
  75: "75ms",
  100: "100ms",
  150: "150ms",
  200: "200ms",
  300: "300ms",
  500: "500ms",
  700: "700ms",
  1000: "1000ms",
} as const;

export const easing = {
  linear: "linear",
  in: "cubic-bezier(0.4, 0, 1, 1)",
  out: "cubic-bezier(0, 0, 0.2, 1)",
  "in-out": "cubic-bezier(0.4, 0, 0.2, 1)",
  bounce: "cubic-bezier(0.68, -0.55, 0.265, 1.55)",
} as const;

export const cssVariables: Record<string, string> = {
  ...Object.fromEntries(
    Object.entries(duration).map(([key, value]) => [`--astra-duration-${key}`, value]),
  ),
  ...Object.fromEntries(
    Object.entries(easing).map(([key, value]) => [`--astra-ease-${key}`, value]),
  ),
};
