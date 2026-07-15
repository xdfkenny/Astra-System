import { describe, expect, it } from "vitest";
import { readFileSync } from "node:fs";
import { fileURLToPath } from "node:url";
import { color, dark } from "./tokens";

/**
 * Drift guard: tokens.css is hand-maintained in parallel with tokens.ts for
 * Tailwind v4's zero-JS `@theme` pipeline. This test fails CI the moment a
 * designer updates one file and forgets the other.
 *
 * The CSS semantic layer references base tokens via `var(--color-*)`, so the
 * guard resolves the var() chain within the relevant theme block before
 * comparing against the flattened hex values exported from tokens.ts.
 */
const cssPath = fileURLToPath(new URL("./tokens.css", import.meta.url));
const css = readFileSync(cssPath, "utf-8");

function extractBlock(source: string, selectorIndex: number): string {
  const open = source.indexOf("{", selectorIndex);
  if (open === -1) throw new Error("No block found for selector");
  let depth = 0;
  for (let i = open; i < source.length; i++) {
    if (source[i] === "{") depth++;
    else if (source[i] === "}") {
      depth--;
      if (depth === 0) return source.slice(open + 1, i);
    }
  }
  throw new Error("Unbalanced braces in CSS block");
}

function parseDeclarations(block: string): Map<string, string> {
  const map = new Map<string, string>();
  const re = /(--[\w-]+)\s*:\s*([^;]+);/g;
  let m: RegExpExecArray | null;
  while ((m = re.exec(block)) !== null) {
    const [, name, value] = m;
    if (name && value) map.set(name, value.trim());
  }
  return map;
}

function resolve(name: string, map: Map<string, string>, seen = new Set<string>()): string | undefined {
  if (seen.has(name)) return undefined;
  seen.add(name);
  const value = map.get(name);
  if (value === undefined) return undefined;
  const varMatch = /^var\((--[\w-]+)\)$/.exec(value);
  const referenced = varMatch?.[1];
  if (referenced) return resolve(referenced, map, seen);
  return value;
}

function cssVarName(key: string): string {
  return `--color-${key.replace(/([A-Z])/g, "-$1").toLowerCase()}`;
}

const rootMap = parseDeclarations(extractBlock(css, css.indexOf(":root {")));
const darkMap = new Map(rootMap);
for (const [k, v] of parseDeclarations(extractBlock(css, css.indexOf("html.dark")))) {
  darkMap.set(k, v);
}

describe("design token parity", () => {
  it("every TS color token resolves to a matching CSS custom property (light)", () => {
    for (const [key, value] of Object.entries(color)) {
      if (!value.startsWith("#")) continue;
      const name = cssVarName(key);
      const resolved = resolve(name, rootMap);
      expect(resolved?.toLowerCase(), `Missing or mismatched ${name} for ${key}`).toBe(value.toLowerCase());
    }
  });

  it("every TS dark token resolves to a matching CSS custom property (dark)", () => {
    for (const [key, value] of Object.entries(dark)) {
      if (!value.startsWith("#")) continue;
      const name = cssVarName(key);
      const resolved = resolve(name, darkMap);
      expect(resolved?.toLowerCase(), `Missing or mismatched dark ${name} for ${key}`).toBe(value.toLowerCase());
    }
  });
});
