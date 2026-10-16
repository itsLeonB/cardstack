import { deflateSync, crc32 } from "node:zlib"
import { test, expect } from "playwright/test"
import type { Page, Route } from "playwright/test"

import {
  SIGNED_IN_TAG,
  signInAsTestUser,
  useSignedInSuite,
} from "./support/clerk-auth"

// Creating a binder and scanning one Page on the file-picker path. The photo
// is cut for real in Chromium; the stubbed match answers every cell with the
// same card, and Confirm saves the Page. Signed in through Clerk's real
// development instance; the stubs are what the real session token reads.
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

// A solid grey PNG, so the cut has a whole photo to flatten.
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
    "access-control-allow-methods": "GET, POST, PUT, PATCH, OPTIONS",
    "access-control-allow-headers": "authorization, content-type",
  }
}

interface StubBody {
  data: unknown
  meta?: { total: number; page: number; limit: number }
}

function json(route: Route, body: StubBody, status = 200) {
  return route.fulfill({
    status,
    contentType: "application/json",
    headers: cors(route),
    body: JSON.stringify(body),
  })
}

type SlotState = "empty" | "card" | "unidentified"
interface SavedSlot {
  state: SlotState
  cardId?: string
}

// The 2 by 2 Layout the test creates; a Page holds four Slots in grid order.
const ROWS = 2
const COLUMNS = 2

async function stubApi(page: Page) {
  let binder: { rows: number; columns: number; leafCount: number } | null = null
  let slots: { row: number; column: number; state: SlotState }[] = []
  const saves: { slots: SavedSlot[] }[] = []

  const layout = () =>
    binder && {
      ...binder,
      capacity: binder.rows * binder.columns * binder.leafCount * 2,
    }
  const cardCount = () => slots.filter((s) => s.state === "card").length
  const summary = () => ({
    id: "col-1",
    title: "Binder",
    description: "",
    cardCount: cardCount(),
    maxCardCount: 0,
    binder: layout(),
  })
  const binderPage = () => {
    const cardTotal = slots.filter((s) => s.state !== "empty").length
    return {
      page: 1,
      leaf: 1,
      side: "front",
      layout: layout(),
      cardTotal,
      unidentifiedCount: slots.filter((s) => s.state === "unidentified").length,
      slots: slots.map((slot) =>
        slot.state === "card" ? { ...slot, card } : slot
      ),
    }
  }

  // Page and API document loads share the origin, so only fetches are stubbed.
  const fetchOnly =
    (handler: (route: Route) => Promise<void>) => (route: Route) =>
      route.request().resourceType() === "fetch"
        ? handler(route)
        : route.fallback()

  await page.route(
    "**/collections",
    fetchOnly((route) => json(route, { data: [summary()] }))
  )
  await page.route(
    "**/collections/col-1",
    fetchOnly((route) => json(route, { data: summary() }))
  )
  await page.route("**/collections/col-1/binder", (route) => {
    if (route.request().method() === "OPTIONS")
      return route.fulfill({ status: 204, headers: cors(route) })
    const body = route.request().postDataJSON()
    binder = { rows: body.rows, columns: body.columns, leafCount: 1 }
    slots = Array.from({ length: ROWS * COLUMNS }, (_, index) => ({
      row: Math.floor(index / COLUMNS) + 1,
      column: (index % COLUMNS) + 1,
      state: "empty" as const,
    }))
    return json(route, { data: layout() }, 201)
  })
  await page.route("**/collections/col-1/binder/pages/1", (route) => {
    const request = route.request()
    if (request.method() === "OPTIONS")
      return route.fulfill({ status: 204, headers: cors(route) })
    if (request.method() === "PUT") {
      const body = request.postDataJSON()
      saves.push(body)
      slots = body.slots.map((slot: SavedSlot, index: number) => ({
        row: Math.floor(index / COLUMNS) + 1,
        column: (index % COLUMNS) + 1,
        state: slot.state,
      }))
      return json(route, { data: { slots } })
    }
    return json(route, { data: binderPage() })
  })
  await page.route("**/collections/col-1/entries**", (route) =>
    route.request().method() === "OPTIONS"
      ? route.fulfill({ status: 204, headers: cors(route) })
      : json(route, { data: [], meta: { total: 0, page: 1, limit: 24 } })
  )
  await page.route("**/collections/col-1/facets*", (route) =>
    json(route, {
      data: { categories: [], expansionSets: [], rarities: [], tags: [] },
    })
  )
  await page.route("**/scan/match", (route) => {
    if (route.request().method() === "OPTIONS")
      return route.fulfill({ status: 204, headers: cors(route) })
    return json(route, {
      data: { confident: true, candidates: [{ card, score: 0.99 }] },
    })
  })
  return { saves }
}

test.describe("Page scan", { tag: SIGNED_IN_TAG }, () => {
  useSignedInSuite()

  test("creates a binder, scans a Page, reviews the cells and confirms", async ({
    page,
  }) => {
    const api = await stubApi(page)
    await signInAsTestUser(page)

    // Reached by client navigation: a direct load runs the route loader on the
    // dev server, where these browser-level stubs don't apply.
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

    await page.getByLabel("Layout").selectOption("2x2")
    await page.getByRole("button", { name: "Create binder" }).click()
    await expect(page.getByText("Page 1 of 2")).toBeVisible()

    await page.getByRole("button", { name: "Scan this Page" }).click()
    await page.locator('input[type="file"]').setInputFiles({
      name: "page.png",
      mimeType: "image/png",
      buffer: png(300, 420),
    })
    await page.getByRole("button", { name: "Cut and match" }).click()

    // Confirm stays disabled until all four stubbed matches have answered.
    const confirm = page.getByRole("button", { name: "Confirm" })
    await expect(confirm).toBeEnabled()
    await expect(
      page.getByRole("group", { name: "Row 2, column 2" }).getByText("Card 7")
    ).toBeVisible()
    await confirm.click()

    // The Page reloads on the binder view with the four Slots filled.
    await expect(
      page.getByText("Cards and Unidentified: 4 of 8 slots")
    ).toBeVisible()
    expect(api.saves).toEqual([
      {
        slots: Array.from({ length: 4 }, () => ({
          state: "card",
          cardId: "card-7",
        })),
      },
    ])
  })
})
