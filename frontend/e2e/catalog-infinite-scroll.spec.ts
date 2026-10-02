import { test, expect } from "playwright/test"
import type { Page, Route } from "playwright/test"

// Stubs the catalog API with 300 generated cards (60 per page, so five pages)
// instead of the seeded fixture, which is too small to scroll. Alpha cards are
// the even indexes and Beta the odd ones.
const CARDS = Array.from({ length: 300 }, (_, index) => ({
  id: `card-${index}`,
  name: `${index % 2 === 0 ? "Alpha" : "Beta"} ${index}`,
  localId: String(index),
  category: "Pokémon",
  tags: [],
  illustrator: "someone",
  imageUrl: "",
  expansionSet: { id: "set-1", code: "TST", name: "Test Set", imageUrl: "" },
  rarity: { id: "rarity-1", code: "C", name: "Common" },
}))

type StubBody = {
  data: unknown
  meta?: { total: number; page: number; limit: number }
}

function json(route: Route, body: StubBody) {
  return route.fulfill({
    status: 200,
    contentType: "application/json",
    headers: {
      "access-control-allow-origin": route.request().headers()["origin"] ?? "",
      "access-control-allow-credentials": "true",
    },
    body: JSON.stringify(body),
  })
}

async function stubCatalog(page: Page) {
  const requests: URL[] = []
  await page.route("**/auth/me", (route) =>
    route.fulfill({
      status: 401,
      contentType: "application/json",
      headers: {
        "access-control-allow-origin":
          route.request().headers()["origin"] ?? "",
        "access-control-allow-credentials": "true",
      },
      body: JSON.stringify({ status: 401 }),
    })
  )
  await page.route("**/catalog/series", (route) =>
    json(route, { data: { series: [], ungroupedExpansionSets: [] } })
  )
  await page.route("**/catalog/facets*", (route) =>
    json(route, {
      data: { categories: [], expansionSets: [], rarities: [], tags: [] },
    })
  )
  await page.route("**/catalog/cards?*", (route) => {
    const url = new URL(route.request().url())
    requests.push(url)
    const name = (url.searchParams.get("name") ?? "").toLowerCase()
    const pageNumber = Number(url.searchParams.get("page") ?? 1)
    const limit = Number(url.searchParams.get("limit") ?? 24)
    const matches = CARDS.filter((card) =>
      card.name.toLowerCase().includes(name)
    )
    return json(route, {
      data: matches.slice((pageNumber - 1) * limit, pageNumber * limit),
      meta: { total: matches.length, page: pageNumber, limit },
    })
  })
  return requests
}

// Reached by client navigation: a direct load runs the route loader on the dev
// server, where these browser-level stubs don't apply.
async function openSearch(page: Page) {
  await page.goto("/")
  await page
    .getByRole("main")
    .getByRole("link", { name: "Browse the catalog" })
    .click()
  await page.getByRole("link", { name: "search the catalog" }).click()
  await expect(
    page.getByRole("heading", { name: "Search the catalog", level: 1 })
  ).toBeVisible()
}

const scrollToBottom = (page: Page) =>
  page.evaluate(() => window.scrollTo(0, document.documentElement.scrollHeight))

test.describe("Catalog search infinite scroll", () => {
  test("loads every page on scroll without duplicates and keeps the DOM windowed", async ({
    page,
  }) => {
    const requests = await stubCatalog(page)
    await openSearch(page)
    await expect(page.getByText("60 of 300 cards loaded")).toBeVisible()

    await expect
      .poll(
        async () => {
          await scrollToBottom(page)
          return page.getByText("300 of 300 cards loaded").count()
        },
        { timeout: 15_000 }
      )
      .toBe(1)

    await expect(page.getByTitle("Beta 299")).toBeVisible()
    // Each page was requested exactly once, in order, and the URL has no page.
    expect(requests.map((url) => url.searchParams.get("page"))).toEqual([
      "1",
      "2",
      "3",
      "4",
      "5",
    ])
    expect(page.url()).not.toContain("page=")

    const tiles = page.getByRole("listitem")
    expect(await tiles.count()).toBeLessThan(100)
    const names = await tiles
      .locator("p[title]")
      .evaluateAll((nodes) => nodes.map((node) => node.getAttribute("title")))
    expect(new Set(names).size).toBe(names.length)
  })

  test("the Load more button loads the next page from the keyboard", async ({
    page,
  }) => {
    await stubCatalog(page)
    await openSearch(page)
    await expect(page.getByText("60 of 300 cards loaded")).toBeVisible()

    await page.getByRole("button", { name: "Load more" }).focus()
    await page.keyboard.press("Enter")

    await expect(page.getByText("120 of 300 cards loaded")).toBeVisible()
  })

  test("changing a filter starts a fresh list from the top", async ({
    page,
  }) => {
    const requests = await stubCatalog(page)
    await openSearch(page)
    await expect(page.getByText("60 of 300 cards loaded")).toBeVisible()
    await expect
      .poll(async () => {
        await scrollToBottom(page)
        return page.getByText("120 of 300 cards loaded").count()
      })
      .toBe(1)
    expect(await page.evaluate(() => window.scrollY)).toBeGreaterThan(0)

    await page.getByLabel("Card name").fill("Alpha")
    await page.getByRole("button", { name: "Search" }).click()

    await expect(page.getByText("60 of 150 cards loaded")).toBeVisible()
    await expect.poll(() => page.evaluate(() => window.scrollY)).toBe(0)
    await expect(page.getByTitle("Alpha 0")).toBeVisible()
    await expect(page.getByTitle("Beta 1")).toHaveCount(0)
    await expect(page).toHaveURL(/name=Alpha/)
    expect(page.url()).not.toContain("page=")
    const last = requests.at(-1)!
    expect(last.searchParams.get("name")).toBe("Alpha")
    expect(last.searchParams.get("page")).toBe("1")
  })
})
