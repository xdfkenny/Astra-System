import {
  createContext,
  useContext,
  useMemo,
  useState,
  type CSSProperties,
  type ReactElement,
  type ReactNode,
} from "react";
import { colors, darkColors } from "../tokens/colors";

export type ThemeMode = "light" | "dark";

interface ThemeContextValue {
  mode: ThemeMode;
  setMode: (mode: ThemeMode) => void;
  toggle: () => void;
}

const ThemeContext = createContext<ThemeContextValue | null>(null);

export interface ThemeProviderProps {
  children: ReactNode;
  defaultMode?: ThemeMode;
}

// Provides runtime theme switching. Resolves the active palette into CSS custom
// properties on a wrapping element so consumers can reference them via the
// design-system token variables.
export default function ThemeProvider({
  children,
  defaultMode = "light",
}: ThemeProviderProps): ReactElement {
  const [mode, setMode] = useState<ThemeMode>(defaultMode);

  const value = useMemo<ThemeContextValue>(
    () => ({
      mode,
      setMode,
      toggle: () => setMode((current) => (current === "light" ? "dark" : "light")),
    }),
    [mode],
  );

  const palette = mode === "dark" ? darkColors : colors;
  const styleVars = Object.fromEntries(
    Object.entries(palette).map(([key, val]) => [`--astra-color-${key}`, val]),
  ) as Record<string, string>;

  return (
    <ThemeContext.Provider value={value}>
      <div data-theme={mode} style={styleVars as CSSProperties}>
        {children}
      </div>
    </ThemeContext.Provider>
  );
}

export function useTheme(): ThemeContextValue {
  const ctx = useContext(ThemeContext);
  if (!ctx) {
    throw new Error("useTheme must be used within a ThemeProvider");
  }
  return ctx;
}
