import { render, screen } from "@testing-library/react";
import { useState } from "react";
import { describe, expect, it } from "vitest";
import { OrientationLock } from "./OrientationLock";
import { ViewportLock } from "./ViewportLock";
import { ResponsiveProvider, useResponsive } from "../providers/ResponsiveProvider";

function setViewport(width: number, height: number): void {
  Object.defineProperty(window, "innerWidth", {
    configurable: true,
    value: width,
  });
  Object.defineProperty(window, "innerHeight", {
    configurable: true,
    value: height,
  });
  window.dispatchEvent(new Event("resize"));
}

function ResponsiveProbe(): React.JSX.Element {
  const { orientation, dimensions, scale, isPortrait, isLandscape } = useResponsive();
  return (
    <div>
      <span data-testid="orientation">{orientation}</span>
      <span data-testid="dimensions">{`${dimensions.width}x${dimensions.height}`}</span>
      <span data-testid="scale">{scale.toFixed(3)}</span>
      <span data-testid="portrait">{String(isPortrait)}</span>
      <span data-testid="landscape">{String(isLandscape)}</span>
    </div>
  );
}

function StatefulChild(): React.JSX.Element {
  const [label] = useState("state-preserved");
  return <div>{label}</div>;
}

describe("responsive viewport architecture", () => {
  it("treats square screens as portrait and exposes scaled dimensions", () => {
    setViewport(1080, 1080);

    render(
      <ResponsiveProvider>
        <ResponsiveProbe />
      </ResponsiveProvider>,
    );

    expect(screen.getByTestId("orientation").textContent).toBe("portrait");
    expect(screen.getByTestId("dimensions").textContent).toBe("1080x1080");
    expect(screen.getByTestId("scale").textContent).toBe("1.000");
    expect(screen.getByTestId("portrait").textContent).toBe("true");
    expect(screen.getByTestId("landscape").textContent).toBe("false");
  });

  it("shows the landscape warning without unmounting the child tree", () => {
    setViewport(1600, 900);

    render(
      <ResponsiveProvider>
        <OrientationLock>
          <StatefulChild />
        </OrientationLock>
      </ResponsiveProvider>,
    );

    expect(screen.getByText("state-preserved")).toBeDefined();
    expect(screen.getByText("Vertical orientation required")).toBeDefined();
    expect(screen.getByText(/designed for portrait/i)).toBeDefined();
  });

  it("scales the viewport canvas from the real portrait width", () => {
    setViewport(1440, 2560);

    render(
      <ResponsiveProvider>
        <ViewportLock>
          <div>canvas child</div>
        </ViewportLock>
      </ResponsiveProvider>,
    );

    const canvas = screen.getByTestId("viewport-lock-canvas");
    expect(canvas.getAttribute("style")).toContain("width: 1080px");
    expect(canvas.getAttribute("style")).toContain("height: 1920px");
    expect(canvas.getAttribute("style")).toContain("transform: scale(1.3333333333333333)");
    expect(canvas.getAttribute("style")).toContain("--kiosk-scale: 1.3333333333333333");
  });
});
