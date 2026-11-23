import AxeBuilder from "@axe-core/playwright"
import { test, expect } from "playwright/test"
import type { Route } from "playwright/test"

import {
  SIGNED_IN_TAG,
  signInAsTestUser,
  useSignedInSuite,
} from "./support/clerk-auth"

// Trade list behavior stays in the authenticated Trades page. The API answers
// from stubs; the seat rules and the executed refusal are backend test contracts.
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
      "access-control-allow-methods": "GET, POST, DELETE, OPTIONS",
    },
    body: JSON.stringify(body),
  })
}

interface TradeSummaryStub {
  id: string
  slug?: string
  status: string
  otherName?: string
  createdAt: string
  updatedAt: string
}

function tradeSummary(
  overrides: Partial<TradeSummaryStub> & { id: string }
): TradeSummaryStub {
  const now = Date.now()
  return {
    status: "open",
    createdAt: new Date(now - 3_600_000).toISOString(),
    updatedAt: new Date(now - 3_600_000).toISOString(),
    ...overrides,
  }
}

function hoursAgo(hours: number) {
  return new Date(Date.now() - hours * 3_600_000).toISOString()
}

test.describe("Trades list", { tag: SIGNED_IN_TAG }, () => {
  useSignedInSuite()

  test("opens a Trade from empty space beside its action", async ({ page }) => {
    await page.route(`${API}/trades`, (route) =>
      reply(route, 200, {
        data: [
          tradeSummary({
            id: "t-live",
            slug: "slug-live",
            status: "negotiating",
            otherName: "Ash",
          }),
        ],
      })
    )
    await signInAsTestUser(page)
    await page.goto("/trades")
    const row = page.getByRole("listitem").filter({ hasText: "Ash" })
    const box = await row.boundingBox()
    if (!box) throw new Error("Trade row is not visible")
    await page.mouse.click(box.x + 24, box.y + box.height - 22)
    await expect(page).toHaveURL(/\/trades\/slug-live$/)
  })

  test("lists each Trade's state and age, and cancels a live one", async ({
    page,
  }) => {
    const trades = [
      tradeSummary({
        id: "t-live",
        slug: "slug-live",
        status: "negotiating",
        otherName: "Ash",
        createdAt: hoursAgo(2),
      }),
      tradeSummary({
        id: "t-done",
        slug: "slug-done",
        status: "executed",
        otherName: "Misty",
        createdAt: hoursAgo(50),
      }),
      // A cancelled Trade's slug is dead, so the row is not a link.
      tradeSummary({
        id: "t-dead",
        status: "cancelled",
        otherName: "Gary",
        createdAt: hoursAgo(80),
      }),
    ]
    let cancelledSlug: string | null = null
    await page.route(`${API}/trades`, (route) =>
      reply(route, 200, { data: trades })
    )
    await page.route(`${API}/trades/slug-live/cancel`, (route) => {
      cancelledSlug = "slug-live"
      const row = trades.find((trade) => trade.id === "t-live")
      if (row) {
        row.status = "cancelled"
        delete row.slug
      }
      return reply(route, 204, {})
    })

    await signInAsTestUser(page)
    await page.goto("/trades")

    // State and age show for every row; only a live Trade offers cancel.
    await expect(page.getByRole("link", { name: "With Ash" })).toHaveAttribute(
      "href",
      /\/trades\/slug-live$/
    )
    await expect(
      page.getByText("Negotiating · started 2 hours ago")
    ).toBeVisible()
    await expect(page.getByText("Executed")).toBeVisible()
    await expect(page.getByText("Cancelled")).toBeVisible()
    await expect(page.getByRole("link", { name: "With Gary" })).toHaveCount(0)
    await expect(page.getByRole("button", { name: "Cancel" })).toHaveCount(1)

    await page.getByRole("button", { name: "Cancel" }).click()

    const ashRow = page.getByRole("listitem").filter({ hasText: "Ash" })
    await expect(ashRow).toContainText("Cancelled")
    expect(cancelledSlug).toBe("slug-live")
    // The dead slug leaves the row as plain text and no cancel remains.
    await expect(page.getByRole("link", { name: "With Ash" })).toHaveCount(0)
    await expect(page.getByRole("button", { name: "Cancel" })).toHaveCount(0)
  })

  test("discards an open Trade after confirmation and refreshes the list", async ({
    page,
  }) => {
    const trades = [
      tradeSummary({ id: "t-open", slug: "slug-open", createdAt: hoursAgo(1) }),
      tradeSummary({
        id: "t-live",
        slug: "slug-live",
        status: "negotiating",
        otherName: "Ash",
        createdAt: hoursAgo(2),
      }),
      tradeSummary({
        id: "t-done",
        slug: "slug-done",
        status: "executed",
        otherName: "Misty",
        createdAt: hoursAgo(50),
      }),
    ]
    let listCalls = 0
    let discardedSlug: string | null = null
    await page.route(`${API}/trades`, (route) => {
      listCalls++
      return reply(route, 200, { data: trades })
    })
    await page.route(`${API}/trades/slug-open/discard`, (route) => {
      discardedSlug = "slug-open"
      const index = trades.findIndex((trade) => trade.id === "t-open")
      trades.splice(index, 1)
      return reply(route, 204, {})
    })

    await signInAsTestUser(page)
    await page.goto("/trades")

    // Discard belongs to the open Trade only; Cancel belongs to the joined
    // one, and neither is offered for an executed Trade.
    const openRow = page
      .getByRole("listitem")
      .filter({ hasText: "Waiting for someone to join" })
    await expect(openRow.getByRole("button", { name: "Discard" })).toBeVisible()
    await expect(page.getByRole("button", { name: "Discard" })).toHaveCount(1)
    await expect(page.getByRole("button", { name: "Cancel" })).toHaveCount(1)

    await openRow.getByRole("button", { name: "Discard" }).click()
    const dialog = page.getByRole("alertdialog", {
      name: "Discard this Trade?",
    })
    await expect(dialog).toBeVisible()
    await expect(dialog).toContainText("permanently deletes")
    await expect(dialog.getByRole("button", { name: "Cancel" })).toBeFocused()

    // A stray Enter on the focused safe action closes without deleting.
    await page.keyboard.press("Enter")
    await expect(dialog).toBeHidden()
    expect(discardedSlug).toBeNull()

    await openRow.getByRole("button", { name: "Discard" }).click()
    await dialog.getByRole("button", { name: "Discard" }).click()
    await expect(dialog).toBeHidden()
    expect(discardedSlug).toBe("slug-open")
    await expect(
      page
        .getByRole("listitem")
        .filter({ hasText: "Waiting for someone to join" })
    ).toHaveCount(0)
    expect(listCalls).toBeGreaterThan(1)
  })

  test("reports a failed Discard and leaves the Trade in place", async ({
    page,
  }) => {
    await page.route(`${API}/trades`, (route) =>
      reply(route, 200, {
        data: [tradeSummary({ id: "t-open", slug: "slug-open" })],
      })
    )
    await page.route(`${API}/trades/slug-open/discard`, (route) =>
      reply(route, 503, { detail: "Service unavailable." })
    )

    await signInAsTestUser(page)
    await page.goto("/trades")
    await page.getByRole("button", { name: "Discard" }).click()
    const dialog = page.getByRole("alertdialog", {
      name: "Discard this Trade?",
    })
    await dialog.getByRole("button", { name: "Discard" }).click()

    await expect(dialog.getByRole("alert")).toContainText(
      "Service unavailable."
    )
    await dialog.getByRole("button", { name: "Cancel" }).click()
    await expect(
      page.getByRole("link", { name: "Waiting for someone to join" })
    ).toBeVisible()
  })

  test("Account has no Trade controls and Trades offers Retry after a list error", async ({
    page,
  }) => {
    let attempts = 0
    await page.route(`${API}/trades`, (route) => {
      attempts++
      return attempts === 1
        ? reply(route, 503, { detail: "Service unavailable." })
        : reply(route, 200, {
            data: [
              tradeSummary({
                id: "t-retry",
                slug: "slug-retry",
                otherName: "Ash",
              }),
            ],
          })
    })

    await signInAsTestUser(page)
    await page.goto("/account")
    await expect(page.getByRole("heading", { name: "Account" })).toBeVisible()
    await expect(
      page.getByRole("button", { name: "Start a Trade" })
    ).toHaveCount(0)
    await expect(page.getByText("Your Trades")).toHaveCount(0)
    await page.getByRole("link", { name: "Trades" }).click()
    await expect(page.getByRole("link", { name: "Trades" })).toHaveAttribute(
      "aria-current",
      "page"
    )

    await expect(page.getByRole("alert")).toContainText(
      "Could not load your Trades."
    )
    await page.getByRole("button", { name: "Retry" }).press("Enter")
    await expect(page.getByRole("link", { name: "With Ash" })).toBeVisible()
    expect(attempts).toBe(2)
    const { violations } = await new AxeBuilder({ page }).analyze()
    expect(violations).toEqual([])
  })

  test("a cancel refused as executed reports the refusal", async ({ page }) => {
    await page.route(`${API}/trades`, (route) =>
      reply(route, 200, {
        data: [
          tradeSummary({
            id: "t-race",
            slug: "slug-race",
            status: "locked",
            otherName: "Brock",
          }),
        ],
      })
    )
    await page.route(`${API}/trades/slug-race/cancel`, (route) =>
      reply(route, 409, {
        code: "trade_executed",
        detail: "This Trade already executed.",
      })
    )

    await signInAsTestUser(page)
    await page.goto("/trades")
    await page.getByRole("button", { name: "Cancel" }).click()

    await expect(page.getByText("This Trade already executed.")).toBeVisible()
    // The row stays, still cancellable-looking until the list refreshes.
    await expect(page.getByRole("link", { name: "With Brock" })).toBeVisible()
  })
})
