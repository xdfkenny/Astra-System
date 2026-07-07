import { describe, it, expect } from "vitest";
import { cssVariables as colorVariables, semantic, slate } from "../tokens/colors";
import { cssVariables as elevationVariables } from "../tokens/elevation";
import { cssVariables as motionVariables } from "../tokens/motion";
import { cssVariables as spacingVariables, spacing } from "../tokens/spacing";
import { cssVariables as typographyVariables } from "../tokens/typography";
import { cssVariables as zIndexVariables } from "../tokens/z-index";

describe("tokens", () => {
  it("exposes color tokens as CSS variables", () => {
    expect(slate[50]).toBe("#f8fafc");
    expect(slate[950]).toBe("#020617");
    expect(colorVariables["--astra-color-slate-50"]).toBe("#f8fafc");
    expect(colorVariables["--astra-color-slate-950"]).toBe("#020617");
    expect(colorVariables["--astra-color-primary"]).toBe("#0d9488");
    expect(semantic.cta).toBe("#f59e0b");
    expect(semantic.error).toBe("#f43f5e");
  });

  it("exposes spacing tokens including the 56px touch target", () => {
    expect(spacing[14]).toBe("56px");
    expect(spacingVariables["--astra-space-14"]).toBe("56px");
  });

  it("aggregates token variables across all token categories", () => {
    expect(Object.keys(typographyVariables).length).toBeGreaterThan(0);
    expect(Object.keys(elevationVariables).length).toBeGreaterThan(0);
    expect(Object.keys(motionVariables).length).toBeGreaterThan(0);
    expect(Object.keys(zIndexVariables).length).toBeGreaterThan(0);
  });
});
