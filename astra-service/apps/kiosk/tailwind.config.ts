import type { Config } from "tailwindcss";

/**
 * Astra-Service Kiosk — Tailwind theme configuration.
 *
 * Tailwind CSS v4 reads `@theme inline` in `src/styles/global.css` directly,
 * so this file is kept as a typed JS mirror for components that need to read
 * theme values in TypeScript (e.g. Framer Motion, canvas overlays, chart colors).
 *
 * All values must stay in sync with the CSS custom properties in
 * `packages/design-tokens/src/tokens.css`.
 */

export const colors = {
  // Base / Neutrals
  linen: "#F5F3EF",
  warmCream: "#FEF7E0",
  cardSurface: "#FFFFFF",
  charcoal: "#2D2A26",
  stone: "#6B6862",
  taupe: "#C4B8A8",
  clay: "#B8A99A",

  // Biophilic Accents
  moss: "#5A7A5C",
  "moss-hover": "#4A6A4C",
  "moss-pressed": "#3A5A3C",
  amber: "#B87E6B",
  "amber-hover": "#A06E5D",
  "amber-pressed": "#885E4D",
  denim: "#4A5D70",
  "denim-hover": "#3A4D60",
  deepForest: "#1A3A2A",
  paleMint: "#E8F5E9",
  softRose: "#C4A4A4",
  "softRose-hover": "#B08A8A",
  offline: "#D4A843",
  syncActive: "#5A7A5C",
  printer: "#6B6862",

  // Semantic
  background: "#F5F3EF",
  surface: "#FFFFFF",
  "surface-sunken": "#FEF7E0",
  "surface-overlay": "rgba(255, 255, 255, 0.88)",
  primary: "#5A7A5C",
  "primary-hover": "#4A6A4C",
  "primary-pressed": "#3A5A3C",
  cta: "#B87E6B",
  "cta-hover": "#A06E5D",
  "cta-pressed": "#885E4D",
  secondary: "#4A5D70",
  "secondary-hover": "#3A4D60",
  success: "#5A7A5C",
  warning: "#D4A843",
  info: "#4A5D70",
  error: "#C4A4A4",
  "error-hover": "#B08A8A",
  ink: "#2D2A26",
  "ink-muted": "#6B6862",
  "ink-inverted": "#F5F3EF",
  border: "#C4B8A8",
  "border-strong": "#B8A99A",
  divider: "#C4B8A8",
  overlay: "rgba(45, 42, 38, 0.2)",
  "overlay-strong": "rgba(45, 42, 38, 0.55)",
} as const;

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
  micro: ["14px", { lineHeight: "20px" }],
  caption: ["13px", { lineHeight: "18px" }],
  body: ["18px", { lineHeight: "28px" }],
  bodyLarge: ["22px", { lineHeight: "30px" }],
  title: ["28px", { lineHeight: "36px" }],
  screenTitle: ["36px", { lineHeight: "44px" }],
  hero: ["56px", { lineHeight: "64px" }],
  heroLarge: ["72px", { lineHeight: "80px" }],
} as const;

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

export const borderRadius = {
  none: "0px",
  sm: "8px",
  md: "12px",
  lg: "16px",
  xl: "24px",
  pill: "9999px",
} as const;

export const boxShadow = {
  sm: "0 2px 12px rgba(45, 42, 38, 0.06)",
  md: "0 4px 24px rgba(45, 42, 38, 0.08)",
  lg: "0 8px 32px rgba(45, 42, 38, 0.12)",
  xl: "0 12px 48px rgba(45, 42, 38, 0.16)",
  focusRing: "0 0 0 3px rgba(90, 122, 92, 0.35)",
  focusRingCta: "0 0 0 3px rgba(184, 126, 107, 0.35)",
} as const;

export const zIndex = {
  base: "0",
  content: "10",
  stickyBar: "20",
  floatingCart: "20",
  modal: "30",
  bottomSheet: "30",
  toast: "40",
  offlineBanner: "40",
  attractLoop: "50",
  systemOverlay: "60",
} as const;

export default {
  content: ["./src/**/*.{js,ts,jsx,tsx}"],
  theme: {
    extend: {
      colors,
      fontFamily,
      fontSize,
      spacing,
      borderRadius,
      boxShadow,
      zIndex,
    },
  },
  plugins: [],
} satisfies Config;
