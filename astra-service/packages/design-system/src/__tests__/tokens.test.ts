import { describe, it, expect } from "vitest";
import { cssVariables as colorVariables, semantic } from "../tokens/colors";
import { cssVariables as elevationVariables } from "../tokens/elevation";
import { cssVariables as motionVariables } from "../tokens/motion";
import { cssVariables as spacingVariables, spacing } from "../tokens/spacing";
import { cssVariables as typographyVariables } from "../tokens/typography";
import { cssVariables as zIndexVariables } from "../tokens/z-index";

describe("tokens", () => {
  it("exposes color tokens as CSS variables", () => {
    expect(colorVariables["--astra-color-primary"]).toBe("#5A7A5C");
    expect(semantic.cta).toBe("#B87E6B");
    expect(semantic.error).toBe("#C4A4A4");
  });

  it("exposes spacing tokens including the 56px touch target", () => {
    expect(spacing[7]).toBe("56px");
    expect(spacingVariables["--astra-space-7"]).toBe("56px");
  });

  it("aggregates token variables across all token categories", () => {
    expect(Object.keys(typographyVariables).length).toBeGreaterThan(0);
    expect(Object.keys(elevationVariables).length).toBeGreaterThan(0);
    expect(Object.keys(motionVariables).length).toBeGreaterThan(0);
    expect(Object.keys(zIndexVariables).length).toBeGreaterThan(0);
  });
});
