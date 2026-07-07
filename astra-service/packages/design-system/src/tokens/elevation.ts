export const borderRadius = {
  none: "0px",
  sm: "8px",
  md: "12px",
  lg: "16px",
  xl: "24px",
  pill: "9999px",
} as const;

export const boxShadow = {
  none: "0 0 #0000",
  sm: "0 2px 12px rgba(45, 42, 38, 0.06)",
  DEFAULT: "0 2px 12px rgba(45, 42, 38, 0.06)",
  md: "0 4px 24px rgba(45, 42, 38, 0.08)",
  lg: "0 8px 32px rgba(45, 42, 38, 0.12)",
  xl: "0 12px 48px rgba(45, 42, 38, 0.16)",
  focusRing: "0 0 0 3px rgba(90, 122, 92, 0.35)",
  focusRingCta: "0 0 0 3px rgba(184, 126, 107, 0.35)",
} as const;

function variableName(prefix: string, key: string): string {
  return key === "DEFAULT" ? `--astra-${prefix}` : `--astra-${prefix}-${key}`;
}

export const cssVariables: Record<string, string> = {
  ...Object.fromEntries(
    Object.entries(borderRadius).map(([key, value]) => [variableName("radius", key), value]),
  ),
  ...Object.fromEntries(
    Object.entries(boxShadow).map(([key, value]) => [variableName("shadow", key), value]),
  ),
};
