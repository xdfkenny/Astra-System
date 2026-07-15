import { type CSSProperties, type PropsWithChildren } from "react";
import { useResponsive } from "../providers/ResponsiveProvider";

const LOGICAL_WIDTH = 1080;
const LOGICAL_HEIGHT = 1920;

/**
 * Responsive viewport that fills the actual screen in portrait mode.
 *
 * Rather than emulating a fixed 1080×1920 canvas, this component uses the
 * real viewport dimensions to compute a width-based scale factor. The design
 * canvas is CSS-transformed so every component authored at the Figma spec
 * (1080 px width) maps pixel-perfect to any screen size.
 *
 * On the target kiosk hardware (both 1080×1920 and 1440×2560, 9:16 aspect)
 * the canvas fills the full viewport without letterboxing or clipping.
 *
 * A CSS custom property `--kiosk-scale` is set on the inner container so
 * child components can reference it for responsive calculations if needed.
 */
export function ViewportLock({ children }: PropsWithChildren): React.JSX.Element {
  const { scale } = useResponsive();
  const canvasStyle: CSSProperties & { "--kiosk-scale": number } = {
    width: LOGICAL_WIDTH,
    height: LOGICAL_HEIGHT,
    transform: `scale(${scale})`,
    transformOrigin: "top center",
    "--kiosk-scale": scale,
  };

  return (
    <div className="bg-background fixed inset-0 flex items-start justify-center overflow-hidden">
      <div
        style={canvasStyle}
        className="bg-background relative flex flex-col overflow-hidden"
        data-testid="viewport-lock-canvas"
      >
        {children}
        <div className="grain-overlay" />
      </div>
    </div>
  );
}

export { LOGICAL_WIDTH, LOGICAL_HEIGHT };
