export const slate = {
  50: "#f8fafc",
  100: "#f1f5f9",
  200: "#e2e8f0",
  300: "#cbd5e1",
  400: "#94a3b8",
  500: "#64748b",
  600: "#475569",
  700: "#334155",
  800: "#1e293b",
  900: "#0f172a",
  950: "#020617",
} as const;

export const semantic = {
  primary: "#0d9488",
  "primary-hover": "#0f766e",
  cta: "#f59e0b",
  "cta-hover": "#d97706",
  error: "#f43f5e",
  "error-hover": "#e11d48",
  success: "#10b981",
  warning: "#f59e0b",
  info: "#0ea5e9",
  background: slate[50],
  surface: "#ffffff",
  "surface-elevated": "#ffffff",
  "text-primary": slate[900],
  "text-secondary": slate[600],
  "text-disabled": slate[400],
  border: slate[200],
} as const;

type StringMap = Record<string, string>;

function prefixKeys(record: StringMap, prefix: string): StringMap {
  const result: StringMap = {};
  for (const [key, value] of Object.entries(record)) {
    result[`${prefix}-${key}`] = value;
  }
  return result;
}

export const flatColors: StringMap = {
  ...prefixKeys(slate, "slate"),
  ...semantic,
};

export const cssVariables: StringMap = Object.fromEntries(
  Object.entries(flatColors).map(([key, value]) => [`--astra-color-${key}`, value]),
);
