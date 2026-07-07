export const fontFamily = {
  ui: [
    "Inter",
    "system-ui",
    "-apple-system",
    "sans-serif",
  ],
  heading: [
    "Cormorant Garamond",
    "Georgia",
    "serif",
  ],
  mono: [
    "IBM Plex Mono",
    "ui-monospace",
    "monospace",
  ],
} as const;

export const fontSize = {
  micro: ["14px", "20px"],
  caption: ["13px", "18px"],
  body: ["18px", "28px"],
  bodyLarge: ["22px", "30px"],
  title: ["28px", "36px"],
  screenTitle: ["36px", "44px"],
  hero: ["56px", "64px"],
  heroLarge: ["72px", "80px"],
} as const;

export const fontWeight = {
  light: 300,
  normal: 400,
  medium: 500,
  semibold: 600,
  bold: 700,
} as const;

export const lineHeight = {
  tight: 1.1,
  snug: 1.25,
  normal: 1.5,
  relaxed: 1.625,
} as const;

export const letterSpacing = {
  tight: "-0.02em",
  normal: "-0.01em",
  wide: "0.08em",
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
