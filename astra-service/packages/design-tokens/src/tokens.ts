/**
 * Astra-Service Design Tokens — "Living Weave" (Retail Kiosk)
 *
 * Biophilic, wabi-sabi inspired palette adapted for calm retail efficiency
 * on 9:16 vertical touchscreens in bright, variable lighting.
 *
 * These are the single source of truth. They are consumed:
 *  1. As TypeScript constants (for Framer Motion values, canvas/CV overlays, chart colors)
 *  2. Re-exported as CSS custom properties in `tokens.css` (consumed by Tailwind v4 `@theme`)
 */

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

export const color = {
  // Surfaces
  background: base.linen,
  surface: base.cardSurface,
  surfaceSunken: base.warmCream,
  surfaceOverlay: "rgba(255, 255, 255, 0.88)",

  // Semantic actions
  primary: accent.moss,
  primaryHover: accent.mossHover,
  primaryPressed: accent.mossPressed,
  cta: accent.amber,
  ctaHover: accent.amberHover,
  ctaPressed: accent.amberPressed,
  secondary: accent.denim,
  secondaryHover: accent.denimHover,

  // Status
  success: accent.moss,
  warning: accent.offline,
  info: accent.denim,
  error: accent.softRose,
  errorHover: accent.softRoseHover,
  offline: accent.offline,
  syncActive: accent.syncActive,

  // Typography
  ink: base.charcoal,
  inkMuted: base.stone,
  inkInverted: base.linen,

  // Borders
  border: base.taupe,
  borderStrong: base.clay,
  divider: base.taupe,
  overlay: "rgba(45, 42, 38, 0.2)",
  overlayStrong: "rgba(45, 42, 38, 0.55)",
} as const;

export const dark = {
  background: "#1C1A17",
  surface: "#2A2824",
  surfaceSunken: "#1C1A17",
  surfaceOverlay: "rgba(42, 40, 36, 0.92)",

  primary: "#7A9A7C",
  primaryHover: "#8AAB8C",
  primaryPressed: "#9ABC9D",
  cta: "#C49A8A",
  ctaHover: "#D4AA9A",
  ctaPressed: "#E4BAAA",
  secondary: "#6A7D90",
  secondaryHover: "#7A8DA0",

  success: "#7A9A7C",
  warning: "#E4C46A",
  info: "#6A7D90",
  error: "#D4A4A4",
  errorHover: "#E4B4B4",
  offline: "#E4C46A",
  syncActive: "#7A9A7C",

  ink: "#F5F3EF",
  inkMuted: "#A8A49D",
  inkInverted: "#2D2A26",

  border: "rgba(168, 164, 157, 0.2)",
  borderStrong: "rgba(168, 164, 157, 0.32)",
  divider: "rgba(168, 164, 157, 0.2)",
  overlay: "rgba(0, 0, 0, 0.55)",
  overlayStrong: "rgba(0, 0, 0, 0.72)",
} as const;

export const highContrast = {
  background: "#000000",
  surface: "#0B0B0B",
  surfaceSunken: "#000000",
  surfaceOverlay: "rgba(0, 0, 0, 0.96)",

  primary: "#8FD5EA",
  primaryHover: "#9EE5FA",
  cta: "#FFD166",
  ctaHover: "#FFE18A",
  secondary: "#8FD5EA",
  secondaryHover: "#9EE5FA",

  success: "#8BE3B8",
  warning: "#FFD166",
  info: "#8FD5EA",
  error: "#FF6B6B",
  errorHover: "#FF8B8B",
  offline: "#FFD166",
  syncActive: "#8BE3B8",

  ink: "#FFFFFF",
  inkMuted: "#E0E0E0",
  inkInverted: "#000000",

  border: "rgba(255, 255, 255, 0.32)",
  borderStrong: "rgba(255, 255, 255, 0.48)",
  divider: "rgba(255, 255, 255, 0.32)",
  overlay: "rgba(0, 0, 0, 0.72)",
  overlayStrong: "rgba(0, 0, 0, 0.88)",
} as const;

/** 8px baseline grid. Every margin/padding/gap in the kiosk UI must resolve to one of these. */
export const spacing = {
  "0": "0px",
  "0.5": "4px",
  "1": "8px",
  "2": "16px",
  "3": "24px",
  "4": "32px",
  "5": "40px",
  "6": "48px",
  "7": "56px",
  "8": "64px",
  "10": "80px",
  "12": "96px",
  "14": "112px",
  "16": "128px",
} as const;

/** Minimum interactive target is 56px per WCAG 2.2 AA + kiosk glove-friendly touch. */
export const touchTarget = {
  minimum: "56px",
  comfortable: "64px",
  primaryAction: "96px",
} as const;

export const radius = {
  none: "0px",
  sm: "8px",
  md: "12px",
  lg: "16px",
  xl: "24px",
  pill: "9999px",
} as const;

export const shadow = {
  sm: "0 2px 12px rgba(45, 42, 38, 0.06)",
  md: "0 4px 24px rgba(45, 42, 38, 0.08)",
  lg: "0 8px 32px rgba(45, 42, 38, 0.12)",
  xl: "0 12px 48px rgba(45, 42, 38, 0.16)",
  focusRing: "0 0 0 3px rgba(90, 122, 92, 0.35)",
  focusRingCta: "0 0 0 3px rgba(184, 126, 107, 0.35)",
} as const;

export const typography = {
  fontUi: '"Inter", system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, "Helvetica Neue", Arial, sans-serif',
  fontHeading: '"Cormorant Garamond", Georgia, "Times New Roman", serif',
  fontMono: '"IBM Plex Mono", ui-monospace, SFMono-Regular, Menlo, Monaco, "Cascadia Mono", "Segoe UI Mono", "Roboto Mono", monospace',
  scale: {
    micro: { size: "14px", lineHeight: "20px", weight: 400 },
    caption: { size: "13px", lineHeight: "18px", weight: 500, letterSpacing: "0.08em", uppercase: true },
    body: { size: "18px", lineHeight: "28px", weight: 400 },
    bodyLarge: { size: "22px", lineHeight: "30px", weight: 500 },
    title: { size: "28px", lineHeight: "36px", weight: 600 },
    screenTitle: { size: "36px", lineHeight: "44px", weight: 500 },
    hero: { size: "56px", lineHeight: "64px", weight: 600 },
    heroLarge: { size: "72px", lineHeight: "80px", weight: 600 },
  },
} as const;

/** Framer Motion durations — hardware-accelerated (transform/opacity) only. */
export const motion = {
  durationInstant: 0.08,
  durationFast: 0.15,
  durationBase: 0.25,
  durationSlow: 0.35,
  durationSheet: 0.3,
  durationPage: 0.3,
  easeOutExpo: [0.16, 1, 0.3, 1] as const,
  easeInOutSoft: [0.4, 0, 0.2, 1] as const,
  easeSpring: [0.34, 1.56, 0.64, 1] as const,
  easeStandard: [0.2, 0, 0, 1] as const,
  easeEmphasized: [0.05, 0.7, 0.1, 1] as const,
};

export const zIndex = {
  base: 0,
  content: 10,
  stickyBar: 20,
  floatingCart: 20,
  modal: 30,
  bottomSheet: 30,
  toast: 40,
  offlineBanner: 40,
  attractLoop: 50,
  systemOverlay: 60,
} as const;

/** Texture / materiality values for the performance-first weave look. */
export const texture = {
  grainOpacity: 0.03,
  weaveOpacity: 0.04,
  borderOpacityDefault: 0.12,
  borderOpacityStrong: 0.2,
  cardOpacity: 0.85,
} as const;

export type ColorToken = keyof typeof color;
export type SpacingToken = keyof typeof spacing;
