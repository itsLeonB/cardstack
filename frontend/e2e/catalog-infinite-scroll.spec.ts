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

// `tagsPerCard` makes tiles taller than the grid's estimate, as real tagged
// cards are; `total` limits the stubbed catalog to its first cards.
async function stubCatalog(
  page: Page,
  { tagsPerCard = 0, total = CARDS.length } = {}
) {
  const requests: URL[] = []
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
    const matches = CARDS.slice(0, total).filter(
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

// The first tile whose bottom edge is below the sticky header (about 56px).
const topCard = (page: Page) =>
  page.evaluate(() => {
    const tile = [...document.querySelectorAll("[role=listitem]")].find(
      (node) => node.getBoundingClientRect().bottom > 80
    )
    return {
      position: tile?.getAttribute("aria-posinset") ?? "",
      title: tile?.querySelector("p[title]")?.getAttribute("title") ?? "",
    }
  })

// A tile's top edge in the viewport, or null while it is not rendered.
const cardTop = (page: Page, position: string) =>
  page.evaluate(
    (at) =>
      document
        .querySelector(`[role=listitem][aria-posinset="${at}"]`)
        ?.getBoundingClientRect().top ?? null,
    position
  )

const lowestRenderedPosition = (page: Page) =>
  page.evaluate(() =>
    Math.min(
      ...[...document.querySelectorAll("[role=listitem]")].map((node) =>
        Number(node.getAttribute("aria-posinset"))
      )
    )
  )

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

    expect((await new AxeBuilder({ page }).analyze()).violations).toEqual([])

    // Again with a second page loaded and the list scrolled into it.
    await scrollToBottom(page)
    await expect(page.getByText("120 of 300 cards loaded")).toBeVisible()
    await expect(page.getByTitle("Alpha 58")).toBeVisible()
    expect((await new AxeBuilder({ page }).analyze()).violations).toEqual([])
  })

  test("keeps keyboard focus when its row scrolls out, and tabs on to the footer once Load more is gone", async ({
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
    // With every page loaded, Load more is gone: Tab from the last tile goes
    // straight to the footer.
    await expect
      .poll(async () => {
        await scrollToBottom(page)
        return page.getByText("300 of 300 cards loaded").count()
      })
      .toBe(1)
    await expect(page.getByTitle("Beta 299")).toBeVisible()
    await page.getByRole("listitem").last().getByRole("link").last().focus()
    await expect(page.getByRole("button", { name: "Load more" })).toHaveCount(0)
    await page.keyboard.press("Tab")
    await expect(
      page.getByRole("contentinfo").getByRole("link").first()
    ).toBeFocused()
  })

  for (const [name, from, to] of [
    ["narrowing", 1280, 700],
    ["widening", 700, 1280],
  ] as const) {
    test(`keeps the same card at the top when ${name} across a column breakpoint`, async ({
      page,
    }) => {
      await page.emulateMedia({ reducedMotion: "reduce" })
      await page.setViewportSize({ width: from, height: 800 })
      await stubCatalog(page)
      await openSearch(page)
      await expect(page.getByText("60 of 300 cards loaded")).toBeVisible()
      await page.evaluate(() => window.scrollTo(0, 3000))
      // Wait for the window to move: the first rows are gone from the DOM.
      await expect.poll(() => lowestRenderedPosition(page)).toBeGreaterThan(5)

      // The first tile of the top visible row is the one the grid anchors on.
      const anchor = await topCard(page)
      await page.setViewportSize({ width: to, height: 800 })

      // Re-anchored: its row sits right under the 56px sticky header.
      await expect
        .poll(async () => {
          const top = await cardTop(page, anchor.position)
          return top !== null && top >= 55 && top <= 70
        })
        .toBe(true)
    })
  }

  test("returns to the same place with the loaded pages when navigating back", async ({
    page,
  }) => {
    await stubCatalog(page, { tagsPerCard: 6 })
    await openSearch(page)
    await expect(page.getByText("60 of 300 cards loaded")).toBeVisible()
    await expect
      .poll(async () => {
        await page.evaluate(() => window.scrollBy(0, 2500))
        return loadedCount(page)
      })
      .toBeGreaterThanOrEqual(120)
    // Settled: the scroll position is the same two frames apart.
    await page.waitForFunction(
      () =>
        new Promise<boolean>((done) => {
          const y = window.scrollY
          requestAnimationFrame(() =>
            requestAnimationFrame(() => done(window.scrollY === y))
          )
        })
    )
    const before = await page.evaluate(() => window.scrollY)
    const loaded = await loadedCount(page)
    const firstVisible = await topCard(page)

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

    // The cached pages are back at once, and the same card is at the top.
    await expect(page.getByText(`${loaded} of 300 cards loaded`)).toBeVisible()
    await expect.poll(() => topCard(page)).toEqual(firstVisible)
    expect(
      Math.abs((await page.evaluate(() => window.scrollY)) - before)
    ).toBeLessThan(100)
  })

  test("Tab walks every tile, then Load more (still live), then the footer, never losing focus", async ({
    page,
  }) => {
    // 66 cards: the first page is full (60), so Load more has a next page.
    await stubCatalog(page, { total: 66 })
    await openSearch(page)
    await expect(page.getByText("60 of 66 cards loaded")).toBeVisible()
    const loadMore = page.getByRole("button", { name: "Load more" })
    await expect(loadMore).toHaveAttribute("aria-disabled", "false")

    const where = () =>
      page.evaluate(() => {
        const active = document.activeElement
        return {
          lost: !active || active === document.body,
          inGrid: Boolean(active?.closest("[role=list]")),
          loadMore: active?.textContent === "Load more",
          status: document.querySelector("[aria-live=polite]")?.textContent,
        }
      })

    await page.getByLabel("Card name").focus()
    let enteredGrid = false
    let reachedLoadMore = false
    // 2 links per tile, 60 tiles, plus the filter controls before the grid.
    for (let press = 0; press < 200 && !reachedLoadMore; press++) {
      await page.keyboard.press("Tab")
      const now = await where()
      expect(now.lost).toBe(false)
      if (now.inGrid) enteredGrid = true
      if (enteredGrid && !now.loadMore) {
        expect(now.inGrid).toBe(true)
        // Scroll loading stays paused while tabbing, though the list's end is
        // repeatedly scrolled into view.
        expect(now.status).toBe("60 of 66 cards loaded")
      }
      reachedLoadMore = now.loadMore
    }
    expect(enteredGrid).toBe(true)
    expect(reachedLoadMore).toBe(true)

    await page.keyboard.press("Tab")
    await expect(
      page.getByRole("contentinfo").getByRole("link").first()
    ).toBeFocused()
  })
})

test.describe("Expansion Set infinite scroll", () => {
  test("loads every card of the set on scroll without duplicates", async ({
    page,
  }) => {
    const requests = await stubCatalog(page)
    // Registered after stubCatalog's own series stub, so it wins.
    await page.route("**/catalog/series", (route) =>
      json(route, {
        data: {
          series: [],
          ungroupedExpansionSets: [
            { id: "set-1", code: "TST", name: "Test Set", imageUrl: "" },
          ],
        },
      })
    )
    // Reached by client navigation: see openSearch.
    await page.goto("/")
    await page
      .getByRole("main")
      .getByRole("link", { name: "Browse the catalog" })
      .click()
    await page.getByRole("link", { name: /Test Set/ }).click()
    await expect(
      page.getByRole("heading", { name: "Test Set", level: 1 })
    ).toBeVisible()
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
    expect(requests.map((url) => url.searchParams.get("page"))).toEqual([
      "1",
      "2",
      "3",
      "4",
      "5",
    ])
    expect(
      requests.every((url) => url.searchParams.get("expansionSetId"))
    ).toBe(true)
    expect(page.url()).not.toContain("page=")

    const names = await page
      .getByRole("listitem")
      .locator("p[title]")
      .evaluateAll((nodes) => nodes.map((node) => node.getAttribute("title")))
    expect(new Set(names).size).toBe(names.length)
  })

  test("a legacy ?page= link opens the first page of the set", async ({
    page,
  }) => {
    const requests = await stubCatalog(page)
    await page.goto("/catalog/sets/set-1?page=3")

    await expect(
      page.getByRole("heading", { name: "Test Set", level: 1 })
    ).toBeVisible()
    await expect(page.getByText("60 of 300 cards loaded")).toBeVisible()
    expect(requests[0]?.searchParams.get("page")).toBe("1")
  })
})
