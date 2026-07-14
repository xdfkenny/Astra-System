// Spacing scale for Astra-Service kiosk UI
//
// 8px-based grid system optimized for 9:16 vertical displays
// Touch-accessible with minimum 56px targets
// Retail-optimized for legibility and comfort
export interface SpacingScale {
  // Status Bar (persistent top, 48px height)
  statusBarHeight: string;
  statusBarPadding: { vertical: string; horizontal: string; };

  // Bottom Action Bar (persistent, 96px height)
  actionBarHeight: string;
  actionBarPadding: { vertical: string; horizontal: string; };

  // Safe areas
  safeAreaHorizontal: string;
  safeAreaTop: string;
  safeAreaBottom: string;

  // Component padding
  componentPadding: {
    xs: string;   // 8px
    sm: string;   // 16px
    md: string;   // 24px
    lg: string;   // 32px
    xl: string;   // 40px
    xl2: string;  // 48px
  };

  // Component margin
  componentMargin: {
    xs: string;
    sm: string;
    md: string;
    lg: string;
    xl: string;
    xl2: string;
  };

  // Component radius
  radius: {
    none: string;
    sm: string;   // 8px
    md: string;   // 16px
    lg: string;   // 24px
    xl: string;   // 32px
    full: string; // 9999px
  };

  // Touch targets
  touchMinHeight: string;
  touchMinWidth: string;

  // List spacing
  list: {
    itemSpacing: string;
    groupSpacing: string;
    headerSpacing: string;
  };

  // Grid columns
  grid: {
    gap: string;
    columns: number;
  };
}

export const spacing: SpacingScale = {
  // Status Bar (48px height)
  statusBarHeight: '48px',
  statusBarPadding: {
    vertical: '8px',
    horizontal: '24px',
  },

  // Bottom Action Bar (96px height)
  actionBarHeight: '96px',
  actionBarPadding: {
    vertical: '16px',
    horizontal: '24px',
  },

  // Safe areas based on 1080x1920 (3x scale)
  safeAreaHorizontal: '24px',
  safeAreaTop: '32px',
  safeAreaBottom: '40px',

  // Component padding (8px grid)
  componentPadding: {
    xs: '8px',
    sm: '16px',
    md: '24px',
    lg: '32px',
    xl: '40px',
    xl2: '48px',
  },

  // Component margin
  componentMargin: {
    xs: '8px',
    sm: '16px',
    md: '24px',
    lg: '32px',
    xl: '40px',
    xl2: '48px',
  },

  // Component radius
  radius: {
    none: '0',
    sm: '8px',
    md: '16px',
    lg: '24px',
    xl: '32px',
    full: '9999px',
  },

  // Touch targets (WCAG 2.2 AA minimum 56px)
  touchMinHeight: '56px',
  touchMinWidth: '56px',

  // List spacing
  list: {
    itemSpacing: '16px',
    groupSpacing: '32px',
    headerSpacing: '8px',
  },

  // Grid columns (4-column grid with 16px gutters)
  grid: {
    gap: '16px',
    columns: 4,
  },
};

function toVars(key: string, value: unknown): [string, string][] {
  if (typeof value === "string") {
    return [[`--astra-space-${key}`, value]];
  }
  if (value && typeof value === "object") {
    return Object.entries(value as Record<string, string>).map(
      ([sub, subValue]) => [`--astra-space-${key}-${sub}`, String(subValue)],
    );
  }
  return [[`--astra-space-${key}`, String(value)]];
}

export const cssVariables: Record<string, string> = Object.fromEntries(
  Object.entries(spacing).flatMap(([key, value]) => toVars(key, value)),
);

export default spacing;
