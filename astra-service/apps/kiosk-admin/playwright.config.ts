import { defineConfig, devices } from "@playwright/test";

/**
 * E2E suite for the admin dashboard. Runs against a desktop viewport and
 * exercises role-based navigation plus critical operational flows.
 */
export default defineConfig({
  testDir: "./e2e",
  fullyParallel: true,
  forbidOnly: Boolean(process.env["CI"]),
  retries: process.env["CI"] ? 2 : 0,
  ...(process.env["CI"] ? { workers: 1 } : {}),
  reporter: "list",
  use: {
    baseURL: "http://localhost:5174",
    trace: "on-first-retry",
    viewport: { width: 1280, height: 900 },
  },
  projects: [
    {
      name: "chromium-admin",
      use: { ...devices["Desktop Chrome"], viewport: { width: 1280, height: 900 } },
    },
  ],
  webServer: {
    command: "pnpm dev",
    url: "http://localhost:5174",
    reuseExistingServer: !process.env["CI"],
    timeout: 120_000,
  },
});
