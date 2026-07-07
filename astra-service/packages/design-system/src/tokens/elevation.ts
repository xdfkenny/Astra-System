export const borderRadius = {
  none: "0px",
  sm: "2px",
  DEFAULT: "4px",
  md: "6px",
  lg: "8px",
  xl: "12px",
  "2xl": "16px",
  full: "9999px",
} as const;

export const boxShadow = {
  none: "0 0 #0000",
  sm: "0 1px 2px 0 rgb(15 23 42 / 0.05)",
  DEFAULT:
    "0 1px 3px 0 rgb(15 23 42 / 0.1), 0 1px 2px -1px rgb(15 23 42 / 0.1)",
  md: "0 4px 6px -1px rgb(15 23 42 / 0.1), 0 2px 4px -2px rgb(15 23 42 / 0.1)",
  lg: "0 10px 15px -3px rgb(15 23 42 / 0.1), 0 4px 6px -4px rgb(15 23 42 / 0.1)",
  xl: "0 20px 25px -5px rgb(15 23 42 / 0.1), 0 8px 10px -6px rgb(15 23 42 / 0.1)",
  "2xl": "0 25px 50px -12px rgb(15 23 42 / 0.25)",
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
