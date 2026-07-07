import { test, expect } from "@playwright/test";

/**
 * End-to-end smoke test for the core customer flow:
 * Attract → Menu → Cart → Payment → Receipt.
 *
 * This spec assumes the federated menu/cart/payment dev servers are reachable
 * at their standard ports (5171/5172/5173). When running from the workspace
 * root with `pnpm dev`, all remotes start in parallel.
 */
test("customer completes a full order flow", async ({ page }) => {
  await page.goto("/");

  // Attract screen
  await expect(page.getByRole("heading", { name: "Astra-Service" })).toBeVisible();
  await page.getByLabel("Tap to start shopping").click();

  // Menu screen loads the federated catalog
  await expect(page.getByLabel("Menu categories").or(page.getByText("No items available"))).toBeVisible();

  // Add the first available item to the cart (federated menu tile)
  const firstItem = page.locator("[aria-label^='Add']").first();
  await firstItem.click();

  // Review cart
  await page.getByLabel(/Review cart/i).click();
  await expect(page.getByText("Your Order")).toBeVisible();
  await page.getByRole("button", { name: /Checkout/i }).click();

  // Payment auth (federated payment remote)
  await expect(page.getByText("Choose payment method")).toBeVisible();
  await page.getByRole("button", { name: /Credit \/ Debit/i }).click();

  // Processing → Receipt
  await expect(page.getByText("Finalizing your order")).toBeVisible();
  await expect(page.getByText("Thank you!")).toBeVisible({ timeout: 10_000 });
});
