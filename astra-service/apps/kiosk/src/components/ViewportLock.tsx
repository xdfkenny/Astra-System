import { type PropsWithChildren, useEffect, useState } from "react";

/**
 * Enforces the 9:16 vertical kiosk viewport regardless of the underlying
 * panel's native resolution (1080x1920 target hardware). Recomputes a fixed
 * logical canvas and CSS-scales it up/down so the Figma spec renders
 * identically on both target resolutions.
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
    <div className="fixed inset-0 flex items-center justify-center overflow-hidden bg-black" aria-hidden={false}>
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
