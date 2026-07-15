import { createContext, useContext, type PropsWithChildren, useMemo } from "react";
import { useOrientation, type Orientation, type ScreenDimensions } from "../hooks/useOrientation";

const LOGICAL_WIDTH = 1080;

export interface ResponsiveContextValue {
  readonly orientation: Orientation;
  readonly dimensions: ScreenDimensions;
  /** Scale factor mapping logical 1080px design width to actual viewport width. */
  readonly scale: number;
  readonly isPortrait: boolean;
  readonly isLandscape: boolean;
}

const ResponsiveContext = createContext<ResponsiveContextValue | null>(null);

/**
 * Provides a single source of truth for all responsive/sizing logic.
 *
 * Every descendant reads from this context via `useResponsive()` so there is
 * exactly one `ResizeObserver` + one set of event listeners for the whole app.
 */
export function ResponsiveProvider({ children }: PropsWithChildren): React.JSX.Element {
  const { orientation, dimensions, isPortrait, isLandscape } = useOrientation();

  const value = useMemo<ResponsiveContextValue>(
    () => ({
      orientation,
      dimensions,
      scale: dimensions.width / LOGICAL_WIDTH,
      isPortrait,
      isLandscape,
    }),
    [orientation, dimensions, isPortrait, isLandscape],
  );

  return <ResponsiveContext.Provider value={value}>{children}</ResponsiveContext.Provider>;
}

export function useResponsive(): ResponsiveContextValue {
  const ctx = useContext(ResponsiveContext);
  if (!ctx) {
    throw new Error("useResponsive must be used within a <ResponsiveProvider>");
  }
  return ctx;
}
