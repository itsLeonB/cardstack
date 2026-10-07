import { deflateSync, crc32 } from "node:zlib"
import { test, expect } from "playwright/test"
import type { Page, Route } from "playwright/test"

import {
  SIGNED_IN_TAG,
  signInAsTestUser,
  useSignedInSuite,
} from "./support/clerk-auth"

// The scan, review and add flow on the file-picker path (no camera): a stubbed
// match answers the photo, the stubbed Collection holds 3 copies of the card,
// and its page must show 4 after the add. Needs VITE_SCAN_ENABLED=true in the
// app's build (playwright.config.ts sets it for the dev server it starts).
// Signed in through Clerk's real development instance; the stubs are what the
// real session token reads.
const card = {
  id: "card-7",
  name: "Card 7",
  localId: "7",
  category: "Pokémon",
  tags: [],
  illustrator: "someone",
  imageUrl: "",
  expansionSet: { id: "set-1", code: "TST", name: "Test Set", imageUrl: "" },
  rarity: { id: "rarity-1", code: "C", name: "Common" },
}

// A solid grey PNG, tall enough for the scan's card crop.
function png(width: number, height: number) {
  const chunk = (type: string, data: Buffer) => {
    const body = Buffer.concat([Buffer.from(type), data])
    const length = Buffer.alloc(4)
    length.writeUInt32BE(data.length)
    const crc = Buffer.alloc(4)
    crc.writeUInt32BE(crc32(body))
    return Buffer.concat([length, body, crc])
  }
  const header = Buffer.alloc(13)
  header.writeUInt32BE(width, 0)
  header.writeUInt32BE(height, 4)
  header[8] = 8 // bit depth
  header[9] = 2 // RGB
  const row = Buffer.concat([Buffer.from([0]), Buffer.alloc(width * 3, 128)])
  return Buffer.concat([
    Buffer.from([0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a]),
    chunk("IHDR", header),
    chunk("IDAT", deflateSync(Buffer.concat(Array(height).fill(row)))),
    chunk("IEND", Buffer.alloc(0)),
  ])
}

// Cross-origin calls need the caller's exact origin echoed back.
function cors(route: Route) {
  return {
    "access-control-allow-origin": route.request().headers()["origin"] ?? "",
    "access-control-allow-credentials": "true",
    "access-control-allow-methods": "GET, POST, PATCH, OPTIONS",
    "access-control-allow-headers": "authorization, content-type",
  }
}

interface StubBody {
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

async function stubApi(page: Page) {
  let held = 3
  const bulkBodies: { items: { cardId: string; quantity: number }[] }[] = []
  const summary = () => ({
    id: "col-1",
    title: "Binder",
    description: "",
    cardCount: held,
    maxCardCount: 0,
  })
  const fetchOnly =
    (handler: (route: Route) => Promise<void>) => (route: Route) =>
      route.request().resourceType() === "fetch"
        ? handler(route)
        : route.fallback()

  await page.route("**/catalog/series", (route) =>
    json(route, { data: { series: [], ungroupedExpansionSets: [] } })
  )
  await page.route(
    "**/collections",
    fetchOnly((route) => json(route, { data: [summary()] }))
  )
  await page.route(
    "**/collections/col-1",
    fetchOnly((route) => json(route, { data: summary() }))
  )
  await page.route("**/collections/col-1/facets*", (route) =>
    json(route, {
      data: { categories: [], expansionSets: [], rarities: [], tags: [] },
    })
  )
  await page.route("**/collections/col-1/entries**", (route) => {
    const request = route.request()
    if (request.method() === "OPTIONS")
      return route.fulfill({ status: 204, headers: cors(route) })
    if (request.method() === "PATCH") {
      const body = request.postDataJSON()
      bulkBodies.push(body)
      held = body.items[0].quantity
      return json(route, {
        data: [{ cardId: "card-7", quantity: held, status: "applied" }],
      })
    }
    return json(route, {
      data: [{ card, quantity: held }],
      meta: { total: 1, page: 1, limit: 24 },
    })
  })
  await page.route("**/catalog/cards?*", (route) =>
    json(route, {
      data: [card],
      meta: { total: 1, page: 1, limit: 100 },
    })
  )
  await page.route("**/scan/match", (route) => {
    if (route.request().method() === "OPTIONS")
      return route.fulfill({ status: 204, headers: cors(route) })
    return json(route, {
      data: { confident: true, candidates: [{ card, score: 0.99 }] },
    })
  })
  return { bulkBodies }
}

test.describe("Card scanning", { tag: SIGNED_IN_TAG }, () => {
  useSignedInSuite()

  test("scans a photo, reviews the draft and adds it to the Collection", async ({
    page,
  }) => {
    const api = await stubApi(page)
    await signInAsTestUser(page)

    // Reached by client navigation: a direct load runs the route loader on the
    // dev server, where these browser-level stubs don't apply. The dev server
    // can hydrate after the first click lands, so retry until it navigates.
    await page.goto("/")
    await expect(async () => {
      await page
        .getByRole("banner")
        .getByRole("link", { name: "Collections" })
        .click()
      await expect(page.getByRole("link", { name: "Binder" })).toBeVisible({
        timeout: 1000,
      })
    }).toPass()
    await page.getByRole("link", { name: "Binder" }).click()
    await page.getByRole("link", { name: "Scan cards" }).click()

    await page.locator('input[type="file"]').setInputFiles({
      name: "card.png",
      mimeType: "image/png",
      buffer: png(300, 420),
    })
    await expect(page.getByText("Added Card 7")).toBeAttached()
    await page.getByRole("button", { name: "Review and add" }).click()
    await expect(page.getByText("Holds 3")).toBeVisible()
    await page.getByRole("button", { name: "Add to Collection" }).click()

    await expect(
      page.getByRole("heading", { name: "Binder", level: 1 })
    ).toBeVisible()
    await expect(
      page.getByLabel("Quantity of Card 7", { exact: true })
    ).toHaveValue("4")
    expect(api.bulkBodies).toEqual([
      { items: [{ cardId: "card-7", quantity: 4 }] },
    ])
  })
})
