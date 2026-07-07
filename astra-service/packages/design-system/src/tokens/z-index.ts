export const zIndex = {
  auto: "auto",
  0: 0,
  10: 10,
  20: 20,
  30: 30,
  40: 40,
  50: 50,
  100: 100,
  200: 200,
  300: 300,
  400: 400,
  500: 500,
  top: 9999,
} as const;

export const cssVariables: Record<string, string> = Object.fromEntries(
  Object.entries(zIndex).map(([key, value]) => [`--astra-z-${key}`, String(value)]),
);
