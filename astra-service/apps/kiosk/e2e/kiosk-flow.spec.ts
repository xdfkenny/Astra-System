import { test, expect } from "@playwright/test";

/**
 * End-to-end smoke test for the unified kiosk shell.
 *
 * The e2e job only runs the kiosk dev server (no backend microservices), so
 * this spec exercises the attract → menu transition rather than the full
 * order flow. It verifies the attract screen renders and that tapping to
 * start reveals the menu catalog (or its empty state when no backend is up).
 */
test("kiosk attract screen loads and reveals the menu", async ({ page }) => {
  await page.goto("/");

  // Attract screen heading
  await expect(page.getByRole("heading", { name: "Astra" })).toBeVisible();

  // Start shopping
  await page.getByLabel("Touch to begin shopping").click();

  // Menu catalog becomes available (or the empty-state copy)
  await expect(
    page.getByLabel("Menu categories").or(page.getByText("No items available")),
  ).toBeVisible();
});
