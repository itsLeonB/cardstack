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
import {
  getGetSharedCollectionPageQueryKey,
  getGetSharedCollectionQueryKey,
} from "@/generated/endpoints/shared/shared"
import { catalogCard } from "@/test-binder-api"
import { Route as SharedRoute } from "./$collectionSlug"

// A Guest opens a Shared Collection: the page is served by the public
// `/shared` routes, so only the network is faked. The generated client, the
// query cache and the router are the app's own.

const API = "/shared/profile-ash/holiday-binder-3f9a2"
const PAGE = "/s/profile-ash/holiday-binder-3f9a2"
const pikachu = catalogCard("pikachu", "Pikachu", "025")

// A binder of one Leaf: Page 1 is its front, Page 2 its back, 2 by 2 Slots each.
const binderLayout = { rows: 2, columns: 2, leafCount: 1, capacity: 8 }
const binderHeader = {
  data: {
    ownerName: "Ash Ketchum",
    title: "Holiday Binder",
    description: "",
    binder: { layout: binderLayout, unidentifiedCount: 2 },
  },
}
const frontPage = {
  page: 1,
  leaf: 1,
  side: "front",
  layout: binderLayout,
  cardTotal: 3,
  unidentifiedCount: 1,
  slots: [
    { row: 1, column: 1, state: "card", card: pikachu },
    { row: 1, column: 2, state: "empty", card: null },
    { row: 2, column: 1, state: "unidentified", card: null },
    { row: 2, column: 2, state: "empty", card: null },
  ],
}
const backPage = {
  page: 2,
  leaf: 1,
  side: "back",
  layout: binderLayout,
  cardTotal: 3,
  unidentifiedCount: 1,
  slots: [
    { row: 1, column: 1, state: "empty", card: null },
    { row: 1, column: 2, state: "empty", card: null },
    { row: 2, column: 1, state: "empty", card: null },
    { row: 2, column: 2, state: "unidentified", card: null },
  ],
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
  // A binder's Pages in order, from Page 1; any other Page is a not-found.
  pages?: unknown[]
  // Answers every Page request with this reply instead, e.g. a failure.
  pageReply?: { status: number; body: ApiBody }
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
    const pageNumber = url.pathname.match(new RegExp(`^${API}/pages/(\\d+)$`))
    if (pageNumber) {
      if (stub.pageReply)
        return reply(stub.pageReply.status, stub.pageReply.body)
      const page = stub.pages?.[Number(pageNumber[1]) - 1]
      return page
        ? reply(200, { data: page })
        : reply(404, { detail: "not found" })
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

  it("shows a binder's Pages of Slots read-only, with its Unidentified count", async () => {
    stubSharedApi({
      header: { status: 200, body: binderHeader },
      pages: [frontPage],
    })
    await renderSharedPage()

    const binder = await screen.findByRole("region", { name: "Binder page" })
    expect(within(binder).getByText("Leaf 1, front")).toBeTruthy()
    expect(screen.getByText("Unidentified: 2")).toBeTruthy()
    expect(
      within(binder).getByRole("group", {
        name: "Row 1, column 1: Pikachu, Base Set",
      })
    ).toBeTruthy()
    expect(
      within(binder).getByRole("group", { name: "Row 1, column 2: empty" })
    ).toBeTruthy()
  })

  it("shows an Unidentified Slot as text, never as a Card", async () => {
    stubSharedApi({
      header: { status: 200, body: binderHeader },
      pages: [frontPage],
    })
    await renderSharedPage()

    const binder = await screen.findByRole("region", { name: "Binder page" })
    const unidentified = within(binder).getByRole("group", {
      name: "Row 2, column 1: Unidentified",
    })
    expect(unidentified.textContent).toBe("Unidentified")
    expect(unidentified.querySelector("img")).toBeNull()
  })

  it("turns through the binder's Pages one at a time", async () => {
    const requests = stubSharedApi({
      header: { status: 200, body: binderHeader },
      pages: [frontPage, backPage],
    })
    await renderSharedPage()
    await screen.findByText("Page 1 of 2")

    fireEvent.click(screen.getByRole("button", { name: "Next page" }))

    const binder = await screen.findByRole("region", { name: "Binder page" })
    expect(await within(binder).findByText("Leaf 1, back")).toBeTruthy()
    expect(
      within(binder).getByRole("group", {
        name: "Row 2, column 2: Unidentified",
      })
    ).toBeTruthy()
    expect(screen.getByText("Page 2 of 2")).toBeTruthy()
    expect(
      screen.getByRole("button", { name: "Next page" }).hasAttribute("disabled")
    ).toBe(true)
    expect(requests).toContain(`${API}/pages/2`)

    fireEvent.click(screen.getByRole("button", { name: "Previous page" }))

    expect(await screen.findByText("Page 1 of 2")).toBeTruthy()
    expect(
      screen
        .getByRole("button", { name: "Previous page" })
        .hasAttribute("disabled")
    ).toBe(true)
  })

  it("offers no edit controls on the binder", async () => {
    stubSharedApi({
      header: { status: 200, body: binderHeader },
      pages: [frontPage],
    })
    await renderSharedPage()

    const binder = await screen.findByRole("region", { name: "Binder page" })
    expect(within(binder).queryAllByRole("button")).toHaveLength(0)
    for (const name of [
      "Add Leaf before",
      "Add Leaf after",
      "Remove Leaf",
      "Change layout",
      "Remove binder",
      "Go to page",
    ]) {
      expect(screen.queryByRole("button", { name })).toBeNull()
    }
    expect(screen.queryByRole("spinbutton")).toBeNull()
  })

  it("shows the not-found page for a Page the binder does not have", async () => {
    stubSharedApi({
      header: { status: 200, body: binderHeader },
      pages: [frontPage],
    })
    await renderSharedPage()
    await screen.findByText("Page 1 of 2")

    fireEvent.click(screen.getByRole("button", { name: "Next page" }))

    expect(
      await screen.findByRole("heading", { name: "Page not found", level: 2 })
    ).toBeTruthy()
  })

  it("keeps the page noindex when a Page the binder does not have is asked for", async () => {
    stubSharedApi({
      header: { status: 200, body: binderHeader },
      pages: [frontPage],
    })
    const router = await renderSharedPage()
    await screen.findByText("Page 1 of 2")

    fireEvent.click(screen.getByRole("button", { name: "Next page" }))

    expect(
      await screen.findByRole("heading", { name: "Page not found", level: 2 })
    ).toBeTruthy()
    const match = router.state.matches.find(
      (m) => m.routeId === "/s/$profileSlug/$collectionSlug"
    )
    expect(match?.meta).toContainEqual({ name: "robots", content: "noindex" })
  })

  it("shows the Cards and Unidentified count against the binder's Slots", async () => {
    stubSharedApi({
      header: { status: 200, body: binderHeader },
      pages: [frontPage],
    })
    await renderSharedPage()

    expect(
      await screen.findByText("Cards and Unidentified: 3 of 8 slots")
    ).toBeTruthy()
    expect(screen.getByText("2 × 2 slots per Page · 1 Leaf")).toBeTruthy()
  })

  it("says Leaves when the binder has more than one", async () => {
    const twoLeaves = {
      data: {
        ...binderHeader.data,
        binder: {
          layout: { ...binderLayout, leafCount: 2, capacity: 16 },
          unidentifiedCount: 2,
        },
      },
    }
    stubSharedApi({
      header: { status: 200, body: twoLeaves },
      pages: [frontPage],
    })
    await renderSharedPage()

    expect(
      await screen.findByText("2 × 2 slots per Page · 2 Leaves")
    ).toBeTruthy()
  })

  it("offers a Retry when a Page fails to load, and shows the Page on retry", async () => {
    const stub: SharedStub = {
      header: { status: 200, body: binderHeader },
      pages: [frontPage],
      pageReply: { status: 500, body: { detail: "boom" } },
    }
    stubSharedApi(stub)
    await renderSharedPage()

    expect(
      within(await screen.findByRole("alert")).getByText("boom")
    ).toBeTruthy()

    stub.pageReply = undefined
    fireEvent.click(screen.getByRole("button", { name: "Retry" }))

    expect(
      await screen.findByRole("group", {
        name: "Row 1, column 1: Pikachu, Base Set",
      })
    ).toBeTruthy()
  })

  it("does not keep a Page cached once its visit is over", async () => {
    stubSharedApi({
      header: { status: 200, body: binderHeader },
      pages: [frontPage],
    })
    const queryClient = new QueryClient({
      defaultOptions: { queries: { retry: false } },
    })
    await renderSharedPage(PAGE, queryClient)
    await screen.findByRole("group", {
      name: "Row 1, column 1: Pikachu, Base Set",
    })

    cleanup()

    const pageKey = getGetSharedCollectionPageQueryKey(
      "profile-ash",
      "holiday-binder-3f9a2",
      1
    )
    await vi.waitFor(() =>
      expect(queryClient.getQueryData(pageKey)).toBeUndefined()
    )
  })

  it("starts at Page 1 when the link changes to another binder", async () => {
    stubSharedApi({
      header: { status: 200, body: binderHeader },
      pages: [frontPage, backPage],
    })
    // The second link is answered by the same stub, under its own slug.
    const inner = globalThis.fetch
    const asked: string[] = []
    vi.stubGlobal("fetch", (input: RequestInfo | URL) => {
      const url = String(input)
      asked.push(new URL(url).pathname)
      return inner(url.replace("other-binder-77777", "holiday-binder-3f9a2"))
    })
    const router = await renderSharedPage()
    await screen.findByText("Page 1 of 2")
    fireEvent.click(screen.getByRole("button", { name: "Next page" }))
    await screen.findByText("Page 2 of 2")

    await router.history.push("/s/profile-ash/other-binder-77777")
    await router.load()

    await vi.waitFor(() =>
      expect(asked).toContain("/shared/profile-ash/other-binder-77777/pages/1")
    )
    expect(await screen.findByText("Page 1 of 2")).toBeTruthy()
  })

  it("shows no binder and asks for no Page for a plain Collection", async () => {
    const requests = stubSharedApi()
    await renderSharedPage()
    await screen.findByText("Pikachu")

    expect(screen.queryByRole("region", { name: "Binder page" })).toBeNull()
    expect(requests.some((r) => r.includes("/pages/"))).toBe(false)
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
