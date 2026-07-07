import { describe, expect, it } from "vitest";
import { readFileSync } from "node:fs";
import { fileURLToPath } from "node:url";
import { color, dark } from "./tokens";

/**
 * Drift guard: tokens.css is hand-maintained in parallel with tokens.ts for
 * Tailwind v4's zero-JS `@theme` pipeline. This test fails CI the moment a
 * designer updates one file and forgets the other.
 */
describe("design token parity", () => {
  const cssPath = fileURLToPath(new URL("./tokens.css", import.meta.url));
  const css = readFileSync(cssPath, "utf-8");

  it("every TS color token has a matching CSS custom property with the same hex value", () => {
    for (const [key, value] of Object.entries(color)) {
      if (!value.startsWith("#")) continue; // skip rgba() computed tokens
      const cssVarName = `--color-${key.replace(/([A-Z])/g, "-$1").toLowerCase()}`;
      const pattern = new RegExp(`${cssVarName}:\\s*${value}`, "i");
      expect(css, `Missing or mismatched ${cssVarName} for ${key}`).toMatch(pattern);
    }
  });

  it("every TS dark token has a matching CSS custom property with the same hex value", () => {
    for (const [key, value] of Object.entries(dark)) {
      if (!value.startsWith("#")) continue;
      const cssVarName = `--color-${key.replace(/([A-Z])/g, "-$1").toLowerCase()}`;
      const pattern = new RegExp(`${cssVarName}:\\s*${value}`, "i");
      expect(css, `Missing or mismatched dark ${cssVarName} for ${key}`).toMatch(pattern);
    }
  });
});
