// Typography scale for Astra-Service kiosk UI
//
// Distance-optimized scale for 9:16 vertical displays (1080x1920, 1440x2560)
// Font stack optimized for retail legibility (1000+ lux environments)
import { colors } from './colors';

export interface TypographyScale {
  // Display / Headings
  heroTitle: {
    fontFamily: string;
    fontWeight: number;
    fontSize: string;
    lineHeight: number;
    letterSpacing: string;
    color: string;
  };
  screenTitle: {
    fontFamily: string;
    fontWeight: number;
    fontSize: string;
    lineHeight: number;
    letterSpacing: string;
    color: string;
    textAlign?: 'left' | 'center' | 'right';
  };
  sectionHeader: {
    fontFamily: string;
    fontWeight: number;
    fontSize: string;
    lineHeight: number;
    letterSpacing: string;
    color: string;
  };

  // UI / Body
  body: {
    fontFamily: string;
    fontWeight: number;
    fontSize: string;
    lineHeight: number;
    letterSpacing: string;
    color: string;
  };
  label: {
    fontFamily: string;
    fontWeight: number;
    fontSize: string;
    lineHeight: number;
    letterSpacing: string;
    color: string;
    textTransform?: 'uppercase' | 'lowercase' | 'capitalize' | 'none';
  };

  // Prices & Data
  price: {
    fontFamily: string;
    fontWeight: number;
    fontSize: string;
    lineHeight: number;
    letterSpacing: string;
    color: string;
    fontVariantNumeric?: string;
  };
  totalPrice: {
    fontFamily: string;
    fontWeight: number;
    fontSize: string;
    lineHeight: number;
    letterSpacing: string;
    color: string;
    fontVariantNumeric?: string;
  };

  // Micro / Caption
  micro: {
    fontFamily: string;
    fontWeight: number;
    fontSize: string;
    lineHeight: number;
    letterSpacing: string;
    color: string;
  };

  // Monospace for data
  monospace: {
    fontFamily: string;
    fontWeight: number;
    fontSize: string;
    lineHeight: number;
    letterSpacing: string;
    color: string;
  };
}

const typography: TypographyScale = {
  // Display / Headings - Cormorant Garamond
  heroTitle: {
    fontFamily: '\"Cormorant Garamond\", Georgia, serif',
    fontWeight: 600,
    fontSize: '56px', // 1080px width
    lineHeight: 1.1,
    letterSpacing: '-0.02em',
    color: colors.charcoal,
  },
  screenTitle: {
    fontFamily: '\"Inter\", system-ui, -apple-system, sans-serif',
    fontWeight: 500,
    fontSize: '36px',
    lineHeight: 1.11,
    letterSpacing: '-0.01em',
    color: colors.charcoal,
    textAlign: 'left',
  },
  sectionHeader: {
    fontFamily: '\"Inter\", system-ui, -apple-system, sans-serif',
    fontWeight: 500,
    fontSize: '24px',
    lineHeight: 1.33,
    letterSpacing: '-0.01em',
    color: colors.charcoal,
  },

  // UI / Body - Inter
  body: {
    fontFamily: '\"Inter\", system-ui, -apple-system, sans-serif',
    fontWeight: 400,
    fontSize: '18px',
    lineHeight: 1.5,
    letterSpacing: '-0.01em',
    color: colors.charcoal,
  },
  label: {
    fontFamily: '\"Inter\", system-ui, -apple-system, sans-serif',
    fontWeight: 500,
    fontSize: '13px',
    lineHeight: 1.23,
    letterSpacing: '0.08em',
    color: colors.stone,
    textTransform: 'uppercase',
  },

  // Prices & Data
  price: {
    fontFamily: '\"Inter\", system-ui, -apple-system, sans-serif',
    fontWeight: 600,
    fontSize: '28px',
    lineHeight: 1.21,
    letterSpacing: '-0.01em',
    color: colors.charcoal,
    fontVariantNumeric: 'tabular-nums',
  },
  totalPrice: {
    fontFamily: '\"Inter\", system-ui, -apple-system, sans-serif',
    fontWeight: 600,
    fontSize: '42px',
    lineHeight: 1.14,
    letterSpacing: '-0.01em',
    color: colors.amber,
    fontVariantNumeric: 'tabular-nums',
  },

  // Micro / Caption
  micro: {
    fontFamily: '\"Inter\", system-ui, -apple-system, sans-serif',
    fontWeight: 400,
    fontSize: '14px',
    lineHeight: 1.43,
    letterSpacing: '-0.01em',
    color: colors.stone,
  },

  // Monospace for data - IBM Plex Mono
  monospace: {
    fontFamily: '\"IBM Plex Mono\", Menlo, Monaco, monospace',
    fontWeight: 400,
    fontSize: '14px',
    lineHeight: 1.4,
    letterSpacing: '0em',
    color: colors.charcoal,
  },
};

export const cssVariables: Record<string, string> = Object.fromEntries(
  Object.entries(typography).flatMap(([group, props]) =>
    Object.entries(props as Record<string, string>).map(
      ([prop, val]) => [`--astra-font-${group}-${prop}`, String(val)],
    ),
  ),
);

export default typography;
