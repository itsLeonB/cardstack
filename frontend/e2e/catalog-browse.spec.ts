import { test, expect } from "playwright/test"

// Exercises /catalog against the seeded e2e fixture (see
// backend/internal/adapters/db/postgres/testdata/e2e_seed.sql): a grouped
// Series/Expansion Set and a series-less ("ungrouped") Expansion Set.

test.describe("Catalog browse", () => {
  test("lists a Series with its Expansion Set and drills into its cards", async ({
    page,
  }) => {
    await page.goto("/catalog")

    await expect(
      page.getByRole("heading", { name: "Test Series Alpha", level: 2 })
    ).toBeVisible()

    const setLink = page.getByRole("link", { name: /Test Set Alpha/ })
    await expect(setLink).toContainText("TSA")
    await expect(setLink).toContainText(/Released January 1, 2020/)

    await setLink.click()

    await expect(page).toHaveURL(/\/catalog\/sets\//)
    await expect(
      page.getByRole("heading", { name: "Test Set Alpha", level: 1 })
    ).toBeVisible()
    await expect(page.getByText("3 cards", { exact: true })).toBeVisible()

    const cards = page.getByRole("listitem")
    await expect(cards).toHaveCount(3)
    await expect(cards.filter({ hasText: "E2E Sparky" })).toHaveCount(1)
    await expect(cards.filter({ hasText: "E2E Boulder" })).toHaveCount(1)
    await expect(cards.filter({ hasText: "E2E Trainer Card" })).toHaveCount(1)
  })

  test("lists an Ungrouped Expansion Set and drills into its card", async ({
    page,
  }) => {
    await page.goto("/catalog")

    await expect(
      page.getByRole("heading", { name: "Ungrouped Expansion Sets", level: 2 })
    ).toBeVisible()

    const setLink = page.getByRole("link", { name: /Ungrouped Test Set/ })
    await expect(setLink).toContainText("TSU")

    await setLink.click()

    await expect(page).toHaveURL(/\/catalog\/sets\//)
    await expect(
      page.getByRole("heading", { name: "Ungrouped Test Set", level: 1 })
    ).toBeVisible()
    await expect(page.getByText("1 card", { exact: true })).toBeVisible()

    const cards = page.getByRole("listitem")
    await expect(cards).toHaveCount(1)
    await expect(cards).toContainText("E2E Orphan Card")
  })
})
