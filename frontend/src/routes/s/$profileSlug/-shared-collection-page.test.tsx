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
import { getGetSharedCollectionQueryKey } from "@/generated/endpoints/shared/shared"
import { catalogCard } from "@/test-binder-api"
import { Route as SharedRoute } from "./$collectionSlug"

// A Guest opens a Shared Collection: the page is served by the public
// `/shared` routes, so only the network is faked. The generated client, the
// query cache and the router are the app's own.

const API = "/shared/profile-ash/holiday-binder-3f9a2"
const PAGE = "/s/profile-ash/holiday-binder-3f9a2"
const pikachu = catalogCard("pikachu", "Pikachu", "025")

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
          data: [{ card: pikachu, quantity: 2 }],
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
        stub.header?.body ?? {
          data: {
            ownerName: "Ash Ketchum",
            title: "Holiday Binder",
            description: "Cards for the trade table",
          },
        }
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
async function renderSharedPage(path = PAGE) {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  })
  const root = createRootRouteWithContext<{ queryClient: QueryClient }>()({
    component: Outlet,
    notFoundComponent: NotFound,
  })
  const shared = createRoute({
    getParentRoute: () => root,
    path: "/s/$profileSlug/$collectionSlug",
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

// Leaves the page and comes back, as a second visit does.
async function revisit(router: Awaited<ReturnType<typeof renderSharedPage>>) {
  await router.history.push(PAGE)
  await router.load()
}

describe("Shared Collection page", () => {
  it("shows who shares it, its title and description, and the Cards read-only", async () => {
    stubSharedApi()
    await renderSharedPage()

    expect(
      await screen.findByRole("heading", { name: "Holiday Binder", level: 1 })
    ).toBeTruthy()
    expect(screen.getByText("Shared by Ash Ketchum")).toBeTruthy()
    expect(screen.getByText("Cards for the trade table")).toBeTruthy()
    expect(await screen.findByText("Pikachu")).toBeTruthy()
    // No edit controls: a Guest's view carries no quantity stepper.
    expect(screen.queryByRole("spinbutton")).toBeNull()
  })

  it("shows the not-found page for a link that is private, unshared or missing", async () => {
    stubSharedApi({ header: { status: 404, body: { detail: "not found" } } })
    await renderSharedPage()

    expect(
      await screen.findByRole("heading", {
        name: "Collection not found",
        level: 1,
      })
    ).toBeTruthy()
  })

  it("asks for the Cards matching the name filter", async () => {
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

  it("shows each Card's quantity, and no control in the list to change it", async () => {
    stubSharedApi()
    await renderSharedPage()

    expect(await screen.findByText("Quantity: 2")).toBeTruthy()
    const list = within(screen.getByRole("list"))
    expect(list.queryAllByRole("button")).toHaveLength(0)
    expect(list.queryAllByRole("spinbutton")).toHaveLength(0)
    expect(list.queryAllByRole("textbox")).toHaveLength(0)
  })

  it("offers a Retry when the header fails to load, and loads on retry", async () => {
    const stub: SharedStub = {
      header: { status: 500, body: { detail: "boom" } },
    }
    stubSharedApi(stub)
    await renderSharedPage()
    expect(
      await screen.findByText("Could not load this Collection.")
    ).toBeTruthy()

    stub.header = {
      status: 200,
      body: {
        data: {
          ownerName: "Ash Ketchum",
          title: "Holiday Binder",
          description: "",
        },
      },
    }
    fireEvent.click(screen.getByRole("button", { name: "Retry" }))

    expect(
      await screen.findByRole("heading", { name: "Holiday Binder", level: 1 })
    ).toBeTruthy()
  })

  it("shows the not-found page on a revisit after the sharing was stopped", async () => {
    const stub: SharedStub = {}
    stubSharedApi(stub)
    const router = await renderSharedPage()
    await screen.findByRole("heading", { name: "Holiday Binder", level: 1 })

    stub.header = { status: 404, body: { detail: "not found" } }
    await revisit(router)

    expect(
      await screen.findByRole("heading", {
        name: "Collection not found",
        level: 1,
      })
    ).toBeTruthy()
    const match = router.state.matches.find(
      (m) => m.routeId === "/s/$profileSlug/$collectionSlug"
    )
    expect(match?.meta).toContainEqual({ name: "robots", content: "noindex" })
    // The not-found is not kept in the cache for the next visit to read.
    expect(
      router.options.context.queryClient.getQueryData(
        getGetSharedCollectionQueryKey("profile-ash", "holiday-binder-3f9a2")
      )
    ).toBeUndefined()
  })

  it("requests the header again on a revisit", async () => {
    const requests = stubSharedApi()
    const router = await renderSharedPage()
    await screen.findByRole("heading", { name: "Holiday Binder", level: 1 })

    await revisit(router)

    await vi.waitFor(() =>
      expect(requests.filter((r) => r === API)).toHaveLength(2)
    )
  })

  it("keeps the not-found page out of search engines too", async () => {
    stubSharedApi({ header: { status: 404, body: { detail: "not found" } } })
    const router = await renderSharedPage()
    await screen.findByRole("heading", {
      name: "Collection not found",
      level: 1,
    })

    const match = router.state.matches.find(
      (m) => m.routeId === "/s/$profileSlug/$collectionSlug"
    )
    expect(match?.meta).toContainEqual({ name: "robots", content: "noindex" })
  })

  it("is kept out of search engines", async () => {
    stubSharedApi()
    const router = await renderSharedPage()
    await screen.findByRole("heading", { name: "Holiday Binder", level: 1 })

    const match = router.state.matches.find(
      (m) => m.routeId === "/s/$profileSlug/$collectionSlug"
    )
    expect(match?.meta).toContainEqual({ name: "robots", content: "noindex" })
  })
})
