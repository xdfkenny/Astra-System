// Color tokens for Astra-Service kiosk UI
//
// Retail-adapted Living Weave color system:
// - Optimized for bright retail environments (1000+ lux)
// - Warm, sophisticated palette with biophilic accents
// - Full dark mode support for night shifts
// - WCAG 2.2 AA compliant

export interface ColorTokens {
  // Base & Neutrals
  linen: string;              // #F5F3EF - primary background
  warmCream: string;          // #FEF7E0 - secondary background
  cardSurface: string;        // #FFFFFF at 88% opacity
  charcoal: string;           // #2D2A26 - primary text
  stone: string;              // #6B6862 - secondary text
  taupe: string;              // #C4B8A8 - dividers, hairlines
  clay: string;               // #B8A99A - subtle borders

  // Biophilic Accents
  moss: string;               // #5A7A5C - primary action, success
  amber: string;              // #B87E6B - CTAs, total price highlights
  denim: string;              // #4A5D70 - informational elements
  deepForest: string;         // #1A3A2A - high-emphasis text (dark mode)
  paleMint: string;           // #E8F5E9 - success backgrounds
  softRose: string;           // #C4A4A4 - error states

  // Functional Colors
  offline: string;            // #D4A843 - offline mode banner
  syncActive: string;         // #5A7A5C - P2P mesh active
  printer: string;            // #6B6862 - thermal printer status

  // Opacity levels for overlays
  white75: string;            // bg-white/75
  white60: string;            // bg-white/60
  black20: string;            // bg-charcoal/20

  // Status-specific tints
  successTint: string;        // #E8F5E9
  warningTint: string;        // #FEF7E0
  errorTint: string;          // #F5F3EF with #C4A4A4 border
}

export const colors: ColorTokens = {
  // Base & Neutrals
  linen: '#F5F3EF',
  warmCream: '#FEF7E0',
  cardSurface: 'rgba(255, 255, 255, 0.88)',
  charcoal: '#2D2A26',
  stone: '#6B6862',
  taupe: '#C4B8A8',
  clay: '#B8A99A',

  // Biophilic Accents
  moss: '#5A7A5C',
  amber: '#B87E6B',
  denim: '#4A5D70',
  deepForest: '#1A3A2A',
  paleMint: '#E8F5E9',
  softRose: '#C4A4A4',

  // Functional Colors
  offline: '#D4A843',
  syncActive: '#5A7A5C',
  printer: '#6B6862',

  // Opacity levels
  white75: 'rgba(255, 255, 255, 0.75)',
  white60: 'rgba(255, 255, 255, 0.60)',
  black20: 'rgba(45, 42, 38, 0.20)',

  // Status tints
  successTint: '#E8F5E9',
  warningTint: '#FEF7E0',
  errorTint: '#F5F3EF',
};

// Dark theme overrides
export const darkColors: Partial<ColorTokens> = {
  charcoal: '#F5F3EF',           // Linen in dark mode
  stone: '#A8A49D',              // Lighter stone for dark mode
  taupe: '#6B6862',              // Slightly darker taupe
  clay: '#8B847E',               // Muted clay for dark mode

  deepForest: '#7A9A7C',         // Moss in dark mode
  moss: '#7A9A7C',                // Brightened moss for dark
  paleMint: '#1A3A2A',           // Dark mint background
  softRose: '#8C6E6E',           // Softer rose for dark

  // Dark mode specific
  cardSurface: 'rgba(42, 40, 38, 0.88)',
  white75: 'rgba(42, 40, 38, 0.75)',
  white60: 'rgba(42, 40, 38, 0.60)',
  black20: 'rgba(255, 255, 255, 0.20)',
};

export const cssVariables: Record<string, string> = Object.fromEntries(
  Object.entries(colors).map(([key, value]) => [`--astra-color-${key}`, value]),
);