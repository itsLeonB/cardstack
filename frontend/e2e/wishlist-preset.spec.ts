import { test, expect } from "playwright/test"

test.describe("Wishlist Preset", () => {
  test("opens the preset sheet from the Wishlist page and shows live preview count", async ({ page }) => {
    // Stub the facets endpoint to return some expansion sets
    await page.route(
      "**/wishlist/facets?status=all",
      async (route) => {
        await route.fulfill({
          status: 200,
          contentType: "application/json",
          headers: {
            "access-control-allow-origin": route.request().headers()["origin"] ?? "",
          },
          body: JSON.stringify({
            data: {
              expansionSets: [
                { id: "set-1", code: "S1", name: "Set 1", imageUrl: "", available: true, seriesId: null },
                { id: "set-2", code: "S2", name: "Set 2", imageUrl: "", available: true, seriesId: null },
              ],
              rarities: [
                { id: "rarity-1", code: "C", name: "Common", available: true },
              ],
              categories: [{ value: "Pokémon", available: true }],
              tags: [],
            },
          }),
        })
      }
    )

    // Stub the wishlist list to be empty
    await page.route("**/wishlist?status=all&page=1&limit=24", async (route) => {
      await route.fulfill({
        status: 200,
        contentType: "application/json",
        headers: {
          "access-control-allow-origin": route.request().headers()["origin"] ?? "",
        },
        body: JSON.stringify({
          data: [],
          meta: { total: 0, page: 1, limit: 24 },
        }),
      })
    })

    // Stub the preset preview endpoint
    await page.route("**/wishlist/preset/preview", async (route) => {
      const body = await route.request().postDataJSON()
      await route.fulfill({
        status: 200,
        contentType: "application/json",
        headers: {
          "access-control-allow-origin": route.request().headers()["origin"] ?? "",
        },
        body: JSON.stringify({
          data: {
            count: 5,
            cards: [],
          },
        }),
      })
    })

    // Stub the preset apply endpoint
    await page.route("**/wishlist/preset/apply", async (route) => {
      await route.fulfill({
        status: 200,
        contentType: "application/json",
        headers: {
          "access-control-allow-origin": route.request().headers()["origin"] ?? "",
        },
        body: JSON.stringify({
          data: {
            count: 5,
            cards: [],
          },
        }),
      })
    })

    // Stub auth check to pass
    await page.route("**/auth/**", async (route) => {
      await route.fulfill({
        status: 200,
        contentType: "application/json",
        headers: {
          "access-control-allow-origin": route.request().headers()["origin"] ?? "",
        },
        body: JSON.stringify({
          data: { id: "user-1", email: "test@example.com" },
        }),
      })
    })

    await page.goto("/wishlist")

    // Click "Fill from sets" button
    await page.getByRole("button", { name: "Fill from sets" }).click()

    // The sheet should be open
    await expect(page.getByRole("heading", { name: "Fill from sets" })).toBeVisible()

    // Select an expansion set
    await page.getByRole("button", { name: "Set 1" }).click()

    // Click Preview
    await page.getByRole("button", { name: "Preview" }).click()

    // Should show the live count
    await expect(page.getByText("5 Cards will be added")).toBeVisible()

    // Click "Add to Wishlist"
    await page.getByRole("button", { name: "Add to Wishlist" }).click()

    // Sheet should close
    await expect(page.getByRole("heading", { name: "Fill from sets" })).not.toBeVisible()
  })
})
