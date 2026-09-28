import { test, expect } from "playwright/test"

// Exercises /catalog/search against the seeded e2e fixture (see
// backend/internal/adapters/db/postgres/testdata/e2e_seed.sql): name search,
// Expansion Set + card number, and category/tag/rarity filters, plus the
// series-less ("ungrouped") Expansion Set being reachable from the
// Expansion Set dropdown.
//
// These six tests (and catalog-browse.spec.ts's two) repeat the same
// page.goto(...) + page.getByRole("listitem") results shape. Left inline
// rather than extracted into a CatalogSearchPage-style helper/page object,
// since two spec files isn't enough duplication to justify one yet — worth
// revisiting once a third e2e spec file is added to this suite.

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

    await page.getByLabel("Expansion Set").click()
    await page.getByRole("option", { name: "Test Set Alpha (TSA)" }).click()

    await page.getByLabel("Card number").fill("002")
    await page.getByRole("button", { name: "Search" }).click()

    const results = page.getByRole("listitem")
    await expect(results).toHaveCount(1)
    await expect(results).toContainText("E2E Boulder")
  })

  test("filters by category", async ({ page }) => {
    await page.goto("/catalog/search")

    await page.getByLabel("Expansion Set").click()
    await page.getByRole("option", { name: "Test Set Alpha (TSA)" }).click()

    await page.getByLabel("Category").click()
    await page.getByRole("option", { name: "Support", exact: true }).click()

    const results = page.getByRole("listitem")
    await expect(results).toHaveCount(1)
    await expect(results).toContainText("E2E Trainer Card")
  })

  test("filters by tag", async ({ page }) => {
    await page.goto("/catalog/search")

    await page.getByLabel("Expansion Set").click()
    await page.getByRole("option", { name: "Test Set Alpha (TSA)" }).click()

    await page.getByLabel("Tag").click()
    await page.getByRole("option", { name: "Evolved", exact: true }).click()

    const results = page.getByRole("listitem")
    await expect(results).toHaveCount(1)
    await expect(results).toContainText("E2E Boulder")
  })

  test("filters by rarity", async ({ page }) => {
    await page.goto("/catalog/search")

    await page.getByLabel("Expansion Set").click()
    await page.getByRole("option", { name: "Test Set Alpha (TSA)" }).click()

    await page.getByLabel("Rarity").click()
    await page.getByRole("option", { name: "Rare", exact: true }).click()

    const results = page.getByRole("listitem")
    await expect(results).toHaveCount(1)
    await expect(results).toContainText("E2E Boulder")
  })

  test("offers the Ungrouped Expansion Set in the dropdown and searches within it", async ({
    page,
  }) => {
    await page.goto("/catalog/search")

    await page.getByLabel("Expansion Set").click()

    const listbox = page.getByRole("listbox")
    await expect(listbox.getByText("Ungrouped", { exact: true })).toBeVisible()
    const option = listbox.getByRole("option", { name: "Ungrouped Test Set (TSU)" })
    await expect(option).toBeVisible()
    await option.click()

    const results = page.getByRole("listitem")
    await expect(results).toHaveCount(1)
    await expect(results).toContainText("E2E Orphan Card")
  })
})
