import AxeBuilder from "@axe-core/playwright"
import { test, expect } from "playwright/test"
import type { Page, Route } from "playwright/test"

// The catalog as a Guest meets it (security-hardening ticket 11), against a
// stubbed API that applies the real lock (ticket 10): one page of at most 24,
// 401 `login_required` for a later page, a rarity, category or tag filter, the
// facets or Collections. Signing in itself is covered by sign-in.spec.ts; here
// the prompts are followed only as far as the sign-in address they build.
const TOTAL = 100
const GUEST_PAGE = 24

const expansionSet = {
  id: "set-1",
  code: "TST",
  name: "Test Set",
  imageUrl: "",
}
const cards = Array.from({ length: TOTAL }, (_, index) => ({
  id: `card-${index}`,
  name: `${index % 2 === 0 ? "Alpha" : "Beta"} ${index}`,
  localId: String(index),
  category: "Pokémon",
  tags: [],
  illustrator: "someone",
  imageUrl: "",
  expansionSet,
  rarity: { id: "rarity-1", code: "C", name: "Common" },
}))

type StubBody = {
  data?: unknown
  meta?: { total: number; page: number; limit: number }
  code?: string
  detail?: string
}

function reply(route: Route, status: number, body: StubBody) {
  return route.fulfill({
    status,
    contentType: "application/json",
    headers: {
      "access-control-allow-origin": route.request().headers()["origin"] ?? "",
    },
    body: JSON.stringify(body),
  })
}

const loginRequired = (route: Route) =>
  reply(route, 401, {
    code: "login_required",
    detail: "sign in to use this part of the catalog",
  })

async function stubGuestCatalog(page: Page) {
  const requests: URL[] = []
  await page.route("**/catalog/series", (route) =>
    reply(route, 200, {
      data: {
        series: [
          {
            id: "series-1",
            code: "TS",
            name: "Test Series",
            expansionSets: [expansionSet],
          },
        ],
        ungroupedExpansionSets: [],
      },
    })
  )
  await page.route("**/catalog/facets*", loginRequired)
  await page.route("**/catalog/cards?*", (route) => {
    const url = new URL(route.request().url())
    requests.push(url)
    const pageNumber = Number(url.searchParams.get("page") ?? 1)
    const locked = ["rarityId", "category", "tag"].some((key) =>
      url.searchParams.has(key)
    )
    if (pageNumber > 1 || locked) return loginRequired(route)
    const limit = Math.min(
      Number(url.searchParams.get("limit") ?? GUEST_PAGE),
      GUEST_PAGE
    )
    const name = (url.searchParams.get("name") ?? "").toLowerCase()
    const matches = cards.filter((card) =>
      card.name.toLowerCase().includes(name)
    )
    return reply(route, 200, {
      data: matches.slice(0, limit),
      meta: { total: matches.length, page: 1, limit },
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

const pagesRequested = (requests: URL[]) =>
  new Set(requests.map((url) => url.searchParams.get("page")))

test.describe("Catalog as a Guest", () => {
  test("shows the filters locked with a prompt, and Expansion Set still open", async ({
    page,
  }) => {
    await stubGuestCatalog(page)
    await openSearch(page)
    await expect(page.getByText("24 of 100 cards loaded")).toBeVisible()

    for (const filter of ["Rarity", "Category", "Tag"]) {
      await expect(page.getByRole("button", { name: filter })).toBeDisabled()
    }
    await expect(
      page.getByRole("button", { name: "Expansion Set" })
    ).toBeEnabled()
    await expect(
      page.getByRole("link", { name: "Sign in to use filters" })
    ).toBeVisible()
    // Name search stays open.
    await page.getByLabel("Card name").fill("Alpha")
    await page.getByRole("button", { name: "Search" }).click()
    await expect(page.getByText("24 of 50 cards loaded")).toBeVisible()
  })

  test("ends the first page with 'Sign in to see more' and never asks for page 2", async ({
    page,
  }) => {
    const requests = await stubGuestCatalog(page)
    await openSearch(page)
    await expect(page.getByText("24 of 100 cards loaded")).toBeVisible()

    await scrollToBottom(page)
    const more = page.getByRole("link", { name: "Sign in to see more" })
    await expect(more).toBeVisible()
    await expect(page.getByRole("button", { name: "Load more" })).toHaveCount(0)
    // Give a (wrong) scroll-triggered load time to show itself.
    await page.waitForTimeout(500)
    expect(pagesRequested(requests)).toEqual(new Set(["1"]))

    // Reachable by keyboard: it is a link, so it takes focus.
    await more.focus()
    await expect(more).toBeFocused()
  })

  test("takes the user to sign in and back to the same search", async ({
    page,
  }) => {
    await stubGuestCatalog(page)
    await openSearch(page)
    await page.getByLabel("Card name").fill("Alpha")
    await page.getByRole("button", { name: "Search" }).click()
    await expect(page.getByText("24 of 50 cards loaded")).toBeVisible()

    await scrollToBottom(page)
    await page.getByRole("link", { name: "Sign in to see more" }).click()

    await expect(page).toHaveURL(/\/auth\/login/)
    expect(new URL(page.url()).searchParams.get("redirect")).toBe(
      "/catalog/search?name=Alpha"
    )
  })

  test("passes axe with the prompts showing", async ({ page }) => {
    await stubGuestCatalog(page)
    await openSearch(page)
    await expect(page.getByText("24 of 100 cards loaded")).toBeVisible()
    await scrollToBottom(page)
    await expect(
      page.getByRole("link", { name: "Sign in to see more" })
    ).toBeVisible()

    expect((await new AxeBuilder({ page }).analyze()).violations).toEqual([])
  })

  test("shows the same prompt at the end of an Expansion Set's first page", async ({
    page,
  }) => {
    const requests = await stubGuestCatalog(page)
    await page.goto("/")
    await page
      .getByRole("main")
      .getByRole("link", { name: "Browse the catalog" })
      .click()
    await page
      .getByRole("link", { name: /Test Set/ })
      .first()
      .click()
    await expect(page.getByText("24 of 100 cards loaded")).toBeVisible()

    await scrollToBottom(page)
    await expect(
      page.getByRole("link", { name: "Sign in to see more" })
    ).toBeVisible()
    expect(pagesRequested(requests)).toEqual(new Set(["1"]))
  })
})
