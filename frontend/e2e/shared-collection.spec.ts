import AxeBuilder from "@axe-core/playwright"
import { test, expect } from "playwright/test"
import type { Page, Route } from "playwright/test"

// A Guest opens a Shared Collection's link (ADR-0025): no sign-in, read-only,
// the owner's name shown, and `noindex`. The API answers from stubs, so the
// spec needs no seeded backend. A direct load runs the route loader on the dev
// server, where browser-level stubs don't apply, so the page is reached by
// client navigation from `/`, as the catalog guest specs do.
const PROFILE = "profile-ash"
const COLLECTION = "holiday-binder-3f9a2"
const API = `**/shared/${PROFILE}/${COLLECTION}`

const expansionSet = {
  id: "set-1",
  code: "TST",
  name: "Test Set",
  imageUrl: "",
}
const pikachu = {
  id: "card-pikachu",
  name: "Pikachu",
  localId: "025",
  category: "Pokémon",
  tags: [],
  illustrator: "someone",
  imageUrl: "",
  expansionSet,
  rarity: { id: "rarity-1", code: "C", name: "Common" },
}

// The JSON bodies the API answers with: an envelope, or an error detail.
interface ApiBody {
  data?: unknown
  meta?: { total: number; page: number; limit: number }
  detail?: string
}

function reply(route: Route, status: number, body: ApiBody) {
  return route.fulfill({
    status,
    contentType: "application/json",
    headers: {
      // Echo the caller's origin instead of hard-coding the port.
      "access-control-allow-origin": route.request().headers()["origin"] ?? "",
    },
    body: JSON.stringify(body),
  })
}

async function stubSharedCollection(page: Page) {
  await page.route(API, (route) =>
    reply(route, 200, {
      data: {
        ownerName: "Ash Ketchum",
        title: "Holiday Binder",
        description: "Cards for the trade table",
      },
    })
  )
  await page.route(`${API}/entries*`, (route) =>
    reply(route, 200, {
      data: [{ card: pikachu, quantity: 2 }],
      meta: { total: 1, page: 1, limit: 100 },
    })
  )
  await page.route(`${API}/facets*`, (route) =>
    reply(route, 200, {
      data: { categories: [], expansionSets: [], rarities: [], tags: [] },
    })
  )
  await page.route("**/catalog/series", (route) =>
    reply(route, 200, { data: { series: [], ungroupedExpansionSets: [] } })
  )
}

test.describe("Shared Collection as a Guest", () => {
  test("opens the link read-only, names the owner, and asks search engines to skip it", async ({
    page,
  }) => {
    await stubSharedCollection(page)
    await page.goto("/")
    await page.evaluate((path) => {
      window.history.pushState(null, "", path)
      window.dispatchEvent(new PopStateEvent("popstate", { state: null }))
    }, `/s/${PROFILE}/${COLLECTION}`)

    await expect(
      page.getByRole("heading", { name: "Holiday Binder", level: 1 })
    ).toBeVisible()
    await expect(page.getByText("Shared by Ash Ketchum")).toBeVisible()
    await expect(page.getByText("Cards for the trade table")).toBeVisible()
    await expect(page.getByRole("link", { name: "Pikachu" })).toBeVisible()
    // The quantity is read-only: no stepper, input or button in the card list.
    const list = page.getByRole("list")
    await expect(list.getByText("Quantity: 2")).toBeVisible()
    await expect(list.getByRole("button")).toHaveCount(0)
    await expect(list.getByRole("spinbutton")).toHaveCount(0)
    await expect(list.getByRole("textbox")).toHaveCount(0)
    await expect(page.locator('meta[name="robots"]')).toHaveAttribute(
      "content",
      "noindex"
    )

    const results = await new AxeBuilder({ page }).analyze()
    expect(results.violations).toEqual([])
  })
})
