import { type CSSProperties, type PropsWithChildren } from "react";
import { useResponsive } from "../providers/ResponsiveProvider";

const LOGICAL_WIDTH = 1080;
const LOGICAL_HEIGHT = 1920;

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
