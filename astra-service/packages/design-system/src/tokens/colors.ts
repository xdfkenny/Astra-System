export const base = {
  linen: "#F5F3EF",
  warmCream: "#FEF7E0",
  cardSurface: "#FFFFFF",
  charcoal: "#2D2A26",
  stone: "#6B6862",
  taupe: "#C4B8A8",
  clay: "#B8A99A",
} as const;

export const accent = {
  moss: "#5A7A5C",
  mossHover: "#4A6A4C",
  mossPressed: "#3A5A3C",
  amber: "#B87E6B",
  amberHover: "#A06E5D",
  amberPressed: "#885E4D",
  denim: "#4A5D70",
  denimHover: "#3A4D60",
  deepForest: "#1A3A2A",
  paleMint: "#E8F5E9",
  softRose: "#C4A4A4",
  softRoseHover: "#B08A8A",
  offline: "#D4A843",
  syncActive: "#5A7A5C",
  printer: "#6B6862",
} as const;

export const semantic = {
  background: base.linen,
  surface: base.cardSurface,
  surfaceSunken: base.warmCream,
  surfaceOverlay: "rgba(255, 255, 255, 0.88)",
  primary: accent.moss,
  "primary-hover": accent.mossHover,
  "primary-pressed": accent.mossPressed,
  cta: accent.amber,
  "cta-hover": accent.amberHover,
  "cta-pressed": accent.amberPressed,
  secondary: accent.denim,
  "secondary-hover": accent.denimHover,
  success: accent.moss,
  warning: accent.offline,
  info: accent.denim,
  error: accent.softRose,
  "error-hover": accent.softRoseHover,
  ink: base.charcoal,
  "ink-muted": base.stone,
  "ink-inverted": base.linen,
  border: base.taupe,
  "border-strong": base.clay,
  divider: base.taupe,
  overlay: "rgba(45, 42, 38, 0.2)",
  "overlay-strong": "rgba(45, 42, 38, 0.55)",
} as const;

export const dark = {
  background: "#1C1A17",
  surface: "#2A2824",
  surfaceSunken: "#1C1A17",
  surfaceOverlay: "rgba(42, 40, 36, 0.92)",
  primary: "#7A9A7C",
  "primary-hover": "#8AAB8C",
  "primary-pressed": "#9ABC9D",
  cta: "#C49A8A",
  "cta-hover": "#D4AA9A",
  "cta-pressed": "#E4BAAA",
  secondary: "#6A7D90",
  "secondary-hover": "#7A8DA0",
  success: "#7A9A7C",
  warning: "#E4C46A",
  info: "#6A7D90",
  error: "#D4A4A4",
  "error-hover": "#E4B4B4",
  ink: "#F5F3EF",
  "ink-muted": "#A8A49D",
  "ink-inverted": "#2D2A26",
  border: "rgba(168, 164, 157, 0.2)",
  "border-strong": "rgba(168, 164, 157, 0.32)",
  divider: "rgba(168, 164, 157, 0.2)",
  overlay: "rgba(0, 0, 0, 0.55)",
  "overlay-strong": "rgba(0, 0, 0, 0.72)",
} as const;

export const highContrast = {
  background: "#000000",
  surface: "#0B0B0B",
  surfaceSunken: "#000000",
  surfaceOverlay: "rgba(0, 0, 0, 0.96)",
  primary: "#8FD5EA",
  "primary-hover": "#9EE5FA",
  cta: "#FFD166",
  "cta-hover": "#FFE18A",
  secondary: "#8FD5EA",
  "secondary-hover": "#9EE5FA",
  success: "#8BE3B8",
  warning: "#FFD166",
  info: "#8FD5EA",
  error: "#FF6B6B",
  "error-hover": "#FF8B8B",
  ink: "#FFFFFF",
  "ink-muted": "#E0E0E0",
  "ink-inverted": "#000000",
  border: "rgba(255, 255, 255, 0.32)",
  "border-strong": "rgba(255, 255, 255, 0.48)",
  divider: "rgba(255, 255, 255, 0.32)",
  overlay: "rgba(0, 0, 0, 0.72)",
  "overlay-strong": "rgba(0, 0, 0, 0.88)",
} as const;

type StringMap = Record<string, string>;

function prefixKeys(record: StringMap, prefix: string): StringMap {
  const result: StringMap = {};
  for (const [key, value] of Object.entries(record)) {
    result[prefix ? `${prefix}-${key}` : key] = value;
  }
  return result;
}

export const flatColors: StringMap = {
  ...prefixKeys(base, ""),
  ...prefixKeys(accent, ""),
  ...prefixKeys(semantic, ""),
};

export const cssVariables: StringMap = Object.fromEntries(
  Object.entries(flatColors).map(([key, value]) => [`--astra-color-${key}`, value]),
);

export type BaseColorToken = keyof typeof base;
export type AccentColorToken = keyof typeof accent;
export type SemanticColorToken = keyof typeof semantic;
export type DarkColorToken = keyof typeof dark;
export type HighContrastColorToken = keyof typeof highContrast;
