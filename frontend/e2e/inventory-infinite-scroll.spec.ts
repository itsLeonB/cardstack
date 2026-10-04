import { test, expect } from "playwright/test"
import type { Page, Route } from "playwright/test"

// Stubs a signed-in user with one Collection of 350 entries (100 per page, so
// four pages) and the same 350 cards as the Master Inventory, instead of the
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
  let entries = Array.from({ length: 350 }, (_, index) => entry(index))
  const entryRequests: URL[] = []
  const bulkBodies: { items: { cardId: string; quantity: number }[] }[] = []
  // Page 4 of the Collection is answered only once the test lets it.
  let releasePageFour = () => {}
  const pageFour = new Promise<void>((resolve) => (releasePageFour = resolve))
  let holdPageFour = false

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
          cardCount: 350,
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
        cardCount: 350,
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
    if (holdPageFour && url.searchParams.get("page") === "4") {
      await pageFour
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
        Array.from({ length: 350 }, (_, index) => entry(index, index + 1))
      )
    )
  )
  await page.route("**/inventory/cards/facets*", (route) => json(route, FACETS))

  return {
    entryRequests,
    bulkBodies,
    holdPageFour() {
      holdPageFour = true
    },
    releasePageFour,
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

// FIXME(ticket 09): both suites need a real Clerk session (Clerk Testing
// Tokens); stubApi's /auth/me stub simulated one the API no longer has.
test.describe.fixme("Collection entries infinite scroll", () => {
  test("keeps a quantity saved on page 3 after page 4 loads and after a focus refetch", async ({
    page,
  }) => {
    const api = await stubApi(page)
    api.holdPageFour()
    await openCollection(page)
    await expect(page.getByText("100 of 350 cards loaded")).toBeVisible()

    // Scroll until page 3 is in; page 4 is requested but held.
    await expect
      .poll(
        async () => {
          await scrollToBottom(page)
          return page.getByText("300 of 350 cards loaded").count()
        },
        { timeout: 15_000 }
      )
      .toBe(1)

    // Card 299 is on page 3, in the last rendered row.
    const quantity = page.getByLabel("Quantity of Card 299", { exact: true })
    await expect(async () => {
      await scrollToBottom(page)
      await expect(quantity).toHaveValue("3", { timeout: 1000 })
    }).toPass()
    await page
      .getByRole("button", { name: "Increase quantity of Card 299" })
      .click()
    await expect(quantity).toHaveValue("4")
    await expect.poll(() => api.bulkBodies.length).toBe(1)
    expect(api.bulkBodies[0]).toEqual({
      items: [{ cardId: "card-299", quantity: 4 }],
    })

    api.releasePageFour()
    await expect
      .poll(
        async () => {
          await scrollToBottom(page)
          return page.getByText("350 of 350 cards loaded").count()
        },
        { timeout: 15_000 }
      )
      .toBe(1)

    // Every page was requested (the dev server's StrictMode remount can
    // repeat one, so counts are not asserted), and the URL has no page.
    await expect(async () => {
      await scrollToBottom(page)
      await expect(page.getByTitle("Card 349")).toBeVisible({ timeout: 1000 })
    }).toPass()
    const pages = api.entryRequests.map((url) => url.searchParams.get("page"))
    expect(new Set(pages)).toEqual(new Set(["1", "2", "3", "4"]))
    expect(page.url()).not.toContain("page=")

    // Scroll back to card 199: still 4, not the stale 3 from page 3's first read.
    await page.evaluate(() => window.scrollTo(0, 0))
    await expect
      .poll(async () => {
        await scrollToBottom(page)
        return page.getByText("350 of 350 cards loaded").count()
      })
      .toBe(1)
    await page.evaluate(() => {
      const row = document.querySelector('[aria-posinset="300"]')
      row?.scrollIntoView({ block: "center" })
    })
    await expect(
      page.getByLabel("Quantity of Card 299", { exact: true })
    ).toHaveValue("4")

    // Refocusing the tab refetches every loaded page; the saved value is the server's.
    const before = api.entryRequests.length
    await page.evaluate(() => {
      const setVisibility = (state: string) => {
        Object.defineProperty(document, "visibilityState", {
          configurable: true,
          get: () => state,
        })
        document.dispatchEvent(new Event("visibilitychange", { bubbles: true }))
      }
      setVisibility("hidden")
      setVisibility("visible")
    })
    await expect
      .poll(() => api.entryRequests.length - before, { timeout: 15_000 })
      .toBeGreaterThanOrEqual(4)
    await page.evaluate(() => {
      const row = document.querySelector('[aria-posinset="300"]')
      row?.scrollIntoView({ block: "center" })
    })
    await expect(
      page.getByLabel("Quantity of Card 299", { exact: true })
    ).toHaveValue("4")
  })
})

test.describe.fixme("Master Inventory infinite scroll", () => {
  test("loads every page on scroll without duplicates", async ({ page }) => {
    await stubApi(page)
    await navigateTo(page, "Master Inventory", "Master Inventory")
    await expect(page.getByText("100 of 350 cards loaded")).toBeVisible()

    await expect
      .poll(
        async () => {
          await scrollToBottom(page)
          return page.getByText("350 of 350 cards loaded").count()
        },
        { timeout: 15_000 }
      )
      .toBe(1)

    await expect(page.getByText("×350", { exact: true })).toBeVisible()
    expect(page.url()).not.toContain("page=")
    const names = await page
      .getByRole("listitem")
      .locator("p[title]")
      .evaluateAll((nodes) => nodes.map((node) => node.getAttribute("title")))
    expect(new Set(names).size).toBe(names.length)
  })
})
