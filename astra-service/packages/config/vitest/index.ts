import path from "node:path";
import { defineConfig } from "vitest/config";

export default defineConfig({
  test: {
    environment: "jsdom",
    globals: false,
    include: ["src/**/*.{test,spec}.{ts,tsx}"],
    setupFiles: [path.resolve(import.meta.dirname, "./setup.ts")],
    coverage: {
      provider: "v8",
      reporter: ["text", "lcov", "html"],
      exclude: ["node_modules/", "dist/", ".output/", "coverage/", "**/*.d.ts", "**/*.config.ts"],
    },
  },
});
