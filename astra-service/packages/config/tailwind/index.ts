/**
 * Astra-Service Tailwind preset.
 *
 * The design-system package does not ship a Tailwind preset, so this config
 * defines the same visual tokens directly: slate neutrals, teal-600 primary,
 * amber-500 CTA, and rose-500 error.
 */

const slate = {
  50: "#f8fafc",
  100: "#f1f5f9",
  200: "#e2e8f0",
  300: "#cbd5e1",
  400: "#94a3b8",
  500: "#64748b",
  600: "#475569",
  700: "#334155",
  800: "#1e293b",
  900: "#0f172a",
  950: "#020617",
} as const;

const teal = {
  50: "#f0fdfa",
  100: "#ccfbf1",
  200: "#99f6e4",
  300: "#5eead4",
  400: "#2dd4bf",
  500: "#14b8a6",
  600: "#0d9488",
  700: "#0f766e",
  800: "#115e59",
  900: "#134e4a",
  950: "#042f2e",
} as const;

const amber = {
  50: "#fffbeb",
  100: "#fef3c7",
  200: "#fde68a",
  300: "#fcd34d",
  400: "#fbbf24",
  500: "#f59e0b",
  600: "#d97706",
  700: "#b45309",
  800: "#92400e",
  900: "#78350f",
  950: "#451a03",
} as const;

const rose = {
  50: "#fff1f2",
  100: "#ffe4e6",
  200: "#fecdd3",
  300: "#fda4af",
  400: "#fb7185",
  500: "#f43f5e",
  600: "#e11d48",
  700: "#be123c",
  800: "#9f1239",
  900: "#881337",
  950: "#4c0519",
} as const;

const preset = {
  darkMode: "class",
  theme: {
    extend: {
      colors: {
        slate,
        teal,
        amber,
        rose,
        primary: {
          DEFAULT: teal[600],
          hover: teal[700],
          pressed: teal[800],
        },
        cta: {
          DEFAULT: amber[500],
          hover: amber[600],
        },
        error: {
          DEFAULT: rose[500],
          hover: rose[600],
        },
        background: slate[50],
        surface: "#ffffff",
        "surface-sunken": slate[100],
        border: slate[200],
        "border-strong": slate[300],
        ink: slate[950],
        "ink-muted": slate[500],
        overlay: "rgba(15, 23, 42, 0.48)",
      },
      fontFamily: {
        sans: [
          '"Inter"',
          "system-ui",
          "-apple-system",
          "BlinkMacSystemFont",
          '"Segoe UI"',
          "sans-serif",
        ].join(", "),
        heading: ['"Space Grotesk"', '"Inter"', "system-ui", "sans-serif"].join(", "),
      },
      spacing: Object.fromEntries(
        Array.from({ length: 25 }, (_, i) => [String(i), `${i * 0.25}rem`]),
      ),
      borderRadius: {
        none: "0px",
        sm: "4px",
        md: "8px",
        lg: "12px",
        xl: "16px",
        pill: "9999px",
      },
      boxShadow: {
        sm: "0 1px 2px 0 rgb(15 23 42 / 0.05), 0 1px 1px -1px rgb(15 23 42 / 0.04)",
        md: "0 4px 6px -1px rgb(15 23 42 / 0.08), 0 2px 4px -2px rgb(15 23 42 / 0.04)",
        lg: "0 10px 15px -3px rgb(15 23 42 / 0.1), 0 4px 6px -4px rgb(15 23 42 / 0.05)",
        xl: "0 20px 25px -5px rgb(15 23 42 / 0.12), 0 8px 10px -6px rgb(15 23 42 / 0.04)",
        focus: "0 0 0 3px rgb(13 148 136 / 0.35)",
      },
      zIndex: {
        base: "0",
        dropdown: "10",
        sticky: "20",
        drawer: "30",
        modal: "40",
        toast: "50",
        tooltip: "60",
        attractLoop: "100",
      },
    },
  },
};

export default preset;
export { slate, teal, amber, rose };
