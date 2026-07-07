import { defineConfig } from "vitest/config";

export default defineConfig({
  test: {
    globals: true,
    environment: "happy-dom",
    setupFiles: ["./src/test-utils/setup.ts"],
    exclude: ["e2e/**", "node_modules", "dist"],
    coverage: {
      provider: "v8",
      reporter: ["text", "html"],
    },
  },
});
