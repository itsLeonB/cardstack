import { afterEach, describe, expect, it, vi } from "vitest"
import {
  cleanup,
  fireEvent,
  render,
  screen,
  waitFor,
  within,
} from "@testing-library/react"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import {
  Outlet,
  RouterProvider,
  createMemoryHistory,
  createRootRouteWithContext,
  createRoute,
  createRouter,
} from "@tanstack/react-router"
import { NotFound } from "@/components/layout/not-found"
import { catalogCard, fakeBinder, pageBody } from "@/test-binder-api"
import type { FakeBinder } from "@/test-binder-api"
import { stubGridLayout } from "@/test-grid-layout"
import { Route as CollectionRoute } from "./index"

// The Move reads the signed-in user through the Collection page.
// oxlint-disable-next-line anti-slop/no-module-mocking
vi.mock("@/lib/session", () => ({
  useSession: () => ({
    user: { id: "user-1", name: null, email: null },
    isAuthenticated: true,
    isLoading: false,
  }),
}))

afterEach(() => {
  cleanup()
  vi.unstubAllGlobals()
})

// A plain Collection's network, faked: the page's reads and the Move. The
// generated client, the query cache and the invalidation under test are the
// app's own; only the network answers.
interface FakeCollectionApi {
  /** The user's Collections, as the destination sheet lists them. */
  collections: object[]
  /** The Collection being viewed, as GET /collections/col-1 answers. */
  source: object
  /** Status of the user's Collections list; anything but 200 is an error. */
  collectionsStatus: number
  /** When set, the Collections list never answers, so it stays loading. */
  collectionsPending: boolean
  entries: { card: object; quantity: number }[]
  /** The list's total when more Cards match than `entries` holds. Defaults to entries.length. */
  total?: number
  /** What GET .../entries/count answers with. */
  count: { cards: number; copies: number }
  /** When set, GET .../entries/count waits for this before it answers. */
  countGate?: Promise<unknown>
  /** When set, GET .../entries/count fails with this message. */
  countError?: string
  moveStatus: number
  moveBody: FakeBody
  /** The binder's Pages, as GET .../binder/pages/N answers. */
  binder: FakeBinder
  requests: { method: string; path: string; body?: unknown }[]
}

/** The JSON the fake answers with: an envelope, or an error. */
type FakeBody =
  { data: unknown; meta?: unknown } | { code?: string; detail: string }

function json(status: number, body: FakeBody) {
  return new Response(JSON.stringify(body), {
    status,
    headers: { "Content-Type": "application/json" },
  })
}

function collection(
  id: string,
  title: string,
  binder: { rows: number; columns: number; leafCount: number } | null = null
) {
  return {
    id,
    title,
    description: "",
    maxCardCount: 0,
    cardCount: 0,
    binder,
    slug: null,
  }
}

function fakeCollectionApi(
  overrides: Partial<FakeCollectionApi> = {}
): FakeCollectionApi {
  return {
    source: collection("col-1", "Deck A"),
    collectionsStatus: 200,
    collectionsPending: false,
    collections: [
      collection("col-1", "Deck A"),
      collection("col-2", "Deck B"),
      collection("col-3", "Binder", { rows: 2, columns: 2, leafCount: 1 }),
    ],
    entries: [],
    count: { cards: 0, copies: 0 },
    moveStatus: 200,
    moveBody: { data: { cards: 0, copies: 0 } },
    binder: fakeBinder({ rows: 2, columns: 2, leafCount: 1 }),
    requests: [],
    ...overrides,
  }
}

function stubCollectionApi(api: FakeCollectionApi) {
  const fetchMock = vi.fn(
    async (input: RequestInfo | URL, init?: RequestInit) => {
      const { pathname } = new URL(String(input))
      const method = init?.method ?? "GET"
      const body = init?.body ? JSON.parse(String(init.body)) : undefined
      api.requests.push({ method, path: pathname, body })
      if (method === "POST" && pathname.endsWith("/entries/move")) {
        // A Move that succeeds empties Page 1's Slot 1,1, as a binder source does.
        if (api.moveStatus === 200) delete api.binder.filled[1]?.["1,1"]
        return json(api.moveStatus, api.moveBody)
      }
      const page = pathname.match(/\/binder\/pages\/(\d+)$/)
      if (method === "GET" && page)
        return json(200, { data: pageBody(api.binder, Number(page[1])) })
      if (method === "GET" && pathname.endsWith("/collections")) {
        if (api.collectionsPending) return new Promise<Response>(() => {})
        if (api.collectionsStatus !== 200)
          return json(api.collectionsStatus, { detail: "boom" })
        return json(200, { data: api.collections })
      }
      if (method === "GET" && pathname.endsWith("/collections/col-1"))
        return json(200, { data: api.source })
      if (
        method === "GET" &&
        pathname.endsWith("/collections/col-1/entries/count")
      ) {
        if (api.countGate) await api.countGate
        if (api.countError) return json(500, { detail: api.countError })
        return json(200, { data: api.count })
      }
      if (method === "GET" && pathname.endsWith("/collections/col-1/entries")) {
        const pageNumber = Number(
          new URL(String(input)).searchParams.get("page") ?? 1
        )
        return json(200, {
          data: pageNumber === 1 ? api.entries : [],
          meta: {
            total: api.total ?? api.entries.length,
            page: pageNumber,
            limit: 100,
          },
        })
      }
      if (method === "GET" && pathname.endsWith("/facets"))
        return json(200, {
          data: { categories: [], expansionSets: [], rarities: [], tags: [] },
        })
      if (method === "GET" && pathname.endsWith("/catalog/series"))
        return json(200, { data: { series: [] } })
      return json(500, { detail: `unexpected ${method} ${pathname}` })
    }
  )
  vi.stubGlobal("fetch", fetchMock)
  return fetchMock
}

// The real Collection page in a memory router, over the faked network.
async function renderCollectionPage(api: FakeCollectionApi, title = "Deck A") {
  // jsdom does no layout, so the virtual grid needs the stub to render rows.
  stubGridLayout()
  stubCollectionApi(api)
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  })
  const root = createRootRouteWithContext<{ queryClient: QueryClient }>()({
    component: Outlet,
    notFoundComponent: NotFound,
  })
  const collectionRoute = createRoute({
    getParentRoute: () => root,
    path: "/collections/$collectionId/",
    ...pick(CollectionRoute.options),
  })
  const router = createRouter({
    routeTree: root.addChildren([collectionRoute]),
    history: createMemoryHistory({
      initialEntries: ["/collections/col-1/"],
    }),
    context: { queryClient },
  })
  render(
    <QueryClientProvider client={queryClient}>
      <RouterProvider router={router} />
    </QueryClientProvider>
  )
  // The page's title shows once its Collection has loaded.
  await screen.findByRole("heading", { name: title, level: 1 })
  return router
}

function pick(options: {
  loader?: unknown
  component?: unknown
  validateSearch?: unknown
}): any {
  const { loader, component, validateSearch } = options
  return { loader, component, validateSearch }
}

const pikachu = catalogCard("pikachu", "Pikachu", "025")
const eevee = catalogCard("eevee", "Eevee", "133")
const twoCardsThreeCopies = [
  { card: pikachu, quantity: 2 },
  { card: eevee, quantity: 1 },
]
const binderA = collection("col-1", "Binder A", {
  rows: 2,
  columns: 2,
  leafCount: 1,
})
const binderCollections = [
  binderA,
  collection("col-2", "Deck B"),
  collection("col-3", "Binder", { rows: 2, columns: 2, leafCount: 1 }),
]

function requestsTo(api: FakeCollectionApi, method: string, suffix: string) {
  return api.requests.filter(
    (request) => request.method === method && request.path.endsWith(suffix)
  )
}

describe("Collection page: selecting cards to move", () => {
  it("turns selection mode on with a checkbox per card, counts the ticks, and Cancel clears them", async () => {
    await renderCollectionPage(
      fakeCollectionApi({ entries: twoCardsThreeCopies })
    )

    fireEvent.click(screen.getByRole("button", { name: "Select" }))
    fireEvent.click(
      await screen.findByRole("checkbox", {
        name: "Select Pikachu (BS 025)",
      })
    )
    fireEvent.click(
      screen.getByRole("checkbox", { name: "Select Eevee (BS 133)" })
    )
    expect(screen.getByText("2 selected")).toBeTruthy()

    fireEvent.click(screen.getByRole("button", { name: "Cancel" }))
    expect(screen.queryByRole("checkbox")).toBeNull()
    fireEvent.click(screen.getByRole("button", { name: "Select" }))
    expect(
      screen
        .getByRole("checkbox", { name: "Select Pikachu (BS 025)" })
        .matches(":checked")
    ).toBe(false)
    expect(screen.getByText("0 selected")).toBeTruthy()
  })

  it("starts with nothing selected when selection mode is re-entered after Cancel", async () => {
    await renderCollectionPage(
      fakeCollectionApi({ entries: twoCardsThreeCopies })
    )
    fireEvent.click(screen.getByRole("button", { name: "Select" }))
    fireEvent.click(
      await screen.findByRole("checkbox", { name: "Select Pikachu (BS 025)" })
    )
    fireEvent.click(
      screen.getByRole("checkbox", { name: "Select Eevee (BS 133)" })
    )
    expect(screen.getByText("2 selected")).toBeTruthy()

    fireEvent.click(screen.getByRole("button", { name: "Cancel" }))
    fireEvent.click(screen.getByRole("button", { name: "Select" }))

    expect(screen.getByText("0 selected")).toBeTruthy()
    expect(
      screen
        .getByRole("checkbox", { name: "Select Eevee (BS 133)" })
        .matches(":checked")
    ).toBe(false)
  })

  it("returns focus to Select when Cancel leaves selection mode", async () => {
    await renderCollectionPage(
      fakeCollectionApi({ entries: twoCardsThreeCopies })
    )
    fireEvent.click(screen.getByRole("button", { name: "Select" }))
    fireEvent.click(
      await screen.findByRole("checkbox", { name: "Select Pikachu (BS 025)" })
    )

    fireEvent.click(screen.getByRole("button", { name: "Cancel" }))

    await waitFor(() =>
      expect(document.activeElement).toBe(
        screen.getByRole("button", { name: "Select" })
      )
    )
  })
})

describe("Collection page: quantity edits made before selecting", () => {
  it("won't let a Card edited down to 0 copies be ticked, since it has no entry to move", async () => {
    await renderCollectionPage(
      fakeCollectionApi({ entries: twoCardsThreeCopies })
    )
    const decrease = await screen.findByRole("button", {
      name: "Decrease quantity of Pikachu",
    })
    fireEvent.click(decrease)
    fireEvent.click(decrease)

    fireEvent.click(screen.getByRole("button", { name: "Select" }))

    const pikachuBox = screen.getByRole("checkbox", {
      name: "Select Pikachu (BS 025)",
    })
    expect(pikachuBox.matches(":disabled")).toBe(true)
    expect(
      screen
        .getByRole("checkbox", { name: "Select Eevee (BS 133)" })
        .matches(":disabled")
    ).toBe(false)
  })

  it("reports a failed save and keeps the sheet closed when Move to… can't save the edits first", async () => {
    // The fake has no bulk-update route, so saving the edit fails with a 500.
    await renderCollectionPage(
      fakeCollectionApi({ entries: twoCardsThreeCopies })
    )
    fireEvent.click(
      await screen.findByRole("button", {
        name: "Increase quantity of Pikachu",
      })
    )
    fireEvent.click(screen.getByRole("button", { name: "Select" }))
    fireEvent.click(
      screen.getByRole("checkbox", { name: "Select Pikachu (BS 025)" })
    )

    fireEvent.click(screen.getByRole("button", { name: "Move to…" }))

    expect(await screen.findByText(/could not be saved/)).toBeTruthy()
    expect(screen.queryByRole("dialog")).toBeNull()
  })
})

describe("Collection page: selecting cards in a binder", () => {
  it("turns selection mode on for a binder's card list, and Move to… offers plain Collections only", async () => {
    await renderCollectionPage(
      fakeCollectionApi({
        source: binderA,
        collections: binderCollections,
        entries: twoCardsThreeCopies,
      }),
      "Binder A"
    )
    fireEvent.click(await screen.findByRole("button", { name: "Select" }))
    fireEvent.click(
      await screen.findByRole("checkbox", { name: "Select Pikachu (BS 025)" })
    )
    fireEvent.click(
      screen.getByRole("checkbox", { name: "Select Eevee (BS 133)" })
    )
    expect(screen.getByText("2 selected")).toBeTruthy()

    fireEvent.click(screen.getByRole("button", { name: "Move to…" }))
    const sheet = await screen.findByRole("dialog")
    expect(
      within(sheet)
        .getByRole("button", { name: /Deck B/ })
        .getAttribute("aria-disabled")
    ).toBe("false")
    expect(
      within(sheet)
        .getByRole("button", { name: /Binder A/ })
        .getAttribute("aria-disabled")
    ).toBe("true")
  })
})

describe("Collection page: the 100-card limit", () => {
  it("disables the unticked checkboxes once 100 are ticked and says why", async () => {
    const entries = Array.from({ length: 101 }, (_, index) => ({
      card: catalogCard(`card-${index}`, `Card ${index}`, String(index)),
      quantity: 1,
    }))
    // A window tall enough to render every row, so all 101 checkboxes exist.
    vi.stubGlobal("innerHeight", 100_000)
    await renderCollectionPage(fakeCollectionApi({ entries }))
    fireEvent.click(screen.getByRole("button", { name: "Select" }))
    const boxes = await screen.findAllByRole("checkbox")
    expect(boxes).toHaveLength(101)

    boxes.slice(0, 100).forEach((box) => fireEvent.click(box))

    expect(screen.getByText("Up to 100 cards at a time")).toBeTruthy()
    expect(screen.getByText("100 selected")).toBeTruthy()
    expect(boxes[100].matches(":disabled")).toBe(true)
  })
})

describe("Collection page: the Move to sheet", () => {
  it("lists the user's Collections with the source and every binder refused, each with its reason", async () => {
    await renderCollectionPage(
      fakeCollectionApi({ entries: twoCardsThreeCopies })
    )
    fireEvent.click(screen.getByRole("button", { name: "Select" }))
    fireEvent.click(
      await screen.findByRole("checkbox", { name: "Select Pikachu (BS 025)" })
    )
    fireEvent.click(screen.getByRole("button", { name: "Move to…" }))

    const sheet = await screen.findByRole("dialog")
    const source = within(sheet).getByRole("button", { name: /Deck A/ })
    const binder = within(sheet).getByRole("button", { name: /Binder/ })
    const destination = within(sheet).getByRole("button", { name: /Deck B/ })

    // aria-disabled, not disabled: a refused destination stays focusable.
    expect(source.getAttribute("aria-disabled")).toBe("true")
    expect(binder.getAttribute("aria-disabled")).toBe("true")
    expect(destination.getAttribute("aria-disabled")).toBe("false")
    expect(
      document.getElementById(source.getAttribute("aria-describedby") ?? "")
        ?.textContent
    ).toBe("This is the Collection you are moving from.")
    expect(
      document.getElementById(binder.getAttribute("aria-describedby") ?? "")
        ?.textContent
    ).toBe("A binder needs a Slot for each card, and a Move chooses none.")

    source.focus()
    expect(document.activeElement).toBe(source)
    fireEvent.click(source)
    expect(screen.queryByRole("alertdialog")).toBeNull()
  })

  it("shows a loading state while the Collections load", async () => {
    await renderCollectionPage(
      fakeCollectionApi({
        collectionsPending: true,
        entries: twoCardsThreeCopies,
      })
    )
    fireEvent.click(screen.getByRole("button", { name: "Select" }))
    fireEvent.click(
      await screen.findByRole("checkbox", { name: "Select Pikachu (BS 025)" })
    )
    fireEvent.click(screen.getByRole("button", { name: "Move to…" }))

    const sheet = await screen.findByRole("dialog")
    expect(within(sheet).getByText("Loading your Collections…")).toBeTruthy()
    expect(within(sheet).queryByRole("button", { name: /Deck B/ })).toBeNull()
  })

  it("shows an error, not an empty sheet, when the Collections fail to load", async () => {
    await renderCollectionPage(
      fakeCollectionApi({
        collectionsStatus: 500,
        entries: twoCardsThreeCopies,
      })
    )
    fireEvent.click(screen.getByRole("button", { name: "Select" }))
    fireEvent.click(
      await screen.findByRole("checkbox", { name: "Select Pikachu (BS 025)" })
    )
    fireEvent.click(screen.getByRole("button", { name: "Move to…" }))

    const sheet = await screen.findByRole("dialog")
    expect(
      await within(sheet).findByText("Could not load your Collections.")
    ).toBeTruthy()
    expect(within(sheet).queryByRole("button", { name: /Deck B/ })).toBeNull()
  })

  it("returns focus to Move to… when the sheet closes without a destination", async () => {
    await renderCollectionPage(
      fakeCollectionApi({ entries: twoCardsThreeCopies })
    )
    fireEvent.click(screen.getByRole("button", { name: "Select" }))
    fireEvent.click(
      await screen.findByRole("checkbox", { name: "Select Pikachu (BS 025)" })
    )
    fireEvent.click(screen.getByRole("button", { name: "Move to…" }))
    fireEvent.keyDown(await screen.findByRole("dialog"), { key: "Escape" })

    await waitFor(() =>
      expect(document.activeElement).toBe(
        screen.getByRole("button", { name: "Move to…" })
      )
    )
  })
})

describe("Collection page: selecting every card matching a filter", () => {
  // Selects the two loaded cards, which is what enables the banner.
  async function selectLoadedPage() {
    fireEvent.click(screen.getByRole("button", { name: "Select" }))
    fireEvent.click(
      await screen.findByRole("checkbox", { name: "Select Pikachu (BS 025)" })
    )
    fireEvent.click(
      screen.getByRole("checkbox", { name: "Select Eevee (BS 133)" })
    )
  }

  it("offers nothing when the filter fits on one page", async () => {
    await renderCollectionPage(
      fakeCollectionApi({ entries: twoCardsThreeCopies })
    )
    await selectLoadedPage()

    expect(
      screen.queryByRole("button", { name: /matching this filter/ })
    ).toBeNull()
  })

  it("offers Select all N matching as soon as selection mode is on, with nothing ticked", async () => {
    await renderCollectionPage(
      fakeCollectionApi({ entries: twoCardsThreeCopies, total: 250 })
    )
    fireEvent.click(screen.getByRole("button", { name: "Select" }))

    expect(
      await screen.findByRole("button", {
        name: "Select all 250 matching this filter",
      })
    ).toBeTruthy()
  })

  it("offers Select all N once 200 of 250 matching cards are loaded, with nothing ticked", async () => {
    // Page 2 is not fetched here: the 200 loaded cards stand for the loaded pages.
    const loaded = Array.from({ length: 200 }, (_, i) => ({
      card: { ...pikachu, id: String(i) },
      quantity: 1,
    }))
    await renderCollectionPage(
      fakeCollectionApi({ entries: loaded, total: 250 })
    )
    fireEvent.click(screen.getByRole("button", { name: "Select" }))

    expect(
      await screen.findByRole("button", {
        name: "Select all 250 matching this filter",
      })
    ).toBeTruthy()
  })

  it("moves every matching card by filter, and the confirm dialog states the server's counts", async () => {
    const api = fakeCollectionApi({
      entries: twoCardsThreeCopies,
      total: 250,
      count: { cards: 37, copies: 64 },
      moveBody: { data: { cards: 37, copies: 64 } },
    })
    await renderCollectionPage(api)
    await selectLoadedPage()
    fireEvent.click(
      screen.getByRole("button", {
        name: "Select all 250 matching this filter",
      })
    )
    expect(
      screen.getByText("All 250 matching this filter selected")
    ).toBeTruthy()

    fireEvent.click(screen.getByRole("button", { name: "Move to…" }))
    fireEvent.click(
      within(await screen.findByRole("dialog")).getByRole("button", {
        name: /Deck B/,
      })
    )
    const confirm = await screen.findByRole("alertdialog")
    expect(
      await within(confirm).findByText("Move 37 cards to Deck B?")
    ).toBeTruthy()
    expect(
      within(confirm).getByText(
        "This moves 37 cards and 64 copies from Deck A to Deck B. There is no undo."
      )
    ).toBeTruthy()

    fireEvent.click(within(confirm).getByRole("button", { name: "Move" }))
    expect(
      await screen.findByText("Moved 37 cards (64 copies) to Deck B.")
    ).toBeTruthy()
    const [move] = requestsTo(api, "POST", "/collections/col-1/entries/move")
    expect(move.body).toEqual({ destinationId: "col-2", filter: {} })
  })

  it("sends the current filter with the Move, not the ticked ids", async () => {
    const api = fakeCollectionApi({
      entries: twoCardsThreeCopies,
      total: 250,
      count: { cards: 3, copies: 4 },
      moveBody: { data: { cards: 3, copies: 4 } },
    })
    const router = await renderCollectionPage(api)
    await router.history.push("/collections/col-1/?name=pika")
    await selectLoadedPage()
    fireEvent.click(
      screen.getByRole("button", {
        name: "Select all 250 matching this filter",
      })
    )

    fireEvent.click(screen.getByRole("button", { name: "Move to…" }))
    fireEvent.click(
      within(await screen.findByRole("dialog")).getByRole("button", {
        name: /Deck B/,
      })
    )
    const confirm = await screen.findByRole("alertdialog")
    await within(confirm).findByText("Move 3 cards to Deck B?")
    fireEvent.click(within(confirm).getByRole("button", { name: "Move" }))

    await screen.findByText("Moved 3 cards (4 copies) to Deck B.")
    const [move] = requestsTo(api, "POST", "/collections/col-1/entries/move")
    expect(move.body).toEqual({
      destinationId: "col-2",
      filter: { name: "pika" },
    })
  })

  it("clears the filter selection when the filter changes", async () => {
    const router = await renderCollectionPage(
      fakeCollectionApi({ entries: twoCardsThreeCopies, total: 250 })
    )
    await selectLoadedPage()
    fireEvent.click(
      screen.getByRole("button", {
        name: "Select all 250 matching this filter",
      })
    )

    await router.history.push("/collections/col-1/?name=pika")

    await waitFor(() =>
      expect(
        screen.queryByText("All 250 matching this filter selected")
      ).toBeNull()
    )
    expect(screen.getByText("0 selected")).toBeTruthy()
  })

  it("closes the sheet and sends no Move when the filter changes while it is open", async () => {
    const api = fakeCollectionApi({ entries: twoCardsThreeCopies, total: 250 })
    const router = await renderCollectionPage(api)
    await selectEveryMatch()
    fireEvent.click(screen.getByRole("button", { name: "Move to…" }))
    await screen.findByRole("dialog")

    await router.history.push("/collections/col-1/?name=pika")

    await waitFor(() => expect(screen.queryByRole("dialog")).toBeNull())
    expect(requestsTo(api, "POST", "/collections/col-1/entries/move")).toEqual(
      []
    )
  })

  it("closes the confirm dialog and sends no Move when the filter changes while it is open", async () => {
    const api = fakeCollectionApi({
      entries: twoCardsThreeCopies,
      total: 250,
      count: { cards: 37, copies: 64 },
    })
    const router = await renderCollectionPage(api)
    await selectEveryMatch()
    const confirm = await openConfirmToDeckB()
    await within(confirm).findByText("Move 37 cards to Deck B?")

    await router.history.push("/collections/col-1/?name=pika")

    await waitFor(() => expect(screen.queryByRole("alertdialog")).toBeNull())
    expect(requestsTo(api, "POST", "/collections/col-1/entries/move")).toEqual(
      []
    )
  })

  it("blocks a filter Move while a quantity edit can't be saved, and says so", async () => {
    // The fake has no bulk-update route, so saving the edit fails with a 500.
    const api = fakeCollectionApi({ entries: twoCardsThreeCopies, total: 250 })
    await renderCollectionPage(api)
    fireEvent.click(
      await screen.findByRole("button", {
        name: "Increase quantity of Pikachu",
      })
    )
    fireEvent.click(screen.getByRole("button", { name: "Select" }))
    fireEvent.click(
      screen.getByRole("button", {
        name: "Select all 250 matching this filter",
      })
    )

    fireEvent.click(screen.getByRole("button", { name: "Move to…" }))

    expect(await screen.findByText(/could not be saved/)).toBeTruthy()
    expect(screen.queryByRole("dialog")).toBeNull()
    expect(requestsTo(api, "POST", "/collections/col-1/entries/move")).toEqual(
      []
    )
  })

  it("clears the filter selection when Cancel leaves selection mode", async () => {
    await renderCollectionPage(
      fakeCollectionApi({ entries: twoCardsThreeCopies, total: 250 })
    )
    await selectLoadedPage()
    fireEvent.click(
      screen.getByRole("button", {
        name: "Select all 250 matching this filter",
      })
    )

    fireEvent.click(screen.getByRole("button", { name: "Cancel" }))
    fireEvent.click(screen.getByRole("button", { name: "Select" }))

    expect(
      screen.queryByText("All 250 matching this filter selected")
    ).toBeNull()
    expect(screen.getByText("0 selected")).toBeTruthy()
  })

  // Selects every loaded card and then Select all matching, so the bar is in filter mode.
  async function selectEveryMatch() {
    await selectLoadedPage()
    fireEvent.click(
      screen.getByRole("button", {
        name: "Select all 250 matching this filter",
      })
    )
  }

  // Opens the confirm dialog for a Move to Deck B from the filter selection.
  async function openConfirmToDeckB() {
    fireEvent.click(screen.getByRole("button", { name: "Move to…" }))
    fireEvent.click(
      within(await screen.findByRole("dialog")).getByRole("button", {
        name: /Deck B/,
      })
    )
    return screen.findByRole("alertdialog")
  }

  it("keeps Confirm disabled while the count is pending, and enables it with the server's numbers", async () => {
    let release!: () => void
    await renderCollectionPage(
      fakeCollectionApi({
        entries: twoCardsThreeCopies,
        total: 250,
        count: { cards: 37, copies: 64 },
        countGate: new Promise<void>((resolve) => {
          release = resolve
        }),
      })
    )
    await selectEveryMatch()

    const confirm = await openConfirmToDeckB()
    expect(
      within(confirm).getByText("Counting the matching cards…")
    ).toBeTruthy()
    expect(
      within(confirm).getByRole("button", { name: "Move" }).matches(":disabled")
    ).toBe(true)

    release()
    expect(
      await within(confirm).findByText("Move 37 cards to Deck B?")
    ).toBeTruthy()
    expect(
      within(confirm).getByRole("button", { name: "Move" }).matches(":disabled")
    ).toBe(false)
  })

  it("shows a count error inline and keeps Confirm disabled", async () => {
    await renderCollectionPage(
      fakeCollectionApi({
        entries: twoCardsThreeCopies,
        total: 250,
        countError: "Could not count right now.",
      })
    )
    await selectEveryMatch()

    const confirm = await openConfirmToDeckB()
    expect((await within(confirm).findByRole("alert")).textContent).toBe(
      "Could not count right now."
    )
    expect(
      within(confirm).getByRole("button", { name: "Move" }).matches(":disabled")
    ).toBe(true)
  })

  it("does not show the previous count when the dialog reopens and the count is fetched again", async () => {
    const api = fakeCollectionApi({
      entries: twoCardsThreeCopies,
      total: 250,
      count: { cards: 37, copies: 64 },
    })
    await renderCollectionPage(api)
    await selectEveryMatch()
    const first = await openConfirmToDeckB()
    await within(first).findByText("Move 37 cards to Deck B?")
    fireEvent.click(within(first).getByRole("button", { name: "Cancel" }))
    await waitFor(() => expect(screen.queryByRole("alertdialog")).toBeNull())

    // The next count call never answers, so anything shown is the cached 37.
    api.countGate = new Promise(() => {})
    const second = await openConfirmToDeckB()

    expect(
      within(second).getByText("Counting the matching cards…")
    ).toBeTruthy()
    expect(within(second).queryByText("Move 37 cards to Deck B?")).toBeNull()
    expect(
      within(second).getByRole("button", { name: "Move" }).matches(":disabled")
    ).toBe(true)
  })
})

describe("Collection page: confirming a Move", () => {
  it("states the cards, copies and both Collection names, and says there is no undo", async () => {
    await renderCollectionPage(
      fakeCollectionApi({ entries: twoCardsThreeCopies })
    )
    fireEvent.click(screen.getByRole("button", { name: "Select" }))
    fireEvent.click(
      await screen.findByRole("checkbox", { name: "Select Pikachu (BS 025)" })
    )
    fireEvent.click(
      screen.getByRole("checkbox", { name: "Select Eevee (BS 133)" })
    )
    fireEvent.click(screen.getByRole("button", { name: "Move to…" }))
    fireEvent.click(
      within(await screen.findByRole("dialog")).getByRole("button", {
        name: /Deck B/,
      })
    )

    const confirm = await screen.findByRole("alertdialog")
    expect(within(confirm).getByText("Move 2 cards to Deck B?")).toBeTruthy()
    expect(
      within(confirm).getByText(
        "This moves 2 cards and 3 copies from Deck A to Deck B. There is no undo."
      )
    ).toBeTruthy()
  })

  it("says the Slots will be emptied when the source is a binder", async () => {
    await renderCollectionPage(
      fakeCollectionApi({
        source: binderA,
        collections: binderCollections,
        entries: twoCardsThreeCopies,
      }),
      "Binder A"
    )
    fireEvent.click(await screen.findByRole("button", { name: "Select" }))
    fireEvent.click(
      await screen.findByRole("checkbox", { name: "Select Pikachu (BS 025)" })
    )
    fireEvent.click(
      screen.getByRole("checkbox", { name: "Select Eevee (BS 133)" })
    )
    fireEvent.click(screen.getByRole("button", { name: "Move to…" }))
    fireEvent.click(
      within(await screen.findByRole("dialog")).getByRole("button", {
        name: /Deck B/,
      })
    )

    const confirm = await screen.findByRole("alertdialog")
    expect(
      within(confirm).getByText(
        "This moves 2 cards and 3 copies from Binder A to Deck B. The Slots holding them will be emptied. There is no undo."
      )
    ).toBeTruthy()
  })

  it("refetches the binder's pages after a Move out of it", async () => {
    const api = fakeCollectionApi({
      source: binderA,
      collections: binderCollections,
      entries: twoCardsThreeCopies,
      moveBody: { data: { cards: 2, copies: 3 } },
      binder: fakeBinder({
        rows: 2,
        columns: 2,
        leafCount: 1,
        catalog: [pikachu],
        filled: { 1: { "1,1": { state: "card", card: pikachu } } },
      }),
    })
    await renderCollectionPage(api, "Binder A")
    expect(
      await screen.findByRole("button", {
        name: "Row 1, column 1: Pikachu, Base Set",
      })
    ).toBeTruthy()

    fireEvent.click(await screen.findByRole("button", { name: "Select" }))
    fireEvent.click(
      await screen.findByRole("checkbox", { name: "Select Pikachu (BS 025)" })
    )
    fireEvent.click(
      screen.getByRole("checkbox", { name: "Select Eevee (BS 133)" })
    )
    fireEvent.click(screen.getByRole("button", { name: "Move to…" }))
    fireEvent.click(
      within(await screen.findByRole("dialog")).getByRole("button", {
        name: /Deck B/,
      })
    )
    fireEvent.click(
      within(await screen.findByRole("alertdialog")).getByRole("button", {
        name: "Move",
      })
    )

    expect(
      await screen.findByText("Moved 2 cards (3 copies) to Deck B.")
    ).toBeTruthy()
    expect(
      await screen.findByRole("button", { name: "Row 1, column 1: empty" })
    ).toBeTruthy()
    expect(
      screen.queryByRole("button", {
        name: "Row 1, column 1: Pikachu, Base Set",
      })
    ).toBeNull()
  })

  it("moves the selected cards, announces the result, and leaves selection mode", async () => {
    const api = fakeCollectionApi({
      entries: twoCardsThreeCopies,
      moveBody: { data: { cards: 2, copies: 3 } },
    })
    await renderCollectionPage(api)
    fireEvent.click(screen.getByRole("button", { name: "Select" }))
    fireEvent.click(
      await screen.findByRole("checkbox", { name: "Select Pikachu (BS 025)" })
    )
    fireEvent.click(
      screen.getByRole("checkbox", { name: "Select Eevee (BS 133)" })
    )
    fireEvent.click(screen.getByRole("button", { name: "Move to…" }))
    fireEvent.click(
      within(await screen.findByRole("dialog")).getByRole("button", {
        name: /Deck B/,
      })
    )
    const entryListsBefore = requestsTo(
      api,
      "GET",
      "/collections/col-1/entries"
    ).length

    fireEvent.click(
      within(await screen.findByRole("alertdialog")).getByRole("button", {
        name: "Move",
      })
    )

    expect(
      await screen.findByText("Moved 2 cards (3 copies) to Deck B.")
    ).toBeTruthy()
    expect(
      requestsTo(api, "POST", "/collections/col-1/entries/move")
    ).toHaveLength(1)
    expect(
      requestsTo(api, "POST", "/collections/col-1/entries/move")[0].body
    ).toEqual({ cardIds: ["pikachu", "eevee"], destinationId: "col-2" })
    await waitFor(() =>
      expect(
        requestsTo(api, "GET", "/collections/col-1/entries").length
      ).toBeGreaterThan(entryListsBefore)
    )
    expect(screen.queryByRole("checkbox")).toBeNull()
    const select = screen.getByRole("button", { name: "Select" })
    expect(select).toBeTruthy()
    await waitFor(() => expect(document.activeElement).toBe(select))
  })

  it("shows the API's message inline and keeps the selection when the Move is refused", async () => {
    const api = fakeCollectionApi({
      entries: twoCardsThreeCopies,
      moveStatus: 422,
      moveBody: {
        detail: "Deck B would pass its limit of 10 cards by 2 copies",
      },
    })
    await renderCollectionPage(api)
    fireEvent.click(screen.getByRole("button", { name: "Select" }))
    fireEvent.click(
      await screen.findByRole("checkbox", { name: "Select Pikachu (BS 025)" })
    )
    fireEvent.click(screen.getByRole("button", { name: "Move to…" }))
    fireEvent.click(
      within(await screen.findByRole("dialog")).getByRole("button", {
        name: /Deck B/,
      })
    )
    fireEvent.click(
      within(await screen.findByRole("alertdialog")).getByRole("button", {
        name: "Move",
      })
    )

    const alert = await within(screen.getByRole("alertdialog")).findByRole(
      "alert"
    )
    expect(alert.textContent).toBe(
      "Deck B would pass its limit of 10 cards by 2 copies"
    )
    fireEvent.click(
      within(screen.getByRole("alertdialog")).getByRole("button", {
        name: "Cancel",
      })
    )
    expect(screen.getByText("1 selected")).toBeTruthy()
  })

  it("shows the API's message inline and keeps the selection when the destination is gone (404)", async () => {
    const api = fakeCollectionApi({
      entries: twoCardsThreeCopies,
      moveStatus: 404,
      moveBody: { detail: "That Collection no longer exists." },
    })
    await renderCollectionPage(api)
    fireEvent.click(screen.getByRole("button", { name: "Select" }))
    fireEvent.click(
      await screen.findByRole("checkbox", { name: "Select Pikachu (BS 025)" })
    )
    fireEvent.click(screen.getByRole("button", { name: "Move to…" }))
    fireEvent.click(
      within(await screen.findByRole("dialog")).getByRole("button", {
        name: /Deck B/,
      })
    )
    fireEvent.click(
      within(await screen.findByRole("alertdialog")).getByRole("button", {
        name: "Move",
      })
    )

    const alert = await within(screen.getByRole("alertdialog")).findByRole(
      "alert"
    )
    expect(alert.textContent).toBe("That Collection no longer exists.")
    fireEvent.click(
      within(screen.getByRole("alertdialog")).getByRole("button", {
        name: "Cancel",
      })
    )
    expect(screen.getByText("1 selected")).toBeTruthy()
  })

  it("shows the API's message inline and keeps the selection when the destination is a binder (409)", async () => {
    const api = fakeCollectionApi({
      entries: twoCardsThreeCopies,
      moveStatus: 409,
      moveBody: {
        code: "move_to_binder",
        detail:
          "A binder can't take a Move: choose a Collection without a Layout.",
      },
    })
    await renderCollectionPage(api)
    fireEvent.click(screen.getByRole("button", { name: "Select" }))
    fireEvent.click(
      await screen.findByRole("checkbox", { name: "Select Pikachu (BS 025)" })
    )
    fireEvent.click(screen.getByRole("button", { name: "Move to…" }))
    fireEvent.click(
      within(await screen.findByRole("dialog")).getByRole("button", {
        name: /Deck B/,
      })
    )
    fireEvent.click(
      within(await screen.findByRole("alertdialog")).getByRole("button", {
        name: "Move",
      })
    )

    const alert = await within(screen.getByRole("alertdialog")).findByRole(
      "alert"
    )
    expect(alert.textContent).toBe(
      "A binder can't take a Move: choose a Collection without a Layout."
    )
    fireEvent.click(
      within(screen.getByRole("alertdialog")).getByRole("button", {
        name: "Cancel",
      })
    )
    expect(screen.getByText("1 selected")).toBeTruthy()
  })

  it("returns focus to Move to… when the confirm dialog closes without moving", async () => {
    await renderCollectionPage(
      fakeCollectionApi({ entries: twoCardsThreeCopies })
    )
    fireEvent.click(screen.getByRole("button", { name: "Select" }))
    fireEvent.click(
      await screen.findByRole("checkbox", { name: "Select Pikachu (BS 025)" })
    )
    fireEvent.click(screen.getByRole("button", { name: "Move to…" }))
    fireEvent.click(
      within(await screen.findByRole("dialog")).getByRole("button", {
        name: /Deck B/,
      })
    )
    fireEvent.click(
      within(await screen.findByRole("alertdialog")).getByRole("button", {
        name: "Cancel",
      })
    )

    await waitFor(() =>
      expect(document.activeElement).toBe(
        screen.getByRole("button", { name: "Move to…" })
      )
    )
  })
})
