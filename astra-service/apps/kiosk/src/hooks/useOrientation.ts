import { useSyncExternalStore } from "react";

export type Orientation = "portrait" | "landscape";

export interface ScreenDimensions {
  readonly width: number;
  readonly height: number;
}

export interface OrientationState {
  readonly orientation: Orientation;
  readonly dimensions: ScreenDimensions;
  readonly isPortrait: boolean;
  readonly isLandscape: boolean;
}

/**
 * Reactive orientation + dimension tracker.
 *
 * Uses `useSyncExternalStore` (React 18+) to subscribe to browser resize
 * and orientationchange events without stale closures. The cached snapshot
 * ensures React skips re-render when dimensions haven't changed.
 *
 * Portrait is defined as `height >= width` (square screens are treated
 * as portrait for kiosk safety).
 */
let cached: OrientationState | null = null;

function getSnapshot(): OrientationState {
  const w = window.innerWidth;
  const h = window.innerHeight;

  if (cached?.dimensions.width === w && cached.dimensions.height === h) {
    return cached;
  }

  const orientation: Orientation = h >= w ? "portrait" : "landscape";

  cached = {
    orientation,
    dimensions: { width: w, height: h },
    isPortrait: orientation === "portrait",
    isLandscape: orientation === "landscape",
  };

  return cached;
}

function getServerSnapshot(): OrientationState {
  return {
    orientation: "portrait",
    dimensions: { width: 1080, height: 1920 },
    isPortrait: true,
    isLandscape: false,
  };
}

function subscribe(callback: () => void): () => void {
  const ro = new ResizeObserver(() => { callback(); });
  try {
    ro.observe(document.documentElement);
  } catch {
    // document not ready — swallow silently (SSR / edge)
  }
  window.addEventListener("resize", callback);
  window.addEventListener("orientationchange", callback);

  return () => {
    ro.disconnect();
    window.removeEventListener("resize", callback);
    window.removeEventListener("orientationchange", callback);
  };
}

export function useOrientation(): OrientationState {
  return useSyncExternalStore(subscribe, getSnapshot, getServerSnapshot);
}
