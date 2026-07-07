/**
 * Astra-Service Design Tokens — "Soft Industrial"
 *
 * These are the single source of truth. They are consumed:
 *  1. As TypeScript constants (for Framer Motion values, canvas/CV overlays, chart colors)
 *  2. Re-exported as CSS custom properties in `tokens.css` (consumed by Tailwind v4 `@theme`)
 *
 * WHY duplicate token sources instead of generating CSS from TS at build time?
 * Tailwind v4 reads CSS-native `@theme` blocks directly with zero JS config step,
 * which keeps the kiosk's Vite dev server cold-start under 400ms. We hand-sync
 * the two files and a Vitest snapshot test (`tokens.spec.ts`) fails CI if they drift.
 */

export const color = {
  background: "#F8F9FC",
  surface: "#FFFFFF",
  surfaceSunken: "#F0F2F7",
  primary: "#4F6D7A",
  primaryHover: "#456170",
  primaryPressed: "#3B5560",
  secondary: "#7B8FA1",
  accent: "#E09F3E",
  accentHover: "#CB8E36",
  error: "#C1666B",
  success: "#6A9E88",
  warning: "#E0B23E",
  ink: "#1F2933",
  inkMuted: "#5A6472",
  border: "rgba(31, 41, 51, 0.08)",
  borderStrong: "rgba(31, 41, 51, 0.16)",
  overlay: "rgba(15, 20, 25, 0.55)",
} as const;

export const dark = {
  background: "#0f1419",
  surface: "#1a2026",
  surfaceSunken: "#13181d",
  primary: "#6b8a99",
  primaryHover: "#7a9ba8",
  primaryPressed: "#8aabba",
  secondary: "#5f6f7d",
  accent: "#f0b156",
  accentHover: "#f5c27a",
  error: "#e07d82",
  success: "#8bc4ad",
  warning: "#ecc55d",
  ink: "#e8ecf1",
  inkMuted: "#9aa3ad",
  border: "rgba(232, 236, 241, 0.10)",
  borderStrong: "rgba(232, 236, 241, 0.20)",
  overlay: "rgba(0, 0, 0, 0.65)",
} as const;

export const highContrast = {
  background: "#000000",
  surface: "#0B0B0B",
  primary: "#8FD5EA",
  accent: "#FFD166",
  error: "#FF6B6B",
  success: "#8BE3B8",
  ink: "#FFFFFF",
  border: "rgba(255, 255, 255, 0.32)",
} as const;

/** 8px baseline grid. Every margin/padding/gap in the kiosk UI must resolve to one of these. */
export const spacing = {
  "0": "0px",
  "1": "4px",
  "2": "8px",
  "3": "16px",
  "4": "24px",
  "5": "32px",
  "6": "48px",
  "7": "64px",
  "8": "96px",
  "9": "128px",
} as const;

/** Minimum interactive target is 48px per WCAG 2.2 AA + industrial glove-friendly touch. */
export const touchTarget = {
  minimum: "48px",
  comfortable: "64px",
  primaryAction: "88px",
} as const;

export const radius = {
  sm: "6px",
  md: "12px",
  lg: "20px",
  pill: "999px",
} as const;

export const shadow = {
  // Real, physical-feeling shadows — never a glow/blur that reads as "AI generated gradient".
  sm: "0 1px 2px rgba(31, 41, 51, 0.06), 0 1px 1px rgba(31, 41, 51, 0.04)",
  md: "0 4px 8px rgba(31, 41, 51, 0.08), 0 2px 4px rgba(31, 41, 51, 0.06)",
  lg: "0 12px 24px rgba(31, 41, 51, 0.12), 0 4px 8px rgba(31, 41, 51, 0.08)",
  focusRing: "0 0 0 3px rgba(224, 159, 62, 0.45)",
} as const;

export const typography = {
  fontUi: '"Inter", system-ui, -apple-system, sans-serif',
  fontHeading: '"Space Grotesk", "Inter", system-ui, sans-serif',
  scale: {
    caption: { size: "14px", lineHeight: "20px", weight: 500 },
    body: { size: "18px", lineHeight: "26px", weight: 400 },
    bodyLarge: { size: "22px", lineHeight: "30px", weight: 500 },
    title: { size: "28px", lineHeight: "36px", weight: 600 },
    display: { size: "40px", lineHeight: "48px", weight: 700 },
    hero: { size: "56px", lineHeight: "64px", weight: 700 },
  },
} as const;

/** Framer Motion durations — hardware-accelerated (transform/opacity) only. */
export const motion = {
  durationInstant: 0.08,
  durationFast: 0.16,
  durationBase: 0.24,
  durationSlow: 0.4,
  easeStandard: [0.2, 0.0, 0, 1.0] as const,
  easeEmphasized: [0.05, 0.7, 0.1, 1.0] as const,
};

export const zIndex = {
  base: 0,
  stickyBar: 10,
  floatingCart: 20,
  modal: 40,
  toast: 50,
  attractLoop: 100,
} as const;

/** Grain texture opacity — subtle noise to avoid the sterile "AI slop" flat-gradient look. */
export const texture = {
  grainOpacity: 0.035,
  borderOpacityDefault: 0.08,
  borderOpacityStrong: 0.16,
} as const;

export type ColorToken = keyof typeof color;
export type SpacingToken = keyof typeof spacing;
