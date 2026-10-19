import { vi } from "vitest"
import type { CardSummary, InventoryItem } from "@/generated/models"
import type { SlotRef } from "@/lib/binder"

// An in-memory binder behind a faked `fetch`, for tests of the binder views.
// The generated client, the query cache and the invalidation under test are
// the app's own; only the network answers. Undo with `vi.unstubAllGlobals()`.

export type FakeCard = CardSummary

/** A catalog Card for the fake: the search returns it and a Slot write may place it. */
export function catalogCard(
  id: string,
  name: string,
  localId: string
): CardSummary {
  return {
    id,
    name,
    localId,
    category: "Pokémon",
    expansionSet: {
      id: "set-1",
      code: "BS",
      name: "Base Set",
      imageUrl: "https://img.example/set.png",
    },
    illustrator: "",
    imageUrl: `https://img.example/${id}.jpg`,
    rarity: { id: "rare", code: "R", name: "Rare" },
    tags: [],
  }
}

export interface FakeSlot {
  state: "card" | "unidentified"
  card?: FakeCard
}

/** The JSON bodies the binder endpoints receive. */
export interface RequestBody {
  rows?: number
  columns?: number
  leafCount?: number
  leaf?: number
  placement?: string
  cardId?: string
  from?: SlotRef
  to?: SlotRef
}

/** What the fake answers with: an envelope, an error, or nothing (204). */
type ResponseBody =
  { data: object; meta?: object } | { code?: string; detail: string } | null

export interface FakeBinder {
  exists: boolean
  rows: number
  columns: number
  leafCount: number
  cardCount: number
  /** Filled Slots by Page number, keyed "row,column". */
  filled: Record<number, Record<string, FakeSlot>>
  /** Cards the catalog search returns, and the only ones a Slot write accepts. */
  catalog: FakeCard[]
  /** When set, every Slot write is answered with this error and changes nothing. */
  writeError?: { status: number; detail: string }
  /** What the Collection's entries list answers with. */
  entries: InventoryItem[]
  requests: { method: string; path: string; body?: RequestBody }[]
}

export function fakeBinder(overrides: Partial<FakeBinder> = {}): FakeBinder {
  return {
    exists: true,
    rows: 2,
    columns: 2,
    leafCount: 10,
    cardCount: 0,
    filled: {},
    catalog: [],
    entries: [],
    requests: [],
    ...overrides,
  }
}

function layoutOf(b: FakeBinder) {
  return {
    rows: b.rows,
    columns: b.columns,
    leafCount: b.leafCount,
    capacity: b.leafCount * 2 * b.rows * b.columns,
  }
}

function allFilled(b: FakeBinder) {
  return Object.values(b.filled).flatMap((page) => Object.values(page))
}

function pageBody(b: FakeBinder, page: number) {
  const slots = []
  for (let row = 1; row <= b.rows; row++) {
    for (let column = 1; column <= b.columns; column++) {
      const slot = b.filled[page]?.[`${row},${column}`]
      slots.push({
        row,
        column,
        state: slot?.state ?? "empty",
        card: slot?.card ?? null,
      })
    }
  }
  const filled = allFilled(b)
  return {
    page,
    leaf: Math.ceil(page / 2),
    side: page % 2 === 1 ? "front" : "back",
    layout: layoutOf(b),
    cardTotal: filled.length,
    unidentifiedCount: filled.filter((s) => s.state === "unidentified").length,
    slots,
  }
}

function json(status: number, body: ResponseBody) {
  return new Response(body === null ? null : JSON.stringify(body), {
    status,
    headers: { "Content-Type": "application/json" },
  })
}

function writeSlot(
  b: FakeBinder,
  method: string,
  [, pageText, rowText, columnText, unidentified]: RegExpMatchArray,
  body: RequestBody | undefined
): Response {
  if (b.writeError)
    return json(b.writeError.status, { detail: b.writeError.detail })
  const page = Number(pageText)
  const row = Number(rowText)
  const column = Number(columnText)
  if (page < 1 || page > b.leafCount * 2) {
    return json(404, { detail: "this binder has no Page with that number" })
  }
  if (row < 1 || row > b.rows || column < 1 || column > b.columns) {
    return json(400, { detail: "this Slot is outside the Layout" })
  }
  const slots = (b.filled[page] ??= {})
  const key = `${row},${column}`
  if (method === "DELETE") {
    delete slots[key]
  } else if (unidentified) {
    slots[key] = { state: "unidentified" }
  } else {
    const card = b.catalog.find((c) => c.id === body?.cardId)
    if (!card) return json(404, { detail: "no Card has that id" })
    slots[key] = { state: "card", card }
  }
  return json(200, { data: pageBody(b, page) })
}

function moveSlot(b: FakeBinder, body: RequestBody | undefined): Response {
  if (b.writeError)
    return json(b.writeError.status, { detail: b.writeError.detail })
  const from = body?.from
  const to = body?.to
  const source = from && b.filled[from.page]?.[`${from.row},${from.column}`]
  if (!from || !to || !source)
    return json(400, { detail: "this Slot holds no card to move" })
  const fromKey = `${from.row},${from.column}`
  const toKey = `${to.row},${to.column}`
  const fromSlots = (b.filled[from.page] ??= {})
  const toSlots = (b.filled[to.page] ??= {})
  // An occupied target swaps: its card takes the source Slot.
  const target = toSlots[toKey]
  if (target) fromSlots[fromKey] = target
  else delete fromSlots[fromKey]
  toSlots[toKey] = source
  return json(200, {
    data: { from: pageBody(b, from.page), to: pageBody(b, to.page) },
  })
}

// A Leaf change renumbers the Pages from `from` on, so the Slots and the
// positions the entries list shows move with them.
function shiftPages(b: FakeBinder, from: number, delta: number) {
  const moved: FakeBinder["filled"] = {}
  for (const [page, slots] of Object.entries(b.filled)) {
    const n = Number(page)
    moved[n >= from ? n + delta : n] = slots
  }
  b.filled = moved
  for (const item of b.entries) {
    item.positions = item.positions?.map((p) =>
      p.page >= from ? { ...p, page: p.page + delta } : p
    )
  }
}

function respond(
  b: FakeBinder,
  method: string,
  path: string,
  body: RequestBody | undefined
): Response {
  const page = path.match(/\/binder\/pages\/(\d+)$/)
  // The fake accepts the replace without storing it, and reports no Slot results.
  if (method === "PUT" && page) {
    return json(200, { data: { slots: null } })
  }
  if (method === "GET" && page) {
    const n = Number(page[1])
    if (n < 1 || n > b.leafCount * 2) {
      return json(404, { detail: "this binder has no Page with that number" })
    }
    return json(200, { data: pageBody(b, n) })
  }
  const slot = path.match(
    /\/binder\/pages\/(\d+)\/slots\/(\d+)\/(\d+)(\/unidentified)?$/
  )
  if (slot && (method === "PUT" || method === "DELETE")) {
    return writeSlot(b, method, slot, body)
  }
  if (method === "POST" && path.endsWith("/binder/slots/move")) {
    return moveSlot(b, body)
  }
  if (method === "GET" && path.endsWith("/catalog/cards")) {
    return json(200, {
      data: b.catalog,
      meta: { total: b.catalog.length, page: 1, limit: 60 },
    })
  }
  if (method === "POST" && path.endsWith("/binder/leaves")) {
    const leaf = body?.leaf ?? 1
    const at = body?.placement === "after" ? leaf + 1 : leaf
    shiftPages(b, 2 * at - 1, 2)
    b.leafCount += 1
    return json(201, { data: layoutOf(b) })
  }
  const leaf = path.match(/\/binder\/leaves\/(\d+)$/)
  if (method === "DELETE" && leaf) {
    const n = Number(leaf[1])
    for (const side of [2 * n - 1, 2 * n]) {
      if (Object.keys(b.filled[side] ?? {}).length > 0) {
        return json(409, {
          code: "binder_leaf_in_use",
          detail: `page ${side} holds a card; clear it before removing its Leaf`,
        })
      }
    }
    shiftPages(b, 2 * n + 1, -2)
    b.leafCount -= 1
    return json(200, { data: layoutOf(b) })
  }
  if (method === "PUT" && path.endsWith("/binder/layout")) {
    b.rows = body?.rows ?? b.rows
    b.columns = body?.columns ?? b.columns
    return json(200, { data: layoutOf(b) })
  }
  if (method === "POST" && path.endsWith("/binder")) {
    b.exists = true
    b.rows = body?.rows ?? b.rows
    b.columns = body?.columns ?? b.columns
    b.leafCount = body?.leafCount ?? b.leafCount
    return json(201, { data: layoutOf(b) })
  }
  if (method === "DELETE" && path.endsWith("/binder")) {
    b.exists = false
    return json(204, null)
  }
  if (method === "GET" && /\/collections\/[^/]+$/.test(path)) {
    return json(200, {
      data: {
        id: "col-1",
        title: "Binder",
        description: "",
        maxCardCount: 0,
        cardCount: b.cardCount,
        binder: b.exists ? layoutOf(b) : null,
      },
    })
  }
  if (method === "GET" && /\/collections\/[^/]+\/entries$/.test(path)) {
    return json(200, {
      data: b.entries,
      meta: { total: b.entries.length, page: 1, limit: 100 },
    })
  }
  return json(500, { detail: `unexpected ${method} ${path}` })
}

/** Routes every request through the fake binder and records it. */
export function stubBinderApi(b: FakeBinder) {
  const fetchMock = vi.fn(
    async (input: RequestInfo | URL, init?: RequestInit) => {
      const { pathname } = new URL(String(input))
      const method = init?.method ?? "GET"
      const body: RequestBody | undefined = init?.body
        ? JSON.parse(String(init.body))
        : undefined
      b.requests.push({ method, path: pathname, body })
      return respond(b, method, pathname, body)
    }
  )
  vi.stubGlobal("fetch", fetchMock)
  return fetchMock
}
