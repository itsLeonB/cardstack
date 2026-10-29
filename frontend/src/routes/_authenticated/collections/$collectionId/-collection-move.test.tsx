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
import { catalogCard } from "@/test-binder-api"
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
  entries: { card: object; quantity: number }[]
  moveStatus: number
  moveBody: FakeBody
  requests: { method: string; path: string; body?: unknown }[]
}

/** The JSON the fake answers with: an envelope, or an error. */
type FakeBody = { data: unknown; meta?: unknown } | { detail: string }

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
    collections: [
      collection("col-1", "Deck A"),
      collection("col-2", "Deck B"),
      collection("col-3", "Binder", { rows: 2, columns: 2, leafCount: 1 }),
    ],
    entries: [],
    moveStatus: 200,
    moveBody: { data: { cards: 0, copies: 0 } },
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
      if (method === "POST" && pathname.endsWith("/entries/move"))
        return json(api.moveStatus, api.moveBody)
      if (method === "GET" && pathname.endsWith("/collections"))
        return json(200, { data: api.collections })
      if (method === "GET" && pathname.endsWith("/collections/col-1"))
        return json(200, { data: collection("col-1", "Deck A") })
      if (method === "GET" && pathname.endsWith("/collections/col-1/entries"))
        return json(200, {
          data: api.entries,
          meta: { total: api.entries.length, page: 1, limit: 100 },
        })
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
async function renderCollectionPage(api: FakeCollectionApi) {
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
  await screen.findByRole("heading", { name: "Deck A", level: 1 })
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
  it("lists the user's Collections with the source and every binder disabled, each with its reason", async () => {
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

    expect(source.matches(":disabled")).toBe(true)
    expect(
      within(sheet).getByText("This is the Collection you are moving from.")
    ).toBeTruthy()
    expect(binder.matches(":disabled")).toBe(true)
    expect(
      within(sheet).getByText(
        "A binder needs a Slot for each card, and a Move chooses none."
      )
    ).toBeTruthy()
    expect(destination.matches(":disabled")).toBe(false)
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
    expect(screen.getByRole("button", { name: "Select" })).toBeTruthy()
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
})
