import { test, expect } from "playwright/test"
import type { Route } from "playwright/test"

import {
  SIGNED_IN_TAG,
  signInAsTestUser,
  useSignedInSuite,
} from "./support/clerk-auth"

// Issue 07: the account area lists the caller's Trades with state and age,
// and a live Trade can be cancelled from there. The API answers from stubs;
// the seat rules and the executed refusal are the routes tests' job (backend).
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
    await page.goto("/account")

    // State and age show for every row; only a live Trade offers cancel.
    await expect(page.getByRole("link", { name: "Ash" })).toHaveAttribute(
      "href",
      /\/trades\/slug-live$/
    )
    await expect(
      page.getByText("Negotiating · started 2 hours ago")
    ).toBeVisible()
    await expect(page.getByText("Executed")).toBeVisible()
    await expect(page.getByText("Cancelled")).toBeVisible()
    await expect(page.getByRole("link", { name: "Gary" })).toHaveCount(0)
    await expect(page.getByRole("button", { name: "Cancel" })).toHaveCount(1)

    await page.getByRole("button", { name: "Cancel" }).click()

    const ashRow = page.getByRole("listitem").filter({ hasText: "Ash" })
    await expect(ashRow).toContainText("Cancelled")
    expect(cancelledSlug).toBe("slug-live")
    // The dead slug leaves the row as plain text and no cancel remains.
    await expect(page.getByRole("link", { name: "Ash" })).toHaveCount(0)
    await expect(page.getByRole("button", { name: "Cancel" })).toHaveCount(0)
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
    await page.goto("/account")
    await page.getByRole("button", { name: "Cancel" }).click()

    await expect(page.getByText("This Trade already executed.")).toBeVisible()
    // The row stays, still cancellable-looking until the list refreshes.
    await expect(page.getByRole("link", { name: "Brock" })).toBeVisible()
  })
})
