import { test, expect } from "playwright/test"
import type { Page } from "playwright/test"

// Exercises /catalog/search against the seeded e2e fixture (see
// backend/internal/adapters/db/postgres/testdata/e2e_seed.sql): name search,
// Expansion Set + card number, and category/tag/rarity filters, plus the
// series-less ("ungrouped") Expansion Set being reachable from the
// Expansion Sets filter. Facet filters are checkboxes that apply immediately.
//
// These six tests (and catalog-browse.spec.ts's two) repeat the same
// page.goto(...) + page.getByRole("listitem") results shape. Left inline
// rather than extracted into a CatalogSearchPage-style helper/page object,
// since two spec files isn't enough duplication to justify one yet — worth
// revisiting once a third e2e spec file is added to this suite.

// The checkbox is controlled by the URL, so wait for it to reflect the click
// before the next filter changes the page under it.
async function check(page: Page, name: string) {
  const box = page.getByRole("checkbox", { name, exact: true })
  await box.click()
  await expect(box).toBeChecked()
}

test.describe("Catalog search", () => {
  test("searches by name", async ({ page }) => {
    await page.goto("/catalog/search")

    await page.getByLabel("Card name").fill("Sparky")
    await page.getByRole("button", { name: "Search" }).click()

    const results = page.getByRole("listitem")
    await expect(results).toHaveCount(1)
    await expect(results).toContainText("E2E Sparky")
  })

  test("searches by Expansion Set and card number", async ({ page }) => {
    await page.goto("/catalog/search")

    await check(page, "Test Set Alpha (TSA)")

    await page.getByLabel("Card number").fill("002")
    await page.getByRole("button", { name: "Search" }).click()

    const results = page.getByRole("listitem")
    await expect(results).toHaveCount(1)
    await expect(results).toContainText("E2E Boulder")
  })

  test("filters by category", async ({ page }) => {
    await page.goto("/catalog/search")

    await check(page, "Test Set Alpha (TSA)")
    await check(page, "Support")

    const results = page.getByRole("listitem")
    await expect(results).toHaveCount(1)
    await expect(results).toContainText("E2E Trainer Card")
  })

  test("filters by tag", async ({ page }) => {
    await page.goto("/catalog/search")

    await check(page, "Test Set Alpha (TSA)")
    await check(page, "Evolved")

    const results = page.getByRole("listitem")
    await expect(results).toHaveCount(1)
    await expect(results).toContainText("E2E Boulder")
  })

  test("filters by rarity", async ({ page }) => {
    await page.goto("/catalog/search")

    await check(page, "Test Set Alpha (TSA)")
    await check(page, "Rare")

    const results = page.getByRole("listitem")
    await expect(results).toHaveCount(1)
    await expect(results).toContainText("E2E Boulder")
  })

  test("offers the Ungrouped Expansion Set in the filter and searches within it", async ({
    page,
  }) => {
    await page.goto("/catalog/search")

    await expect(page.getByText("Ungrouped", { exact: true })).toBeVisible()
    await check(page, "Ungrouped Test Set (TSU)")

    const results = page.getByRole("listitem")
    await expect(results).toHaveCount(1)
    await expect(results).toContainText("E2E Orphan Card")
  })
})
