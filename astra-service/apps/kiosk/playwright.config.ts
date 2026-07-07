import { defineConfig, devices } from "@playwright/test";

/**
 * E2E suite for the unified kiosk. Tests run against a 9:16 viewport at the
 * target hardware resolution (1080x1920) to exercise the touch-target and
 * viewport-lock behavior end-to-end.
 */
export default defineConfig({
  testDir: "./e2e",
  fullyParallel: true,
  forbidOnly: Boolean(process.env["CI"]),
  retries: process.env["CI"] ? 2 : 0,
  workers: process.env["CI"] ? 1 : undefined,
  reporter: "list",
  use: {
    baseURL: "http://localhost:5180",
    trace: "on-first-retry",
    viewport: { width: 1080, height: 1920 },
    hasTouch: true,
    isMobile: true,
  },
  projects: [
    {
      name: "chromium-kiosk",
      use: { ...devices["Desktop Chrome"], viewport: { width: 1080, height: 1920 }, hasTouch: true },
    },
  ],
  webServer: {
    command: "pnpm dev",
    url: "http://localhost:5180",
    reuseExistingServer: !process.env["CI"],
    timeout: 120_000,
  },
});
