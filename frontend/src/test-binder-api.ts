import { vi } from "vitest"

// An in-memory binder behind a faked `fetch`, for tests of the binder views.
// The generated client, the query cache and the invalidation under test are
// the app's own; only the network answers. Undo with `vi.unstubAllGlobals()`.

export interface FakeCard {
  name: string
  localId: string
  imageUrl: string
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

function respond(
  b: FakeBinder,
  method: string,
  path: string,
  body: RequestBody | undefined
): Response {
  const page = path.match(/\/binder\/pages\/(\d+)$/)
  if (method === "GET" && page) {
    const n = Number(page[1])
    if (n < 1 || n > b.leafCount * 2) {
      return json(404, { detail: "this binder has no Page with that number" })
    }
    return json(200, { data: pageBody(b, n) })
  }
  if (method === "POST" && path.endsWith("/binder/leaves")) {
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
  if (method === "GET" && path.endsWith("/collections/col-1")) {
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
  if (method === "GET" && path.endsWith("/collections/col-1/entries")) {
    return json(200, {
      data: [],
      meta: { total: 0, page: 1, limit: 100 },
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
