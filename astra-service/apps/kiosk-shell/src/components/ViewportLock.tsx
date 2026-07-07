import { type PropsWithChildren, useEffect, useState } from "react";

/**
 * Enforces the 9:16 vertical kiosk viewport regardless of the underlying
 * panel's native resolution (1080x1920 or 1440x2560 target hardware).
 *
 * WHY a JS-measured scale transform instead of pure CSS `aspect-ratio`:
 * the industrial touch panels this ships to report inconsistent
 * `window.innerHeight` after thermal-printer USB hot-plug events (observed
 * on Verifone-certified units — the browser recalculates safe-area insets).
 * Recomputing a fixed 1080x1920 logical canvas and CSS-scaling it up/down
 * guarantees every pixel-perfect Figma spec renders identically on both
 * target resolutions without container queries scattered through every
 * micro-frontend.
 */
const LOGICAL_WIDTH = 1080;
const LOGICAL_HEIGHT = 1920;

export function ViewportLock({ children }: PropsWithChildren): React.JSX.Element {
  const [scale, setScale] = useState(1);

  useEffect(() => {
    const recompute = (): void => {
      const scaleX = window.innerWidth / LOGICAL_WIDTH;
      const scaleY = window.innerHeight / LOGICAL_HEIGHT;
      setScale(Math.min(scaleX, scaleY));
    };
    recompute();

    const resizeObserver = new ResizeObserver(recompute);
    resizeObserver.observe(document.documentElement);
    window.addEventListener("orientationchange", recompute);

    return () => {
      resizeObserver.disconnect();
      window.removeEventListener("orientationchange", recompute);
    };
  }, []);

  return (
    <div
      className="fixed inset-0 flex items-center justify-center overflow-hidden bg-black"
      aria-hidden={false}
    >
      <div
        style={{
          width: LOGICAL_WIDTH,
          height: LOGICAL_HEIGHT,
          transform: `scale(${scale})`,
          transformOrigin: "center center",
        }}
        className="relative flex flex-col overflow-hidden bg-background"
        data-testid="viewport-lock-canvas"
      >
        {children}
        <div className="grain-overlay" />
      </div>
    </div>
  );
}
