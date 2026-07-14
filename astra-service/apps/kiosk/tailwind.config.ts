"""Tailwind CSS 4 configuration for Astra-Service kiosk UI

Design tokens based on "Living Weave" biophilic specification
Retail-adapted for 1000+ lux legibility requirements
Exported as CSS variables for runtime theme switching
"""

export default {
  content: ['./src/**/*.{ts,tsx,js,jsx}'],
  theme: {
    // Color system based on Living Weave spec
    colors: {
      transparent: 'transparent',
      current: 'currentColor',

      // Base & Neutrals
      linen: '#F5F3EF',           // background
      warmCream: '#FEF7E0',       // secondary background
      cardSurface: 'rgba(255, 255, 255, 0.88)', // glass effect
      charcoal: '#2D2A26',        // primary text
      stone: '#6B6862',           // secondary text
      taupe: '#C4B8A8',           // borders, hairlines
      clay: '#B8A99A',            // subtle borders

      // Biophilic Accents
      moss: '#5A7A5C',            // primary action
      amber: '#B87E6B',           // CTAs, total price
      denim: '#4A5D70',           // informational
      deepForest: '#1A3A2A',      // high-emphasis text (dark)
      paleMint: '#E8F5E9',       // success backgrounds
      softRose: '#C4A4A4',        // error states

      // Functional Colors
      offline: '#D4A843',         // offline banner
      syncActive: '#5A7A5C',      // P2P mesh active
      printer: '#6B6862',         // printer status

      // Additional colors from design system
      'bg-linen': '{colors.linen}',
      'bg-warm-cream': '{colors.warmCream}',
      'bg-card-surface': '{colors.cardSurface}',
      'text-charcoal': '{colors.charcoal}',
      'text-stone': '{colors.stone}',
      'text-clay': '{colors.clay}',
      'text-moss': '{colors.moss}',
      'text-amber': '{colors.amber}',
      'text-denim': '{colors.denim}',
      'border-taupe': '{colors.taupe}',
      'border-clay': '{colors.clay}',
      'border-stone': '{colors.stone}',
      'border-moss': '{colors.moss}',
      'border-amber': '{colors.amber}',
      'border-denim': '{colors.denim}',
    },

    extend: {
      backgroundColor: {
        // Background layers
        linen: '{colors.linen}',
        warmCream: '{colors.warmCream}',
        'card-surface': '{colors.cardSurface}',
        white: '{colors.white}',

        // Status backgrounds
        'success-light': '#E8F5E9',
        'warning-light': '#FEF7E0',
        'error-light': '#F5F3EF',

        // Dark mode variants
        'linen-dark': '{colors.linen}',
        'card-surface-dark': 'rgba(42, 40, 38, 0.88)',
      },

      textColor: {
        // Primary text
        charcoal: '{colors.charcoal}',
        stone: '{colors.stone}',
        clay: '{colors.clay}',

        // Accent text
        moss: '{colors.moss}',
        amber: '{colors.amber}',
        denim: '{colors.denim}',
        deepForest: '{colors.deepForest}',

        // Status colors
        success: '{colors.moss}',
        warning: '{colors.amber}',
        error: '{colors.softRose}',
        offline: '{colors.offline}',
        sync: '{colors.syncActive}',
        printer: '{colors.printer}',
      },

      borderColor: {
        // Border system
        taupe: '{colors.taupe}',
        clay: '{colors.clay}',
        stone: '{colors.stone}',
        moss: '{colors.moss}',
        amber: '{colors.amber}',
        denim: '{colors.denim}',
      },

      // Text utilities
      fontFamily: {
        cormorant: ['"Cormorant Garamond"', 'Georgia', 'serif'],
        inter: ['"Inter"', 'system-ui', '-apple-system', 'sans-serif'],
        mono: ['"IBM Plex Mono"', 'Menlo', 'Monaco', 'monospace'],
      },

      // Font sizes
      fontSize: {
        '3xl': ['42px', '1.14'],
        '2xl': ['28px', '1.21'],
        'xl': ['24px', '1.33'],
        'lg': ['20px', '1.5'],
        'base': ['18px', '1.5'],
        'sm': ['16px', '1.5'],
        'xs': ['14px', '1.43'],
      },

      // Shadow system
      boxShadow: {
        card: '0 2px 12px rgba(45, 42, 38, 0.06)',
        modal: '0 8px 32px rgba(45, 42, 38, 0.12)',
        dropdown: '0 4px 24px rgba(45, 42, 38, 0.08)',
      },

      // Border radius
      borderRadius: {
        none: '0',
        sm: '8px',
        md: '16px',
        lg: '24px',
        xl: '32px',
        '2xl': '40px',
        '3xl': '48px',
        full: '9999px',
      },

      // Animation timing
      transitionTimingFunction: {
        default: 'cubic-bezier(0.16, 1, 0.3, 1)', // ease-out-expo
        smooth: 'cubic-bezier(0.4, 0, 0.2, 1)', // ease-in-out-soft
        spring: 'cubic-bezier(0.34, 1.56, 0.64, 1)', // gentle spring
      },
    },
  },

  // Tailwind CSS 4 plugin configuration
  plugins: [],

  // Color mode configuration for dark mode support
  darkMode: 'class',

  // Custom variants for accessibility
  variants: {
    extend: {
      screenReaderOnly: ['focus'],
      visuallyHidden: ['focus'],
      srOnly: ['focus'],
    },
  },
};
