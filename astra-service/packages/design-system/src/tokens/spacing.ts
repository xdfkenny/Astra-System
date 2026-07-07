export const spacing = {
  0: "0px",
  0.5: "4px",
  1: "8px",
  2: "16px",
  3: "24px",
  4: "32px",
  5: "40px",
  6: "48px",
  7: "56px",
  8: "64px",
  10: "80px",
  12: "96px",
  14: "112px",
  16: "128px",
} as const;

export const touchTarget = {
  minimum: "56px",
  comfortable: "64px",
  primaryAction: "96px",
} as const;

export const cssVariables: Record<string, string> = {
  ...Object.fromEntries(
    Object.entries(spacing).map(([key, value]) => [`--astra-space-${key}`, value]),
  ),
  ...Object.fromEntries(
    Object.entries(touchTarget).map(([key, value]) => [`--astra-touch-${key}`, value]),
  ),
};
