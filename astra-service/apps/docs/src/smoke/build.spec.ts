import { exec } from "node:child_process";
import { promisify } from "node:util";
import * as fs from "node:fs";
import * as path from "node:path";
import { describe, expect, it } from "vitest";

const execAsync = promisify(exec);

describe("docs smoke test", () => {
  it("builds the docs site and emits index.html", async () => {
    await execAsync("pnpm exec vite build", {
      cwd: process.cwd(),
    });

    const indexHtml = path.resolve(process.cwd(), "dist", "index.html");
    expect(fs.existsSync(indexHtml)).toBe(true);
  });
});
