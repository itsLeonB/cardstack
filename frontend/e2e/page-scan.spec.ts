import { deflateSync, crc32 } from "node:zlib"
import { test, expect } from "playwright/test"
import type { Page, Route } from "playwright/test"

import {
  SIGNED_IN_TAG,
  signInAsTestUser,
  useSignedInSuite,
} from "./support/clerk-auth"

// Creating a binder and scanning one Page on the file-picker path. The photo
// has a colour per quadrant, so the real cut in Chromium can be checked for
// geometry and order: each uploaded cell must show its own quadrant. The match
// answers are held until the test releases them, which exercises the queue and
// the Confirm wait. Signed in through Clerk's real development instance; the
// stubs are what the real session token reads.
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

// Top left, top right, bottom left, bottom right: the cells of the 2 by 2 Page
// read in this order, so each colour marks one cell.
const QUADRANTS = [
  [255, 0, 0],
  [0, 255, 0],
  [0, 0, 255],
  [255, 255, 0],
]

function encodePng(width: number, height: number, raw: Buffer) {
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
  return Buffer.concat([
    Buffer.from([0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a]),
    chunk("IHDR", header),
    chunk("IDAT", deflateSync(raw)),
    chunk("IEND", Buffer.alloc(0)),
  ])
}

// A photo whose top left quarter is red, top right green, bottom left blue and
// bottom right yellow. Each row starts with PNG's filter byte.
function quadrantPng(width: number, height: number) {
  const rows = Array.from({ length: height }, (_, y) => {
    const row = Buffer.alloc(width * 3 + 1)
    for (let x = 0; x < width; x++) {
      const quadrant = (y < height / 2 ? 0 : 2) + (x < width / 2 ? 0 : 1)
      row.set(QUADRANTS[quadrant], 1 + x * 3)
    }
    return row
  })
  return encodePng(width, height, Buffer.concat(rows))
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
  // Each match request, in the order it arrived, and the answers still held.
  const matchBodies: Buffer[] = []
  const matchContentTypes: string[] = []
  const held: Route[] = []
  let peak = 0

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
    const request = route.request()
    matchBodies.push(request.postDataBuffer() ?? Buffer.alloc(0))
    matchContentTypes.push(request.headers()["content-type"] ?? "")
    held.push(route)
    peak = Math.max(peak, held.length)
  })

  // Answers the oldest held matches with the one Card, so every cell matches it.
  function release(count: number) {
    for (const route of held.splice(0, count)) {
      void json(route, {
        data: { confident: true, candidates: [{ card, score: 0.99 }] },
      })
    }
  }

  return {
    saves,
    matchBodies,
    matchContentTypes,
    held,
    release,
    peak: () => peak,
  }
}

// The colour at the middle of each uploaded cell, decoded in the page.
async function cellColours(page: Page, jpegs: Buffer[]) {
  return page.evaluate(
    async (encoded: string[]) => {
      const colours: number[][] = []
      for (const base64 of encoded) {
        const bytes = Uint8Array.from(atob(base64), (c) => c.charCodeAt(0))
        const bitmap = await createImageBitmap(
          new Blob([bytes], { type: "image/jpeg" })
        )
        const canvas = document.createElement("canvas")
        canvas.width = bitmap.width
        canvas.height = bitmap.height
        const context = canvas.getContext("2d")
        if (!context) throw new Error("No 2D canvas in this browser")
        context.drawImage(bitmap, 0, 0)
        const x = Math.floor(bitmap.width / 2) - 5
        const y = Math.floor(bitmap.height / 2) - 5
        const { data } = context.getImageData(x, y, 10, 10)
        const sum = [0, 0, 0]
        for (let i = 0; i < data.length; i += 4) {
          sum[0] += data[i]
          sum[1] += data[i + 1]
          sum[2] += data[i + 2]
        }
        colours.push(sum.map((channel) => channel / (data.length / 4)))
      }
      return colours
    },
    jpegs.map((jpeg) => jpeg.toString("base64"))
  )
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
      buffer: quadrantPng(800, 1000),
    })
    await page.getByRole("button", { name: "Cut and match" }).click()

    // Three matches run at once; the fourth cell waits for one of them.
    await expect.poll(() => api.held.length).toBe(3)
    const confirm = page.getByRole("button", { name: "Confirm" })
    await expect(confirm).toBeDisabled()

    api.release(1)
    await expect.poll(() => api.matchBodies.length).toBe(4)
    expect(api.peak()).toBe(3)
    await expect(confirm).toBeDisabled()

    api.release(3)
    await expect(confirm).toBeEnabled()
    await expect(
      page.getByRole("group", { name: "Row 2, column 2" }).getByText("Card 7")
    ).toBeVisible()

    // The real cut sent four JPEGs, each showing its own quadrant, in grid order.
    expect(api.matchContentTypes).toEqual(Array(4).fill("image/jpeg"))
    const colours = await cellColours(page, api.matchBodies)
    colours.forEach((colour, index) =>
      QUADRANTS[index].forEach((channel, i) =>
        expect(Math.abs(colour[i] - channel)).toBeLessThan(60)
      )
    )

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
