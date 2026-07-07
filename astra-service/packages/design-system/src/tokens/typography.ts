export const fontFamily = {
  sans: [
    "system-ui",
    "-apple-system",
    "BlinkMacSystemFont",
    '"Segoe UI"',
    "Roboto",
    '"Helvetica Neue"',
    "Arial",
    "sans-serif",
  ],
  mono: [
    "ui-monospace",
    "SFMono-Regular",
    "Menlo",
    "Monaco",
    '"Cascadia Mono"',
    '"Segoe UI Mono"',
    '"Roboto Mono"',
    "monospace",
  ],
} as const;

export const fontSize = {
  xs: ["12px", "16px"],
  sm: ["14px", "20px"],
  base: ["16px", "24px"],
  lg: ["18px", "28px"],
  xl: ["20px", "28px"],
  "2xl": ["24px", "32px"],
  "3xl": ["30px", "36px"],
  "4xl": ["36px", "40px"],
  "5xl": ["48px", "48px"],
  "6xl": ["60px", "60px"],
} as const;

export const fontWeight = {
  light: 300,
  normal: 400,
  medium: 500,
  semibold: 600,
  bold: 700,
} as const;

export const lineHeight = {
  tight: 1.25,
  snug: 1.375,
  normal: 1.5,
  relaxed: 1.625,
} as const;

export const letterSpacing = {
  tight: "-0.025em",
  normal: "0em",
  wide: "0.025em",
} as const;

type CssValue = string | number | readonly (string | number)[];

function toCssVariables(
  prefix: string,
  record: Record<string, CssValue>,
): Record<string, string> {
  const result: Record<string, string> = {};
  for (const [key, value] of Object.entries(record)) {
    if (Array.isArray(value)) {
      result[`--astra-${prefix}-${key}`] = String(value[0] ?? "");
    } else {
      result[`--astra-${prefix}-${key}`] = String(value);
    }
  }
  return result;
}

export const cssVariables: Record<string, string> = {
  ...toCssVariables("font", fontFamily),
  ...toCssVariables("text", fontSize),
  ...toCssVariables("weight", fontWeight),
  ...toCssVariables("leading", lineHeight),
  ...toCssVariables("tracking", letterSpacing),
};
