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

  // The kiosk boots into the language selector; pick English to advance.
  await expect(page.getByText("Select your language")).toBeVisible();
  await page.getByRole("button", { name: "English" }).click();

  // Attract screen heading
  await expect(page.getByRole("heading", { name: "Astra" })).toBeVisible();

  // Start shopping (the attract screen is a single tappable region)
  await page.getByLabel("Attract screen. Touch to begin shopping.").click();

  // Menu catalog becomes available (or the empty-state copy)
  await expect(
    page.getByLabel("Menu categories").or(page.getByText("No items available")),
  ).toBeVisible();
});
