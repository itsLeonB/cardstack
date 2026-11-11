import { test, expect } from "playwright/test"
import type { Route } from "playwright/test"

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
    await page.goto("/account")
    await page.getByRole("button", { name: "Start a Trade" }).click()

    await expect(
      page.getByRole("heading", { name: "Trade", level: 1 })
    ).toBeVisible()
    await expect(page.getByText("Ash Ketchum")).toBeVisible()
    await expect(page.getByText("Open")).toBeVisible()
    await expect(page.getByText("Waiting for someone to join")).toBeVisible()
    // The owner gets the link to send, and Trade pages stay unindexed.
    const origin = new URL(page.url()).origin
    await expect(page.getByText(`${origin}/trades/${SLUG}`)).toBeVisible()
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
// seat; a join that loses the race is told the Trade is full.
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
      page.getByRole("button", { name: "Join this Trade" })
    ).toHaveCount(0)
  })

  test("a join that loses the seat is told the Trade is full", async ({
    page,
  }) => {
    // The read answers the post-join truth once someone else holds the seat.
    let seatTaken = false
    await page.route(`${API}/trades/${SLUG}`, (route) =>
      reply(route, 200, {
        data: tradeView(
          seatTaken
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
      seatTaken = true
      return reply(route, 409, {
        code: "trade_seat_taken",
        detail: "Seat taken.",
      })
    })

    await signInAsTestUser(page)
    await page.goto(`/trades/${SLUG}`)
    await page.getByRole("button", { name: "Join this Trade" }).click()

    await expect(page.getByText("This Trade is already full.")).toBeVisible()
    // The page refetches to the truth: the seat is someone else's now.
    await expect(
      page.getByRole("button", { name: "Join this Trade" })
    ).toHaveCount(0)
    await expect(page.getByText("Negotiating")).toBeVisible()
    await expect(page.getByText("Ash Ketchum").nth(1)).toBeVisible()
  })
})
