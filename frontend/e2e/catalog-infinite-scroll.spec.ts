import AxeBuilder from "@axe-core/playwright"
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

// `tagsPerCard` makes tiles taller than the grid's estimate, as real tagged cards are.
async function stubCatalog(page: Page, tagsPerCard = 0) {
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
    const localId = url.searchParams.get("localId")
    const matches = CARDS.filter(
      (card) =>
        card.name.toLowerCase().includes(name) &&
        (!localId || card.localId === localId)
    )
    return json(route, {
      data: matches
        .slice((pageNumber - 1) * limit, pageNumber * limit)
        .map((card) => ({
          ...card,
          tags: Array.from({ length: tagsPerCard }, (_, tag) => `Tag ${tag}`),
        })),
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

// The accessibility and place-keeping behaviour of the virtualized grid.
test.describe("Catalog search grid", () => {
  const loadedCount = async (page: Page) => {
    const text = await page.getByText(/ of 300 cards loaded/).textContent()
    return Number(text?.split(" ")[0])
  }

  test("exposes each tile's position and the total", async ({ page }) => {
    await stubCatalog(page)
    await openSearch(page)
    await expect(page.getByText("60 of 300 cards loaded")).toBeVisible()

    const first = page.getByRole("listitem").first()
    await expect(first).toHaveAttribute("aria-posinset", "1")
    await expect(first).toHaveAttribute("aria-setsize", "300")

    await scrollToBottom(page)
    await expect(page.getByText("120 of 300 cards loaded")).toBeVisible()
    // A tile deep in the list reports its place in the whole set, not its row.
    await expect(page.getByTitle("Alpha 58")).toBeVisible()
    await expect(
      page.getByRole("listitem").filter({ has: page.getByTitle("Alpha 58") })
    ).toHaveAttribute("aria-posinset", "59")
  })

  test("passes axe", async ({ page }) => {
    await stubCatalog(page)
    await openSearch(page)
    await expect(page.getByText("60 of 300 cards loaded")).toBeVisible()

    const results = await new AxeBuilder({ page }).analyze()
    expect(results.violations).toEqual([])
  })

  test("keeps keyboard focus when its row scrolls out, and tabs on to Load more and the footer", async ({
    page,
  }) => {
    await stubCatalog(page)
    await openSearch(page)
    await expect(page.getByText("60 of 300 cards loaded")).toBeVisible()

    const firstLink = page.getByRole("link", { name: "Alpha 0" }).first()
    await firstLink.focus()
    await scrollToBottom(page)
    await expect(page.getByText("120 of 300 cards loaded")).toBeVisible()
    await expect(firstLink).toBeFocused()
    // With every page loaded, the last tile is the end of the list: Tab goes to
    // Load more (inert but focusable), and the next Tab to the footer.
    await expect
      .poll(async () => {
        await scrollToBottom(page)
        return page.getByText("300 of 300 cards loaded").count()
      })
      .toBe(1)
    await expect(page.getByTitle("Beta 299")).toBeVisible()
    await page.getByRole("listitem").last().getByRole("link").last().focus()
    await page.keyboard.press("Tab")
    await expect(page.getByRole("button", { name: "Load more" })).toBeFocused()
    await page.keyboard.press("Tab")
    await expect(
      page.getByRole("contentinfo").getByRole("link").first()
    ).toBeFocused()
  })

  test("keeps the same card in view when resizing across a column breakpoint", async ({
    page,
  }) => {
    await page.emulateMedia({ reducedMotion: "reduce" })
    await page.setViewportSize({ width: 1280, height: 800 })
    await stubCatalog(page)
    await openSearch(page)
    await expect(page.getByText("60 of 300 cards loaded")).toBeVisible()
    await page.evaluate(() => window.scrollTo(0, 3000))

    // The first tile of the top visible row is the one the grid anchors on.
    // Poll: the grid renders the new window a frame after the scroll.
    let anchor: string | null = null
    await expect
      .poll(async () => {
        anchor = await page.evaluate(() => {
          const tiles = [...document.querySelectorAll("[role=listitem]")]
          const top = tiles.find(
            (tile) => tile.getBoundingClientRect().bottom > 80
          )
          return top?.getAttribute("aria-posinset") ?? null
        })
        return anchor
      })
      .not.toBeNull()

    await page.setViewportSize({ width: 700, height: 800 })

    const tile = page.locator(`[role=listitem][aria-posinset="${anchor}"]`)
    await expect(tile).toBeInViewport()
    expect(
      await tile.evaluate((node) => node.getBoundingClientRect().top)
    ).toBeLessThan(400)
  })

  test("returns to the same place with the loaded pages when navigating back", async ({
    page,
  }) => {
    await stubCatalog(page, 6)
    await openSearch(page)
    await expect(page.getByText("60 of 300 cards loaded")).toBeVisible()
    await expect
      .poll(async () => {
        await page.evaluate(() => window.scrollBy(0, 2500))
        return loadedCount(page)
      })
      .toBeGreaterThanOrEqual(120)
    // Let the scroll position settle before leaving.
    await page.waitForTimeout(300)
    const before = await page.evaluate(() => window.scrollY)
    const loaded = await loadedCount(page)

    // A link well inside the viewport, so clicking it does not scroll first.
    const href = await page.evaluate(() => {
      const links = [
        ...document.querySelectorAll<HTMLAnchorElement>(
          "[role=listitem] a[href^='/catalog/cards/']"
        ),
      ]
      return links
        .find((link) => {
          const { top, bottom } = link.getBoundingClientRect()
          return top > 150 && bottom < 650
        })
        ?.getAttribute("href")
    })
    expect(href).toBeTruthy()
    await page.locator(`a[href="${href}"]`).click()
    await expect(page).toHaveURL(/\/catalog\/cards\//)
    await page.goBack()

    await expect(page.getByText(`${loaded} of 300 cards loaded`)).toBeVisible()
    await expect
      .poll(() => page.evaluate(() => window.scrollY))
      .toBeGreaterThan(before - 300)
    expect(await page.evaluate(() => window.scrollY)).toBeLessThan(before + 300)
  })
})
