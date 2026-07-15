import { type PropsWithChildren, createContext, useContext, useEffect, useRef, useState } from "react";

/**
 * Enforces the 9:16 vertical kiosk viewport regardless of the underlying
 * panel's native resolution (1080x1920 target hardware). Recomputes a fixed
 * logical canvas and CSS-scales it up/down so the Figma spec renders
 * identically on both target resolutions.
 *
 * Exposes a correction function via context so screens (e.g. AttractScreen
 * clip-path reveal) can translate CSS-viewport coordinates to logical-space.
 */
const LOGICAL_WIDTH = 1080;
const LOGICAL_HEIGHT = 1920;

interface ViewportState {
  scale: number;
  /** Translate a CSS-viewport clientX/clientY into the logical 1080×1920 space. */
  logicalPoint: (clientX: number, clientY: number) => { x: number; y: number };
}

const ViewportCtx = createContext<ViewportState>({
  scale: 1,
  logicalPoint: (x, y) => ({ x, y }),
});

// eslint-disable-next-line react-refresh/only-export-components
export function useViewport(): ViewportState {
  return useContext(ViewportCtx);
}

export function ViewportLock({ children }: PropsWithChildren): React.JSX.Element {
  const [scale, setScale] = useState(1);
  const canvasRef = useRef<HTMLDivElement>(null);
  const scaleRef = useRef(scale);
  scaleRef.current = scale;

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

  const viewport: ViewportState = {
    scale,
    logicalPoint: (clientX: number, clientY: number) => {
      const s = scaleRef.current;
      if (s === 0) return { x: clientX, y: clientY };
      const rect = canvasRef.current?.getBoundingClientRect();
      if (!rect) return { x: clientX / s, y: clientY / s };
      const offsetX = clientX - rect.left;
      const offsetY = clientY - rect.top;
      return { x: offsetX / s, y: offsetY / s };
    },
  };

  return (
    <div className="fixed inset-0 flex items-center justify-center overflow-hidden bg-black" aria-hidden={false}>
      <div
        ref={canvasRef}
        style={{
          width: LOGICAL_WIDTH,
          height: LOGICAL_HEIGHT,
          transform: `scale(${scale})`,
          transformOrigin: "center center",
        }}
        className="relative flex flex-col overflow-hidden bg-background"
        data-testid="viewport-lock-canvas"
      >
        <ViewportCtx.Provider value={viewport}>
          {children}
        </ViewportCtx.Provider>
        <div className="grain-overlay" />
      </div>
    </div>
  );
}

