import { test, expect } from "playwright/test"
import type { Page } from "playwright/test"

// Exercises /catalog/search against the seeded e2e fixture (see
// backend/internal/adapters/db/postgres/testdata/e2e_seed.sql): name search,
// Expansion Set + card number, and category/tag/rarity filters, plus the
// series-less ("ungrouped") Expansion Set being reachable from the
// Expansion Set filter. Facet filters are dropdown multi-selects of checkboxes
// that apply immediately; selections show as removable chips (buttons, not
// listitems, so they never count as results).
//
// These six tests (and catalog-browse.spec.ts's two) repeat the same
// page.goto(...) + page.getByRole("listitem") results shape. Left inline
// rather than extracted into a CatalogSearchPage-style helper/page object,
// since two spec files isn't enough duplication to justify one yet — worth
// revisiting once a third e2e spec file is added to this suite.

// Opens the dropdown (they stay open while toggling; Escape closes it so it
// doesn't cover the next trigger). The checkbox is controlled by the URL, so
// wait for it to reflect the click before the next filter changes the page.
async function check(page: Page, filter: string, name: string) {
  await page.getByRole("button", { name: new RegExp(`^${filter}`) }).click()
  const box = page.getByRole("checkbox", { name, exact: true })
  await box.click()
  await expect(box).toBeChecked()
  await page.keyboard.press("Escape")
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

    await check(page, "Expansion Set", "Test Set Alpha (TSA)")

    await page.getByLabel("Card number").fill("002")
    await page.getByRole("button", { name: "Search" }).click()

    const results = page.getByRole("listitem")
    await expect(results).toHaveCount(1)
    await expect(results).toContainText("E2E Boulder")
  })

  test("filters by category", async ({ page }) => {
    await page.goto("/catalog/search")

    await check(page, "Expansion Set", "Test Set Alpha (TSA)")
    await check(page, "Category", "Support")

    const results = page.getByRole("listitem")
    await expect(results).toHaveCount(1)
    await expect(results).toContainText("E2E Trainer Card")
  })

  test("filters by tag", async ({ page }) => {
    await page.goto("/catalog/search")

    await check(page, "Expansion Set", "Test Set Alpha (TSA)")
    await check(page, "Tag", "Evolved")

    const results = page.getByRole("listitem")
    await expect(results).toHaveCount(1)
    await expect(results).toContainText("E2E Boulder")
  })

  test("filters by rarity", async ({ page }) => {
    await page.goto("/catalog/search")

    await check(page, "Expansion Set", "Test Set Alpha (TSA)")
    await check(page, "Rarity", "Rare")

    const results = page.getByRole("listitem")
    await expect(results).toHaveCount(1)
    await expect(results).toContainText("E2E Boulder")
  })

  test("offers the Ungrouped Expansion Set in the filter and searches within it", async ({
    page,
  }) => {
    await page.goto("/catalog/search")

    await page.getByRole("button", { name: /^Expansion Set/ }).click()
    await expect(page.getByRole("group", { name: "Ungrouped" })).toBeVisible()
    await page.keyboard.press("Escape")
    await check(page, "Expansion Set", "Ungrouped Test Set (TSU)")

    const results = page.getByRole("listitem")
    await expect(results).toHaveCount(1)
    await expect(results).toContainText("E2E Orphan Card")
  })

  test("narrows Expansion Sets by search and removes a selection via its chip", async ({
    page,
  }) => {
    await page.goto("/catalog/search")

    await page.getByRole("button", { name: /^Expansion Set/ }).click()
    await page.getByLabel("Search Expansion Set").fill("Ungrouped")
    await expect(page.getByRole("checkbox", { name: /Test Set Alpha/ })).toHaveCount(0)
    await page.getByRole("checkbox", { name: "Ungrouped Test Set (TSU)" }).click()
    await page.keyboard.press("Escape")

    await expect(page.getByRole("listitem").filter({ hasText: "E2E Orphan Card" })).toHaveCount(1)
    await page.getByRole("button", { name: "Remove Ungrouped Test Set (TSU)" }).click()
    await expect(page.getByRole("button", { name: /^Remove / })).toHaveCount(0)
  })
})
