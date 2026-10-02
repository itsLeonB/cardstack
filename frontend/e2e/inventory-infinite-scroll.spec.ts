import { test, expect } from "playwright/test"
import type { Page, Route } from "playwright/test"

// Stubs a signed-in user with one Collection of 250 entries (100 per page, so
// three pages) and the same 250 cards as the Master Inventory, instead of the
// seeded fixture, which is too small to scroll.
const entry = (index: number, quantity = 3) => ({
  quantity,
  card: {
    id: `card-${index}`,
    name: `Card ${index}`,
    localId: String(index),
    category: "Pokémon",
    tags: [],
    illustrator: "someone",
    imageUrl: "",
    expansionSet: { id: "set-1", code: "TST", name: "Test Set", imageUrl: "" },
    rarity: { id: "rarity-1", code: "C", name: "Common" },
  },
})

type Entry = ReturnType<typeof entry>

// Credentialed cross-origin calls need the caller's exact origin echoed back.
function cors(route: Route) {
  return {
    "access-control-allow-origin": route.request().headers()["origin"] ?? "",
    "access-control-allow-credentials": "true",
    "access-control-allow-methods": "GET, PATCH, OPTIONS",
    "access-control-allow-headers": "content-type, x-csrf-token",
  }
}

type StubBody = {
  data: unknown
  meta?: { total: number; page: number; limit: number }
}

function json(route: Route, body: StubBody) {
  return route.fulfill({
    status: 200,
    contentType: "application/json",
    headers: cors(route),
    body: JSON.stringify(body),
  })
}

function pageOf(route: Route, rows: Entry[]) {
  const url = new URL(route.request().url())
  const pageNumber = Number(url.searchParams.get("page") ?? 1)
  const limit = Number(url.searchParams.get("limit") ?? 24)
  return {
    data: rows.slice((pageNumber - 1) * limit, pageNumber * limit),
    meta: { total: rows.length, page: pageNumber, limit },
  }
}

const FACETS = {
  data: { categories: [], expansionSets: [], rarities: [], tags: [] },
}

async function stubApi(page: Page) {
  let entries = Array.from({ length: 250 }, (_, index) => entry(index))
  const entryRequests: URL[] = []
  const bulkBodies: { items: { cardId: string; quantity: number }[] }[] = []
  // Page 3 of the Collection is answered only once the test lets it.
  let releasePageThree = () => {}
  const pageThree = new Promise<void>((resolve) => (releasePageThree = resolve))
  let holdPageThree = false

  await page.route("**/auth/me", (route) =>
    json(route, { data: { id: "u1", email: "ada@example.com" } })
  )
  await page.route("**/catalog/series", (route) =>
    json(route, { data: { series: [], ungroupedExpansionSets: [] } })
  )
  await page.route("**/collections", (route) => {
    if (route.request().resourceType() !== "fetch") return route.fallback()
    return json(route, {
      data: [
        {
          id: "col-1",
          title: "Binder",
          description: "",
          cardCount: 250,
          maxCardCount: 0,
        },
      ],
    })
  })
  await page.route("**/collections/col-1", (route) => {
    if (route.request().resourceType() !== "fetch") return route.fallback()
    return json(route, {
      data: {
        id: "col-1",
        title: "Binder",
        description: "",
        cardCount: 250,
        maxCardCount: 0,
      },
    })
  })
  await page.route("**/collections/col-1/facets*", (route) =>
    json(route, FACETS)
  )
  await page.route("**/collections/col-1/entries**", async (route) => {
    const request = route.request()
    if (request.method() === "OPTIONS") {
      return route.fulfill({ status: 204, headers: cors(route) })
    }
    if (request.method() === "PATCH") {
      const body = request.postDataJSON()
      bulkBodies.push(body)
      const results = body.items.map(
        ({ cardId, quantity }: { cardId: string; quantity: number }) => {
          entries =
            quantity === 0
              ? entries.filter((item) => item.card.id !== cardId)
              : entries.map((item) =>
                  item.card.id === cardId ? { ...item, quantity } : item
                )
          return {
            cardId,
            quantity,
            status: quantity === 0 ? "removed" : "applied",
          }
        }
      )
      return json(route, { data: results })
    }
    const url = new URL(request.url())
    entryRequests.push(url)
    if (holdPageThree && url.searchParams.get("page") === "3") {
      await pageThree
    }
    return json(route, pageOf(route, entries))
  })
  // Master Inventory: the same cards, read-only. Registered after the list
  // route because Playwright tries the latest route first.
  await page.route("**/inventory/cards?*", (route) =>
    json(
      route,
      pageOf(
        route,
        Array.from({ length: 250 }, (_, index) => entry(index, index + 1))
      )
    )
  )
  await page.route("**/inventory/cards/facets*", (route) => json(route, FACETS))

  return {
    entryRequests,
    bulkBodies,
    holdPageThree() {
      holdPageThree = true
    },
    releasePageThree,
  }
}

// Reached by client navigation: a direct load runs the route loader on the dev
// server, where these browser-level stubs don't apply. The dev server can
// hydrate after the first click lands, so retry the click until it navigates.
async function navigateTo(page: Page, link: string, heading: string) {
  await page.goto("/")
  await expect(async () => {
    await page.getByRole("banner").getByRole("link", { name: link }).click()
    await expect(
      page.getByRole("heading", { name: heading, level: 1 })
    ).toBeVisible({ timeout: 1000 })
  }).toPass()
}

async function openCollection(page: Page) {
  await navigateTo(page, "Collections", "Collections")
  await page.getByRole("link", { name: "Binder" }).click()
  await expect(
    page.getByRole("heading", { name: "Binder", level: 1 })
  ).toBeVisible()
}

const scrollToBottom = (page: Page) =>
  page.evaluate(() => window.scrollTo(0, document.documentElement.scrollHeight))

test.describe("Collection entries infinite scroll", () => {
  test("keeps a quantity saved on page 2 after page 3 loads and after a refetch", async ({
    page,
  }) => {
    const api = await stubApi(page)
    api.holdPageThree()
    await openCollection(page)
    await expect(page.getByText("100 of 250 cards loaded")).toBeVisible()

    // Scroll until page 2 is in; page 3 is requested but held.
    await expect
      .poll(
        async () => {
          await scrollToBottom(page)
          return page.getByText("200 of 250 cards loaded").count()
        },
        { timeout: 15_000 }
      )
      .toBe(1)

    // Card 199 is on page 2, in the last rendered row.
    const quantity = page.getByLabel("Quantity of Card 199", { exact: true })
    await expect(async () => {
      await scrollToBottom(page)
      await expect(quantity).toHaveValue("3", { timeout: 1000 })
    }).toPass()
    await page
      .getByRole("button", { name: "Increase quantity of Card 199" })
      .click()
    await expect(quantity).toHaveValue("4")
    await expect.poll(() => api.bulkBodies.length).toBe(1)
    expect(api.bulkBodies[0]).toEqual({
      items: [{ cardId: "card-199", quantity: 4 }],
    })

    api.releasePageThree()
    await expect
      .poll(
        async () => {
          await scrollToBottom(page)
          return page.getByText("250 of 250 cards loaded").count()
        },
        { timeout: 15_000 }
      )
      .toBe(1)

    // Every page was requested (the dev server's StrictMode remount can
    // repeat one, so counts are not asserted), and the URL has no page.
    await expect(page.getByTitle("Card 249")).toBeVisible()
    const pages = api.entryRequests.map((url) => url.searchParams.get("page"))
    expect(new Set(pages)).toEqual(new Set(["1", "2", "3"]))
    expect(page.url()).not.toContain("page=")

    // Scroll back to card 199: still 4, not the stale 3 from page 2's first read.
    await page.evaluate(() => window.scrollTo(0, 0))
    await expect
      .poll(async () => {
        await scrollToBottom(page)
        return page.getByText("250 of 250 cards loaded").count()
      })
      .toBe(1)
    await page.evaluate(() => {
      const row = document.querySelector('[aria-posinset="200"]')
      row?.scrollIntoView({ block: "center" })
    })
    await expect(
      page.getByLabel("Quantity of Card 199", { exact: true })
    ).toHaveValue("4")
  })
})

test.describe("Master Inventory infinite scroll", () => {
  test("loads every page on scroll without duplicates", async ({ page }) => {
    await stubApi(page)
    await navigateTo(page, "Master Inventory", "Master Inventory")
    await expect(page.getByText("100 of 250 cards loaded")).toBeVisible()

    await expect
      .poll(
        async () => {
          await scrollToBottom(page)
          return page.getByText("250 of 250 cards loaded").count()
        },
        { timeout: 15_000 }
      )
      .toBe(1)

    await expect(page.getByText("×250", { exact: true })).toBeVisible()
    expect(page.url()).not.toContain("page=")
    const names = await page
      .getByRole("listitem")
      .locator("p[title]")
      .evaluateAll((nodes) => nodes.map((node) => node.getAttribute("title")))
    expect(new Set(names).size).toBe(names.length)
  })
})
