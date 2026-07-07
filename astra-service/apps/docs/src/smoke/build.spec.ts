import { execSync } from "node:child_process";
import * as fs from "node:fs";
import * as path from "node:path";
import { describe, expect, it } from "vitest";

describe("docs smoke test", () => {
  it("builds the docs site and emits index.html", () => {
    execSync("pnpm exec vite build", {
      cwd: process.cwd(),
      stdio: "pipe",
    });

    const indexHtml = path.resolve(process.cwd(), "dist", "index.html");
    expect(fs.existsSync(indexHtml)).toBe(true);
  });
});
