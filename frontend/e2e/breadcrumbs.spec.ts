import { test, expect } from "playwright/test"

// Breadcrumbs on nested public catalog pages, against the seeded e2e fixture
// (see backend/internal/adapters/db/postgres/testdata/e2e_seed.sql). Top-level
// pages carry none.

test.describe("Breadcrumbs", () => {
  test("a card page shows its trail and each crumb navigates up", async ({ page }) => {
    await page.goto("/catalog")
    await page.getByRole("link", { name: /Test Set Alpha/ }).click()
    await page.getByRole("link", { name: "E2E Sparky" }).first().click()

    const trail = page.getByRole("navigation", { name: "Breadcrumb" })
    await expect(trail.getByRole("link")).toHaveText(["Catalog", "Test Set Alpha"])
    await expect(trail.locator("[aria-current=page]")).toHaveText("E2E Sparky")

    await trail.getByRole("link", { name: "Test Set Alpha" }).click()
    await expect(page).toHaveURL(/\/catalog\/sets\//)
    await expect(page.getByRole("heading", { name: "Test Set Alpha", level: 1 })).toBeVisible()
    await expect(trail.locator("[aria-current=page]")).toHaveText("Test Set Alpha")

    await trail.getByRole("link", { name: "Catalog" }).click()
    await expect(page).toHaveURL(/\/catalog$/)
    await expect(page.getByRole("heading", { name: "Catalog", level: 1 })).toBeVisible()
    await expect(page.getByRole("navigation", { name: "Breadcrumb" })).toHaveCount(0)
  })

  test("the search page shows Catalog > Search", async ({ page }) => {
    await page.goto("/catalog/search")

    const trail = page.getByRole("navigation", { name: "Breadcrumb" })
    await expect(trail.locator("[aria-current=page]")).toHaveText("Search")

    await trail.getByRole("link", { name: "Catalog" }).click()
    await expect(page).toHaveURL(/\/catalog$/)
  })
})
