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
import { useUser } from "@clerk/react"
import { NotFound } from "@/components/layout/not-found"
import { getGetSharedWishlistQueryKey } from "@/generated/endpoints/shared/shared"
import { useGetProfile } from "@/generated/endpoints/profile/profile"
import { clerkUserResult } from "@/test-clerk"
import { catalogCard } from "@/test-binder-api"
import { Route as SharedRoute } from "./wishlist"
import type * as ProfileModule from "@/generated/endpoints/profile/profile"

// Network boundary: the page reads the caller's profile to decide whether the
// signed-in viewer is the owner, and Clerk's hooks need a real instance, so
// both are mocked at their edge.
// oxlint-disable-next-line anti-slop/no-module-mocking
vi.mock("@clerk/react", () =>
  import("@/test-clerk").then((m) => m.clerkModule())
)
// oxlint-disable-next-line anti-slop/no-module-mocking
vi.mock("@/generated/endpoints/profile/profile", async () => {
  const actual = await vi.importActual<typeof ProfileModule>(
    "@/generated/endpoints/profile/profile"
  )
  return { ...actual, useGetProfile: vi.fn() }
})

const mockUseUser = vi.mocked(useUser)
const mockUseGetProfile = vi.mocked(useGetProfile)

function session(state: "loading" | "guest" | "signed-in") {
  mockUseUser.mockReturnValue(clerkUserResult(state))
}

function profile(slug: string) {
  // SAFETY: partial hook result covering only the fields the page reads.
  mockUseGetProfile.mockReturnValue({
    data: { status: 200, data: { data: { slug } } },
    isPending: false,
    isError: false,
  } as any)
}

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
  /** The Slug the viewer's own profile resolves to. Defaults to a different
   * profile than the owner's, so a signed-in viewer is a non-owner by default. */
  viewerSlug?: string
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
    if (url.pathname === "/profile") {
      return reply(200, {
        data: { data: { slug: stub.viewerSlug ?? "profile-viewer" } },
      })
    }
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
// Defaults to a Guest (no signed-in viewer), so existing Guest-view tests keep
// working. Pass session "signed-in" and a viewerSlug to simulate a viewer.
async function renderSharedPage(
  path = PAGE,
  {
    clerkSession = "guest",
    viewerSlug,
  }: {
    clerkSession?: "loading" | "guest" | "signed-in"
    viewerSlug?: string
  } = {},
  queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  })
) {
  session(clerkSession)
  profile(viewerSlug ?? "profile-viewer")
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

  it("shows the viewer overlay and viewer filter for a signed-in non-owner viewer", async () => {
    stubSharedApi({
      entries: {
        status: 200,
        body: {
          data: [{ ...wanted, viewerOwnedQuantity: 2 }],
          meta: { total: 1, page: 1, limit: 100 },
        },
      },
    })
    await renderSharedPage(undefined, { clerkSession: "signed-in" })
    await screen.findByText("Pikachu")

    expect(await screen.findByText("You own 2")).toBeTruthy()
    // The viewer filter appears alongside the owner's status filter.
    for (const name of ["I have it", "I do not have it"]) {
      expect(screen.getByRole("button", { name })).toBeTruthy()
    }
    // No Guest banner.
    expect(screen.queryByText(/Sign in to see what you own/)).toBeNull()
  })

  it("shows no viewer overlay for the owner opening their own Shared link", async () => {
    stubSharedApi({
      entries: {
        status: 200,
        body: {
          data: [{ ...wanted, viewerOwnedQuantity: 2 }],
          meta: { total: 1, page: 1, limit: 100 },
        },
      },
    })
    // viewerSlug matches the path's profileSlug: the viewer is the owner.
    await renderSharedPage(undefined, {
      clerkSession: "signed-in",
      viewerSlug: "profile-ash",
    })
    await screen.findByText("Pikachu")

    expect(screen.queryByText("You own 2")).toBeNull()
    expect(screen.queryByRole("button", { name: "I have it" })).toBeNull()
    expect(
      screen.queryByRole("button", { name: "I do not have it" })
    ).toBeNull()
  })

  it("shows a sign-in banner and no overlay for a Guest", async () => {
    stubSharedApi({
      entries: {
        status: 200,
        body: {
          data: [{ ...wanted, viewerOwnedQuantity: 2 }],
          meta: { total: 1, page: 1, limit: 100 },
        },
      },
    })
    await renderSharedPage(undefined, { clerkSession: "guest" })
    await screen.findByText("Pikachu")

    expect(await screen.findByText(/Sign in to see what you own/)).toBeTruthy()
    expect(screen.queryByText("You own 2")).toBeNull()
    expect(screen.queryByRole("button", { name: "I have it" })).toBeNull()
  })

  it("does not send the viewer filter for a Guest", async () => {
    const requests = stubSharedApi({
      entries: {
        status: 200,
        body: { data: [wanted], meta: { total: 1, page: 1, limit: 100 } },
      },
    })
    await renderSharedPage(undefined, { clerkSession: "guest" })
    await screen.findByText("Pikachu")

    fireEvent.click(screen.getByRole("button", { name: "Still needed" }))
    await vi.waitFor(() =>
      expect(
        requests.some(
          (r) => r.startsWith(`${API}/entries?`) && r.includes("viewerFilter")
        )
      ).toBe(false)
    )
  })

  it("does not send or apply the viewer filter from the URL for a Guest", async () => {
    const requests = stubSharedApi({
      entries: {
        status: 200,
        body: { data: [wanted], meta: { total: 1, page: 1, limit: 100 } },
      },
    })
    await renderSharedPage("/s/profile-ash/wishlist?viewerFilter=have", {
      clerkSession: "guest",
    })
    await screen.findByText("Pikachu")

    // The viewer filter is not sent to the entries endpoint
    await vi.waitFor(() =>
      expect(
        requests.some(
          (r) => r.startsWith(API + "/entries?") && r.includes("viewerFilter")
        )
      ).toBe(false)
    )
    // The viewer filter buttons are not shown for a Guest
    expect(screen.queryByRole("button", { name: "I have it" })).toBeNull()
    expect(
      screen.queryByRole("button", { name: "I do not have it" })
    ).toBeNull()
  })
})
