import { test, expect } from "@playwright/test";

test.describe("admin dashboard flows", () => {
  test("dashboard loads and shows navigation", async ({ page }) => {
    await page.goto("/");
    await expect(page.getByText("Astra Admin")).toBeVisible();
    await expect(page.getByRole("heading", { name: "Dashboard" })).toBeVisible();
    await expect(page.getByText("Mesh Topology")).toBeVisible();
  });

  test("navigation switches between pages", async ({ page }) => {
    await page.goto("/");
    await page.getByRole("link", { name: "Kiosks" }).click();
    await expect(page.getByRole("heading", { name: "Kiosks" })).toBeVisible();

    await page.getByRole("link", { name: "Orders" }).click();
    await expect(page.getByRole("heading", { name: "Orders" })).toBeVisible();
  });

  test("theme toggle updates the document class", async ({ page }) => {
    await page.goto("/");
    await page.getByRole("button", { name: "Dark" }).click();
    await expect(page.locator("html")).toHaveClass(/dark/);
  });
});
