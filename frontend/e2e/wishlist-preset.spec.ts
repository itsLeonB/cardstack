import { test, expect } from "playwright/test"
import type { Route } from "playwright/test"

import {
  SIGNED_IN_TAG,
  signInAsTestUser,
  useSignedInSuite,
} from "./support/clerk-auth"

/** A stubbed API answer: always a `data` envelope, plus paging for lists. */
type StubBody = {
  data: unknown
  meta?: { total: number; page: number; limit: number }
}

// The Wishlist is a signed-in page (routes/_authenticated/wishlist.tsx), so the
// suite signs in through Clerk's real development instance; the preset's own
// calls are stubbed for a deterministic count, and the Catalog facets the sheet
// offers are stubbed so the chips do not depend on the seeded data.
function json(route: Route, body: StubBody) {
  const headers = {
    "access-control-allow-origin": route.request().headers()["origin"] ?? "",
    "access-control-allow-credentials": "true",
    // The bearer token makes every call cross-origin preflighted.
    "access-control-allow-headers": "authorization, content-type",
    "access-control-allow-methods": "GET, POST, OPTIONS",
  }
  if (route.request().method() === "OPTIONS") {
    return route.fulfill({ status: 204, headers })
  }
  return route.fulfill({
    status: 200,
    contentType: "application/json",
    headers,
    body: JSON.stringify(body),
  })
}

test.describe("Wishlist Preset", { tag: SIGNED_IN_TAG }, () => {
  useSignedInSuite()

  test("opens the preset sheet from the Wishlist page and shows live preview count", async ({
    page,
  }) => {
    // The sheet's options: the whole catalog's facets, unfiltered.
    await page.route("**/catalog/facets*", (route) =>
      json(route, {
        data: {
          expansionSets: [
            {
              id: "set-1",
              code: "S1",
              name: "Set 1",
              imageUrl: "",
              available: true,
            },
            {
              id: "set-2",
              code: "S2",
              name: "Set 2",
              imageUrl: "",
              available: true,
            },
          ],
          rarities: [
            { id: "rarity-1", code: "C", name: "Common", available: true },
          ],
          categories: [{ value: "Pokémon", available: true }],
          tags: [],
        },
      })
    )

    // An empty Wishlist, so only the Preset's own Cards are in play.
    await page.route("**/wishlist?status=all&page=1&limit=24", (route) =>
      json(route, { data: [], meta: { total: 0, page: 1, limit: 24 } })
    )

    await page.route("**/wishlist/preset/preview", (route) =>
      json(route, { data: { count: 5, cards: [] } })
    )
    await page.route("**/wishlist/preset/apply", (route) =>
      json(route, { data: { count: 5, cards: [] } })
    )

    await signInAsTestUser(page)
    await page.goto("/wishlist")

    await page.getByRole("button", { name: "Fill from sets" }).click()

    const sheet = page.getByRole("heading", { name: "Fill from sets" })
    await expect(sheet).toBeVisible()

    await page.getByRole("button", { name: "Set 1" }).click()
    await page.getByRole("button", { name: "Preview" }).click()
    await expect(page.getByText("5 Cards will be added")).toBeVisible()

    await page.getByRole("button", { name: "Add to Wishlist" }).click()

    await expect(sheet).not.toBeVisible()
  })
})
