import { afterEach, describe, expect, it, vi } from "vitest"
import { cleanup, fireEvent, render, screen } from "@testing-library/react"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { BinderView, CreateBinderPanel } from "./binder-view"

// The generated client runs over a faked `fetch` that answers the binder
// endpoints from a small in-memory binder, so the request, cache and
// invalidation code under test is the app's own.

interface FakeSlot {
  state: "card" | "unidentified"
  card?: { name: string; localId: string; imageUrl: string }
}

/** The JSON bodies the binder endpoints receive. */
interface RequestBody {
  rows?: number
  columns?: number
  leafCount?: number
  leaf?: number
  placement?: string
}

interface FakeBinder {
  exists: boolean
  rows: number
  columns: number
  leafCount: number
  /** Filled Slots by Page number, keyed "row,column". */
  filled: Record<number, Record<string, FakeSlot>>
  requests: { method: string; path: string; body?: RequestBody }[]
}

function fakeBinder(overrides: Partial<FakeBinder> = {}): FakeBinder {
  return {
    exists: true,
    rows: 2,
    columns: 2,
    leafCount: 10,
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

/** What the fake answers with: an envelope, an error, or nothing (204). */
type ResponseBody = { data: object } | { code?: string; detail: string } | null

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
  return json(500, { detail: `unexpected ${method} ${path}` })
}

function stubApi(b: FakeBinder) {
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

function renderWithClient(ui: React.ReactElement) {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  })
  return render(
    <QueryClientProvider client={queryClient}>{ui}</QueryClientProvider>
  )
}

// The swipe handlers read `touches` and `changedTouches` off the event.
function swipe(target: HTMLElement, fromX: number, toX: number) {
  const start = new Event("touchstart", { bubbles: true })
  Object.defineProperty(start, "touches", { value: [{ clientX: fromX }] })
  fireEvent(target, start)
  const end = new Event("touchend", { bubbles: true })
  Object.defineProperty(end, "changedTouches", { value: [{ clientX: toX }] })
  fireEvent(target, end)
}

afterEach(() => {
  cleanup()
  vi.unstubAllGlobals()
})

const pikachu = {
  name: "Pikachu",
  localId: "025",
  imageUrl: "https://img.example/pikachu.jpg",
}

describe("BinderView", () => {
  it("shows the Page's Slots by state and the card total against capacity", async () => {
    stubApi(
      fakeBinder({
        filled: {
          1: {
            "1,1": { state: "card", card: pikachu },
            "1,2": { state: "unidentified" },
          },
        },
      })
    )
    renderWithClient(<BinderView collectionId="col-1" />)

    expect(
      await screen.findByRole("group", { name: "Row 1, column 1: Pikachu" })
    ).toBeTruthy()
    expect(
      screen.getByRole("group", { name: "Row 1, column 2: Unidentified" })
    ).toBeTruthy()
    expect(
      screen.getByRole("group", { name: "Row 2, column 1: empty" })
    ).toBeTruthy()
    expect(screen.getByText("Page 1 of 20")).toBeTruthy()
    expect(
      screen.getByText("Cards and Unidentified: 2 of 80 slots")
    ).toBeTruthy()
    expect(screen.getByText("Unidentified: 1")).toBeTruthy()
  })

  it("swipes, steps and jumps between Pages", async () => {
    const binder = fakeBinder()
    stubApi(binder)
    renderWithClient(<BinderView collectionId="col-1" />)
    await screen.findByText("Page 1 of 20")

    const pageRegion = screen.getByRole("region", { name: "Binder page" })
    swipe(pageRegion, 200, 100)
    expect(await screen.findByText("Page 2 of 20")).toBeTruthy()

    fireEvent.click(screen.getByRole("button", { name: "Next page" }))
    expect(await screen.findByText("Page 3 of 20")).toBeTruthy()

    fireEvent.click(screen.getByRole("button", { name: "Previous page" }))
    expect(await screen.findByText("Page 2 of 20")).toBeTruthy()

    fireEvent.change(screen.getByRole("spinbutton", { name: "Page number" }), {
      target: { value: "7" },
    })
    fireEvent.click(screen.getByRole("button", { name: "Go to page" }))
    expect(await screen.findByText("Page 7 of 20")).toBeTruthy()
    expect(binder.requests.map((r) => r.path)).toContain(
      "/collections/col-1/binder/pages/7"
    )
  })

  it("adds a Leaf after the Leaf being viewed and turns to its front Page", async () => {
    const binder = fakeBinder()
    stubApi(binder)
    renderWithClient(<BinderView collectionId="col-1" />)
    await screen.findByText("Page 1 of 20")

    fireEvent.change(screen.getByRole("spinbutton", { name: "Page number" }), {
      target: { value: "3" },
    })
    fireEvent.click(screen.getByRole("button", { name: "Go to page" }))
    await screen.findByText("Page 3 of 20")

    fireEvent.click(screen.getByRole("button", { name: "Add Leaf after" }))

    expect(await screen.findByText("Page 5 of 22")).toBeTruthy()
    expect(binder.requests).toContainEqual(
      expect.objectContaining({
        method: "POST",
        path: "/collections/col-1/binder/leaves",
        body: { leaf: 2, placement: "after" },
      })
    )
  })

  it("shows the refusal naming the Page when the Leaf holds a card", async () => {
    const binder = fakeBinder({
      filled: { 4: { "1,1": { state: "unidentified" } } },
    })
    stubApi(binder)
    renderWithClient(<BinderView collectionId="col-1" />)
    await screen.findByText("Page 1 of 20")

    fireEvent.change(screen.getByRole("spinbutton", { name: "Page number" }), {
      target: { value: "3" },
    })
    fireEvent.click(screen.getByRole("button", { name: "Go to page" }))
    await screen.findByText("Page 3 of 20")

    fireEvent.click(screen.getByRole("button", { name: "Remove Leaf" }))

    expect(await screen.findByRole("alert")).toBeTruthy()
    expect(
      screen.getByText("page 4 holds a card; clear it before removing its Leaf")
    ).toBeTruthy()
    expect(binder.leafCount).toBe(10)
  })

  it("offers the Layout edit and removal only while every Slot is empty", async () => {
    stubApi(fakeBinder())
    const { unmount } = renderWithClient(<BinderView collectionId="col-1" />)
    await screen.findByText("Page 1 of 20")
    expect(screen.getByRole("button", { name: "Change layout" })).toBeTruthy()
    expect(screen.getByRole("button", { name: "Remove binder" })).toBeTruthy()
    unmount()

    stubApi(fakeBinder({ filled: { 1: { "1,1": { state: "unidentified" } } } }))
    renderWithClient(<BinderView collectionId="col-1" />)
    await screen.findByText("Page 1 of 20")
    expect(screen.queryByRole("button", { name: "Change layout" })).toBeNull()
    expect(screen.queryByRole("button", { name: "Remove binder" })).toBeNull()
  })

  it("removes the binder's Layout after confirming", async () => {
    const binder = fakeBinder()
    stubApi(binder)
    renderWithClient(<BinderView collectionId="col-1" />)
    await screen.findByText("Page 1 of 20")

    fireEvent.click(screen.getByRole("button", { name: "Remove binder" }))
    fireEvent.click(
      screen.getByRole("button", { name: "Confirm remove binder" })
    )

    await vi.waitFor(() =>
      expect(binder.requests).toContainEqual(
        expect.objectContaining({
          method: "DELETE",
          path: "/collections/col-1/binder",
        })
      )
    )
  })
})

describe("CreateBinderPanel", () => {
  it("creates a binder with the chosen preset and the default 10 Leaves", async () => {
    const binder = fakeBinder({ exists: false })
    stubApi(binder)
    renderWithClient(<CreateBinderPanel collectionId="col-1" />)

    fireEvent.change(screen.getByRole("combobox", { name: "Layout" }), {
      target: { value: "3x4" },
    })
    fireEvent.click(screen.getByRole("button", { name: "Create binder" }))

    await vi.waitFor(() =>
      expect(binder.requests).toContainEqual(
        expect.objectContaining({
          method: "POST",
          path: "/collections/col-1/binder",
          body: { rows: 3, columns: 4, leafCount: 10 },
        })
      )
    )
  })

  it("refuses a Leaf count above 100 without sending a request", () => {
    const binder = fakeBinder({ exists: false })
    stubApi(binder)
    renderWithClient(<CreateBinderPanel collectionId="col-1" />)

    fireEvent.change(screen.getByRole("spinbutton", { name: "Leaves" }), {
      target: { value: "101" },
    })
    fireEvent.click(screen.getByRole("button", { name: "Create binder" }))

    expect(
      screen.getByText("A binder holds from 1 to 100 Leaves.")
    ).toBeTruthy()
    expect(binder.requests).toHaveLength(0)
  })
})
