import { afterEach, describe, expect, it, vi } from "vitest"
import {
  cleanup,
  fireEvent,
  render,
  screen,
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
import { getGetSharedWishlistQueryKey } from "@/generated/endpoints/shared/shared"
import { catalogCard } from "@/test-binder-api"
import { Route as SharedRoute } from "./wishlist"

// A Guest opens a Shared Wishlist: the page is served by the public `/shared`
// routes, so only the network is faked. The generated client, the query cache
// and the router are the app's own.

const API = "/shared/profile-ash/wishlist"
const PAGE = "/s/profile-ash/wishlist"
const pikachu = catalogCard("pikachu", "Pikachu", "025")

const wanted = {
  card: pikachu,
  wantedQuantity: 3,
  ownedQuantity: 2,
  fulfilled: false,
}

// The JSON bodies the API answers with: an envelope, or an error detail.
interface ApiBody {
  data?: unknown
  meta?: { total: number; page: number; limit: number }
  detail?: string
}

interface SharedStub {
  header?: { status: number; body: ApiBody }
  entries?: { status: number; body: ApiBody }
}

// Each request is answered by path; the requests made are kept for asserts.
function stubSharedApi(stub: SharedStub = {}) {
  const requests: string[] = []
  const fetchMock = vi.fn(async (input: RequestInfo | URL) => {
    const url = new URL(String(input))
    requests.push(url.pathname + url.search)
    const reply = (status: number, body: ApiBody) =>
      new Response(JSON.stringify(body), {
        status,
        headers: { "content-type": "application/json" },
      })
    if (url.pathname === `${API}/entries`) {
      return reply(
        stub.entries?.status ?? 200,
        stub.entries?.body ?? {
          data: [wanted],
          meta: { total: 1, page: 1, limit: 100 },
        }
      )
    }
    if (url.pathname === `${API}/facets`) {
      return reply(200, {
        data: { categories: [], expansionSets: [], rarities: [], tags: [] },
      })
    }
    if (url.pathname === "/catalog/series") {
      return reply(200, { data: { series: [], ungroupedExpansionSets: [] } })
    }
    if (url.pathname === API) {
      return reply(
        stub.header?.status ?? 200,
        stub.header?.body ?? { data: { ownerName: "Ash Ketchum" } }
      )
    }
    return reply(500, { detail: `unexpected GET ${url.pathname}` })
  })
  vi.stubGlobal("fetch", fetchMock)
  return requests
}

afterEach(() => {
  cleanup()
  vi.unstubAllGlobals()
})

// The route's own options, untyped: this memory router provides no `auth`
// context, which the route's typed context (the app's root) expects.
function pick(options: {
  validateSearch?: unknown
  loader?: unknown
  head?: unknown
  component?: unknown
}): any {
  const { validateSearch, loader, head, component } = options
  return { validateSearch, loader, head, component }
}

// The real shared page in a memory router, with no signed-in layout around it.
async function renderSharedPage(
  path = PAGE,
  queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  })
) {
  const root = createRootRouteWithContext<{ queryClient: QueryClient }>()({
    component: Outlet,
    notFoundComponent: NotFound,
  })
  const shared = createRoute({
    getParentRoute: () => root,
    path: "/s/$profileSlug/wishlist",
    ...pick(SharedRoute.options),
  })
  const router = createRouter({
    routeTree: root.addChildren([shared]),
    history: createMemoryHistory({ initialEntries: [path] }),
    context: { queryClient },
  })
  render(
    <QueryClientProvider client={queryClient}>
      <RouterProvider router={router} />
    </QueryClientProvider>
  )
  return router
}

const metaOf = (router: Awaited<ReturnType<typeof renderSharedPage>>) =>
  router.state.matches.find((m) => m.routeId === "/s/$profileSlug/wishlist")
    ?.meta

describe("Shared Wishlist page", () => {
  it("shows who shares it and each item's wanted and owned, read-only", async () => {
    stubSharedApi()
    await renderSharedPage()

    expect(
      await screen.findByRole("heading", { name: "Wishlist", level: 1 })
    ).toBeTruthy()
    expect(screen.getByText("Shared by Ash Ketchum")).toBeTruthy()
    expect(await screen.findByText("Pikachu")).toBeTruthy()
    // No edit controls: a Guest's view carries no quantity stepper.
    expect(screen.queryByRole("spinbutton")).toBeNull()
    const list = within(screen.getByRole("list"))
    expect(list.getByText("Owns 2 of 3")).toBeTruthy()
    expect(list.getByText("Still needed")).toBeTruthy()
    expect(list.queryAllByRole("button")).toHaveLength(0)
  })

  it("shows Fulfilled on an item the owner owns enough of", async () => {
    stubSharedApi({
      entries: {
        status: 200,
        body: {
          data: [{ ...wanted, ownedQuantity: 3, fulfilled: true }],
          meta: { total: 1, page: 1, limit: 100 },
        },
      },
    })
    await renderSharedPage()

    const list = within(await screen.findByRole("list"))

    expect(list.getByText("Owns 3 of 3")).toBeTruthy()
    expect(list.getByText("Fulfilled")).toBeTruthy()
  })

  it("shows the not-found page for a link that is private, unshared or missing", async () => {
    stubSharedApi({ header: { status: 404, body: { detail: "not found" } } })
    await renderSharedPage()

    expect(
      await screen.findByRole("heading", {
        name: "Wishlist not found",
        level: 1,
      })
    ).toBeTruthy()
  })

  it("is kept out of search engines, not-found included", async () => {
    stubSharedApi({ header: { status: 404, body: { detail: "not found" } } })
    const router = await renderSharedPage()
    await screen.findByRole("heading", { name: "Wishlist not found", level: 1 })

    expect(metaOf(router)).toContainEqual({
      name: "robots",
      content: "noindex",
    })
    // The not-found is not kept in the cache for the next visit to read.
    expect(
      router.options.context.queryClient.getQueryData(
        getGetSharedWishlistQueryKey("profile-ash")
      )
    ).toBeUndefined()
  })

  it("asks for the items matching the status filter", async () => {
    window.scrollTo = vi.fn()
    const requests = stubSharedApi()
    await renderSharedPage()
    await screen.findByText("Pikachu")

    fireEvent.click(screen.getByRole("button", { name: "Still needed" }))

    await vi.waitFor(() =>
      expect(
        requests.some(
          (r) => r.startsWith(`${API}/entries?`) && r.includes("status=needed")
        )
      ).toBe(true)
    )
  })

  it("asks for the items matching the name filter", async () => {
    window.scrollTo = vi.fn()
    const requests = stubSharedApi()
    await renderSharedPage()
    await screen.findByText("Pikachu")

    fireEvent.change(screen.getByLabelText("Card name"), {
      target: { value: "Pika" },
    })
    fireEvent.click(screen.getByRole("button", { name: "Search" }))

    await vi.waitFor(() =>
      expect(
        requests.some(
          (r) => r.startsWith(`${API}/entries?`) && r.includes("name=Pika")
        )
      ).toBe(true)
    )
  })

  it("offers a Retry when the header fails to load, and loads on retry", async () => {
    const stub: SharedStub = {
      header: { status: 500, body: { detail: "boom" } },
    }
    stubSharedApi(stub)
    await renderSharedPage()
    expect(
      await screen.findByText("Could not load this Wishlist.")
    ).toBeTruthy()

    stub.header = { status: 200, body: { data: { ownerName: "Ash Ketchum" } } }
    fireEvent.click(screen.getByRole("button", { name: "Retry" }))

    expect(
      await screen.findByRole("heading", { name: "Wishlist", level: 1 })
    ).toBeTruthy()
  })

  it("offers none of the owner's editing actions", async () => {
    stubSharedApi()
    await renderSharedPage()
    await screen.findByText("Pikachu")

    for (const name of ["Clear fulfilled", "Fill from sets", "Remove"]) {
      expect(screen.queryByRole("button", { name })).toBeNull()
    }
  })
})
