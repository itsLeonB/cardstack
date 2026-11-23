import AxeBuilder from "@axe-core/playwright"
import { test, expect } from "playwright/test"
import type { Page, Route } from "playwright/test"

import {
  SIGNED_IN_TAG,
  signInAsTestUser,
  useSignedInSuite,
} from "./support/clerk-auth"

// Issue 01: a Trade exists, is readable by its link, and one second seat is
// joinable. The API answers from stubs, so the spec needs no seeded backend;
// the seat race and the refusals are the routes tests' job (backend). The
// Trade page is an SPA route (ssr: false), so its loader runs in the browser
// and direct loads see browser-level stubs.
const SLUG = "trade-link-7h2q"
// The API origin, not `**/trades/...`: that glob would also catch the app's
// own page URL /trades/{slug} and answer the navigation itself with JSON.
const API = "http://api:8080"

// The JSON bodies the API answers with: an envelope, or an error code.
interface ApiBody {
  data?: unknown
  code?: string
  detail?: string
  meta?: { total: number; page: number; limit: number }
}

function reply(route: Route, status: number, body: ApiBody) {
  return route.fulfill({
    status,
    contentType: "application/json",
    headers: {
      // Echo the caller's origin instead of hard-coding the port.
      "access-control-allow-origin": route.request().headers()["origin"] ?? "",
      "access-control-allow-headers": "authorization, content-type",
      "access-control-allow-methods": "GET, POST, OPTIONS",
    },
    body: JSON.stringify(body),
  })
}

interface TradeViewStub {
  slug: string
  status: string
  ownerName: string
  otherName?: string
  ownerItems: object[]
  otherItems: object[]
  isOwner: boolean
  isParticipant: boolean
}

function tradeView(overrides: Partial<TradeViewStub> = {}): TradeViewStub {
  return {
    slug: SLUG,
    status: "open",
    ownerName: "Ash Ketchum",
    ownerItems: [],
    otherItems: [],
    isOwner: false,
    isParticipant: false,
    ...overrides,
  }
}

// The caller is the owner: creation, the link, and the read by slug.
test.describe("Trade owner", { tag: SIGNED_IN_TAG }, () => {
  useSignedInSuite()

  test("starts a Trade, gets its link, and the page renders it by slug", async ({
    page,
  }) => {
    await page.route(`${API}/trades`, (route) =>
      reply(route, 201, { data: { slug: SLUG } })
    )
    await page.route(`${API}/trades/${SLUG}`, (route) =>
      reply(route, 200, {
        data: tradeView({ isOwner: true, isParticipant: true }),
      })
    )

    await signInAsTestUser(page)
    await page.goto("/trades")
    await page.getByRole("button", { name: "Start a Trade" }).click()

    await expect(
      page.getByRole("heading", { name: "New trade", level: 1 })
    ).toBeVisible()
    await expect(page).toHaveTitle("New trade · Cardstack")
    await expect(page.getByText("Open")).toBeVisible()
    await expect(
      page.getByRole("heading", { name: "You", level: 2 })
    ).toBeVisible()
    await expect(page.getByText("Waiting for someone to join")).toBeVisible()
    await expect(page.getByText("Ash Ketchum")).toHaveCount(0)
    // Sharing delivers the link without exposing a publication switch.
    const origin = new URL(page.url()).origin
    const share = page.getByRole("button", { name: "Invite" })
    for (const theme of ["Light", "Dark"]) {
      await page
        .getByRole("banner")
        .getByRole("button", { name: "Theme" })
        .click()
      await page.getByRole("menuitemradio", { name: theme }).click()
      await page.keyboard.press("Escape")
      await expect(page.getByRole("menu")).toBeHidden()
      await expect(
        page.getByRole("heading", { name: "New trade", level: 1 })
      ).toBeVisible()
      await expect(
        page.getByRole("heading", { name: "You", level: 2 })
      ).toBeVisible()
      await expect(page.getByText("Open")).toBeVisible()
      await share.press("Enter")
      const dialog = page.getByRole("dialog", { name: "Share Trade link" })
      await expect(
        dialog.getByRole("textbox", { name: "Trade link" })
      ).toHaveValue(`${origin}/trades/${SLUG}`)
      await expect(dialog.getByRole("button", { name: "Copy" })).toBeVisible()
      await expect(dialog.getByRole("switch")).toHaveCount(0)
      const { violations } = await new AxeBuilder({ page }).analyze()
      expect(violations).toEqual([])
      await page.keyboard.press("Escape")
      await expect(dialog).toBeHidden()
      await expect(share).toBeFocused()
    }
    await page.setViewportSize({ width: 375, height: 700 })
    await expect(
      page.getByRole("heading", { name: "New trade", level: 1 })
    ).toBeVisible()
    await expect(
      page.getByRole("heading", { name: "You", level: 2 })
    ).toBeVisible()
    await expect(page.getByText("Open")).toBeVisible()
    await expect(share).toBeVisible()
    await expect(
      page.getByRole("navigation", { name: "Breadcrumb" })
    ).toBeVisible()
    const { violations: mobileViolations } = await new AxeBuilder({
      page,
    }).analyze()
    expect(mobileViolations).toEqual([])

    // Trade pages stay unindexed.
    await expect(page.locator('meta[name="robots"]')).toHaveAttribute(
      "content",
      "noindex"
    )
  })

  test("an unknown slug reads the same not-found as any other missing Trade", async ({
    page,
  }) => {
    await page.route(`${API}/trades/unknown-slug`, (route) =>
      reply(route, 404, {
        code: "trade_not_found",
        detail: "Trade not found.",
      })
    )

    await signInAsTestUser(page)
    await page.goto("/trades/unknown-slug")

    await expect(
      page.getByRole("heading", { name: "Trade not found" })
    ).toBeVisible()
  })
})

// The caller holds neither seat: read what is offered, then take the second
// seat; a join that loses the race is told the Trade is full, and the
// refetch answers the loser's 404 like any other non-participant read.
test.describe("Trade joiner", { tag: SIGNED_IN_TAG }, () => {
  useSignedInSuite()

  test("reads the offer before joining, and joining moves it to negotiating", async ({
    page,
  }) => {
    // The read answers fresh on every visit: after the join lands, the page
    // refetches and sees the joined state.
    let joined = false
    await page.route(`${API}/trades/${SLUG}`, (route) =>
      reply(route, 200, {
        data: tradeView(
          joined
            ? {
                status: "negotiating",
                otherName: "Ash Ketchum",
                isParticipant: true,
              }
            : {}
        ),
      })
    )
    await page.route(`${API}/trades/${SLUG}/join`, (route) => {
      joined = true
      return reply(route, 200, {
        data: tradeView({
          status: "negotiating",
          otherName: "Ash Ketchum",
          isParticipant: true,
        }),
      })
    })

    await signInAsTestUser(page)
    await page.goto(`/trades/${SLUG}`)

    // Read without claiming anything: the seat is still free.
    await expect(
      page.getByRole("button", { name: "Join this Trade" })
    ).toBeVisible()
    await page.getByRole("button", { name: "Join this Trade" }).click()

    await expect(page.getByText("Negotiating")).toBeVisible()
    await expect(page.getByText("Ash Ketchum").nth(1)).toBeVisible()
    await expect(
      page.getByRole("heading", { name: "You", level: 2 })
    ).toBeVisible()
    await expect(page.getByRole("button", { name: "Invite" })).toHaveCount(0)
    await expect(
      page.getByRole("button", { name: "Join this Trade" })
    ).toHaveCount(0)
  })

  test("names the counterparty and hides Invite once someone joins", async ({
    page,
  }) => {
    await page.route(`${API}/trades/${SLUG}`, (route) =>
      reply(route, 200, {
        data: tradeView({
          status: "negotiating",
          isOwner: true,
          isParticipant: true,
          otherName: "Misty",
        }),
      })
    )

    await signInAsTestUser(page)
    await page.goto(`/trades/${SLUG}`)

    // The Trade is visibly present before the absent action is judged.
    await expect(
      page.getByRole("heading", { name: "Misty", level: 1 })
    ).toBeVisible()
    await expect(page).toHaveTitle("Misty · Cardstack")
    await expect(page.getByText("Negotiating")).toBeVisible()
    await expect(
      page.getByRole("heading", { name: "You", level: 2 })
    ).toBeVisible()
    await expect(
      page.getByRole("heading", { name: "Misty", level: 2 })
    ).toBeVisible()
    await expect(page.getByText("Waiting for someone to join")).toHaveCount(0)

    await expect(page.getByRole("button", { name: "Invite" })).toHaveCount(0)

    for (const theme of ["Light", "Dark"]) {
      await page
        .getByRole("banner")
        .getByRole("button", { name: "Theme" })
        .click()
      await page.getByRole("menuitemradio", { name: theme }).click()
      await page.keyboard.press("Escape")
      await expect(
        page.getByRole("heading", { name: "Misty", level: 1 })
      ).toBeVisible()
      await expect(
        page.getByRole("heading", { name: "You", level: 2 })
      ).toBeVisible()
      await expect(page.getByText("Negotiating")).toBeVisible()
      await expect(page.getByRole("button", { name: "Invite" })).toHaveCount(0)
    }

    await page.setViewportSize({ width: 375, height: 700 })
    await expect(
      page.getByRole("heading", { name: "Misty", level: 1 })
    ).toBeVisible()
    await expect(
      page.getByRole("heading", { name: "You", level: 2 })
    ).toBeVisible()
    await expect(page.getByRole("button", { name: "Invite" })).toHaveCount(0)
  })

  test("a join that loses the seat is told the Trade is full", async ({
    page,
  }) => {
    // Once someone else holds the seat the real API answers this caller's
    // read with the same 404 as an unknown slug (a non-participant) — but
    // the caller who just saw the offer keeps it, with the full notice.
    let joined = false
    await page.route(`${API}/trades/${SLUG}`, (route) =>
      joined
        ? reply(route, 404, { code: "trade_not_found" })
        : reply(route, 200, { data: tradeView() })
    )
    await page.route(`${API}/trades/${SLUG}/join`, (route) => {
      joined = true
      return reply(route, 409, {
        code: "trade_seat_taken",
        detail: "Seat taken.",
      })
    })

    await signInAsTestUser(page)
    await page.goto(`/trades/${SLUG}`)
    const seatRead = page.waitForResponse(
      (response) =>
        response.url() === `${API}/trades/${SLUG}` && response.status() === 404
    )
    await page.getByRole("button", { name: "Join this Trade" }).click()
    await seatRead

    await expect(page.getByText("This Trade is already full.")).toBeVisible()
    await expect(page.getByText("Ash Ketchum")).toBeVisible()
    await expect(
      page.getByRole("button", { name: "Join this Trade" })
    ).toHaveCount(0)
    await expect(
      page.getByRole("heading", { name: "Trade not found" })
    ).toHaveCount(0)
  })
})

// A Guest opening a Trade link gets the same not-found as an unknown slug:
// the API answers 404 and the page renders it, never a login redirect.
test.describe("Trade guest", () => {
  test("a guest reads the same not-found as an unknown slug", async ({
    page,
  }) => {
    await page.route(`${API}/trades/${SLUG}`, (route) =>
      reply(route, 404, {
        code: "trade_not_found",
        detail: "Trade not found.",
      })
    )

    await page.goto(`/trades/${SLUG}`)

    await expect(
      page.getByRole("heading", { name: "Trade not found" })
    ).toBeVisible()
    // The guest's only way in is signing in; the 404 itself says nothing.
    await expect(page.getByRole("link", { name: "Sign in" })).toBeVisible()
    await expect(page).not.toHaveURL(/\/auth\/login/)
  })
})

// The picker is the only way to add a Trade Item. These stubs make the round
// trip deterministic: the catalog, the caller's Collections and the Trade read
// answer locally, and the add POST records what the page submitted.
const PICKER_CARD = {
  id: "card-pikachu",
  name: "Pikachu",
  localId: "025",
  category: "Pokémon",
  tags: [],
  imageUrl: "",
  expansionSet: { id: "set-1", code: "BASE", name: "Base Set", imageUrl: "" },
  rarity: { id: "rarity-1", code: "R", name: "Rare" },
}

interface PickerOptions {
  trade?: Partial<TradeViewStub>
  collections?: { id: string; title: string }[]
  refusal?: { status: number; detail: string }
}

async function stubPicker(page: Page, options: PickerOptions = {}) {
  const posts: unknown[] = []
  const collections = options.collections ?? [
    { id: "collection-1", title: "Binder" },
  ]
  let items: object[] = []
  const trade = tradeView({
    status: "negotiating",
    isOwner: true,
    isParticipant: true,
    otherName: "Misty",
    ...options.trade,
  })
  await page.route(`${API}/trades/${SLUG}`, (route) =>
    reply(route, 200, { data: { ...trade, ownerItems: items } })
  )
  await page.route(`${API}/collections`, (route) =>
    reply(route, 200, {
      data: collections,
    })
  )
  await page.route(`${API}/trades/${SLUG}/items`, (route) => {
    if (route.request().method() === "OPTIONS") return reply(route, 200, {})
    const body = JSON.parse(route.request().postData() ?? "{}")
    posts.push(body)
    if (options.refusal)
      return reply(route, options.refusal.status, {
        detail: options.refusal.detail,
      })
    items = [
      ...items,
      {
        id: `item-${items.length + 1}`,
        card: PICKER_CARD,
        quantity: body.quantity,
        collection: collections.find((c) => c.id === body.sourceCollectionId)
          ?.title,
        available: body.quantity,
        short: false,
      },
    ]
    return reply(route, 200, { data: { ...trade, ownerItems: items } })
  })
  await page.route("**/catalog/series", (route) =>
    reply(route, 200, { data: { series: [], ungroupedExpansionSets: [] } })
  )
  await page.route("**/catalog/facets*", (route) =>
    reply(route, 200, {
      data: { categories: [], expansionSets: [], rarities: [], tags: [] },
    })
  )
  await page.route("**/catalog/cards?*", (route) => {
    if (route.request().method() === "OPTIONS") return reply(route, 200, {})
    const url = new URL(route.request().url())
    const name = (url.searchParams.get("name") ?? "").toLowerCase()
    const pageNumber = Number(url.searchParams.get("page") ?? 1)
    const limit = Number(url.searchParams.get("limit") ?? 24)
    const matches = [PICKER_CARD].filter((card) =>
      card.name.toLowerCase().includes(name)
    )
    return reply(route, 200, {
      data: matches.slice((pageNumber - 1) * limit, pageNumber * limit),
      meta: { total: matches.length, page: pageNumber, limit },
    })
  })
  return { posts }
}

async function openPicker(page: Page) {
  await page.getByRole("link", { name: "Add Card", exact: true }).click()
  await expect(page).toHaveURL(/\/add-item$/)
  await expect(
    page.getByRole("heading", { name: "Add a Card", level: 1 })
  ).toBeVisible()
}

async function selectPickerCard(page: Page) {
  await page.getByLabel("Card name").fill("Pikachu")
  await page.getByRole("button", { name: "Search", exact: true }).click()
  await page
    .getByRole("button", { name: "Offer Pikachu, BASE No. 025", exact: true })
    .click()
}

test.describe("Trade item picker", { tag: SIGNED_IN_TAG }, () => {
  useSignedInSuite()

  test("opens from the detail page, adds from the catalog and returns with the item", async ({
    page,
  }) => {
    const { posts } = await stubPicker(page)
    await signInAsTestUser(page)
    await page.goto(`/trades/${SLUG}`)
    await expect(
      page.getByRole("heading", { name: "Misty", level: 1 })
    ).toBeVisible()

    // Back out of the picker before submitting: nothing is added.
    await openPicker(page)
    await page.goBack()
    await expect(page).toHaveURL(new RegExp(`/trades/${SLUG}$`))
    expect(posts).toEqual([])

    await openPicker(page)
    await expect(page.getByLabel("Card name")).toBeVisible()
    await selectPickerCard(page)
    await page.getByRole("combobox", { name: "Source Collection" }).click()
    await page.getByRole("option", { name: "Binder" }).click()
    await page.getByLabel("Quantity", { exact: true }).fill("2")
    await page.getByRole("button", { name: "Add Card", exact: true }).click()

    await expect(page).toHaveURL(new RegExp(`/trades/${SLUG}$`))
    await expect(page.getByText("Pikachu × 2")).toBeVisible()
    expect(posts).toEqual([
      {
        cardId: "card-pikachu",
        quantity: 2,
        sourceCollectionId: "collection-1",
      },
    ])
  })

  test("offers the same Card from two different Collections", async ({
    page,
  }) => {
    const { posts } = await stubPicker(page, {
      collections: [
        { id: "collection-1", title: "Binder" },
        { id: "collection-2", title: "Box" },
      ],
    })
    await signInAsTestUser(page)
    await page.goto(`/trades/${SLUG}`)

    for (const [title, id] of [
      ["Binder", "collection-1"],
      ["Box", "collection-2"],
    ] as const) {
      await openPicker(page)
      await selectPickerCard(page)
      await page.getByRole("combobox", { name: "Source Collection" }).click()
      await page.getByRole("option", { name: title }).click()
      await page.getByRole("button", { name: "Add Card", exact: true }).click()
      await expect(page).toHaveURL(new RegExp(`/trades/${SLUG}$`))
      expect(posts.at(-1)).toEqual({
        cardId: "card-pikachu",
        quantity: 1,
        sourceCollectionId: id,
      })
    }
    await expect(
      page.getByRole("button", { name: "Remove Pikachu", exact: true })
    ).toHaveCount(2)
  })

  test("names the cause when the API refuses an add", async ({ page }) => {
    const { posts } = await stubPicker(page, {
      refusal: { status: 409, detail: "trade item already exists" },
    })
    await signInAsTestUser(page)
    await page.goto(`/trades/${SLUG}`)
    await openPicker(page)
    await selectPickerCard(page)
    await page.getByRole("combobox", { name: "Source Collection" }).click()
    await page.getByRole("option", { name: "Binder" }).click()

    await page.getByLabel("Quantity", { exact: true }).fill("0")
    await page.getByRole("button", { name: "Add Card", exact: true }).click()
    await expect(page.getByRole("alert")).toContainText(
      "Quantity must be at least 1"
    )
    expect(posts).toEqual([])

    await page.getByLabel("Quantity", { exact: true }).fill("1")
    await page.getByRole("button", { name: "Add Card", exact: true }).click()
    await expect(page.getByRole("alert")).toContainText(
      "trade item already exists"
    )
    expect(posts).toHaveLength(1)
  })

  test("a Collection the caller does not own is refused", async ({ page }) => {
    const { posts } = await stubPicker(page, {
      collections: [{ id: "foreign-1", title: "Someone else's Binder" }],
      refusal: { status: 404, detail: "collection not found" },
    })
    await signInAsTestUser(page)
    await page.goto(`/trades/${SLUG}`)
    await openPicker(page)
    await selectPickerCard(page)
    await page.getByRole("combobox", { name: "Source Collection" }).click()
    await page.getByRole("option", { name: "Someone else's Binder" }).click()
    await page.getByRole("button", { name: "Add Card", exact: true }).click()

    await expect(page.getByRole("alert")).toContainText("collection not found")
    expect(posts).toHaveLength(1)
  })

  test("a locked Trade's picker refuses the add before any request", async ({
    page,
  }) => {
    const { posts } = await stubPicker(page, { trade: { status: "locked" } })
    await signInAsTestUser(page)
    await page.goto(`/trades/${SLUG}/add-item`)

    await expect(page.getByRole("alert")).toContainText(
      "trade items can only change while negotiating"
    )
    await expect(
      page.getByRole("button", { name: "Add Card", exact: true })
    ).toHaveCount(0)
    expect(posts).toEqual([])
  })

  test("a signed-in non-participant gets the same not-found on the picker", async ({
    page,
  }) => {
    await page.route(`${API}/trades/unknown-slug`, (route) =>
      reply(route, 404, { code: "trade_not_found", detail: "Trade not found." })
    )
    await signInAsTestUser(page)
    await page.goto("/trades/unknown-slug/add-item")

    await expect(
      page.getByRole("heading", { name: "Trade not found" })
    ).toBeVisible()
    await expect(page.getByRole("link", { name: "Sign in" })).toHaveCount(0)
  })
})

// A Guest cannot use the picker to discover a Trade either.
test.describe("Trade item picker guest", () => {
  test("a guest gets the same not-found as on the detail page", async ({
    page,
  }) => {
    await page.route(`${API}/trades/${SLUG}`, (route) =>
      reply(route, 404, {
        code: "trade_not_found",
        detail: "Trade not found.",
      })
    )
    await page.goto(`/trades/${SLUG}/add-item`)

    await expect(
      page.getByRole("heading", { name: "Trade not found" })
    ).toBeVisible()
    await expect(page.getByRole("link", { name: "Sign in" })).toBeVisible()
  })
})
