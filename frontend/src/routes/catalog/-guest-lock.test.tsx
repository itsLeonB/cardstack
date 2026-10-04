import { afterEach, describe, expect, it, vi } from "vitest"
import { cleanup, render, screen } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
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
import { toast } from "sonner"
import { setTokenGetter } from "@/lib/http"
import { redirectSearchSchema } from "@/lib/route-guard"
import type { AuthGate } from "@/lib/clerk-auth"
import { stubGridLayout } from "@/test-grid-layout"
import { clerkUserResult } from "@/test-clerk"
import { Route as SearchRoute } from "./search"
import { Route as SetRoute } from "./sets/$expansionSetId"

// The catalog as a Guest and as a signed-in caller meet it: the real route
// files in a memory router, Clerk's hooks faked at their edge, and `fetch`
// standing in for the API with its guest lock (ticket 10): one page of at most
// 24, 401 `login_required` for a later page, a rarity, category or tag filter,
// or the facets. Page and request assertions read the request log.
// oxlint-disable-next-line anti-slop/no-module-mocking
vi.mock("@clerk/react", () =>
  import("@/test-clerk").then((m) => m.clerkModule())
)
// oxlint-disable-next-line anti-slop/no-module-mocking
vi.mock("sonner", () => ({ toast: { error: vi.fn(), success: vi.fn() } }))

afterEach(() => {
  cleanup()
  setTokenGetter(null)
  vi.restoreAllMocks()
  vi.unstubAllGlobals()
})

const TOTAL = 100
const GUEST_PAGE = 24

const expansionSet = {
  id: "s1",
  code: "SV1",
  name: "Scarlet Base",
  imageUrl: "",
}
const cardOf = (n: number) => ({
  id: `card-${n}`,
  name: `Card ${n}`,
  localId: String(n),
  category: "Pokémon",
  tags: [],
  illustrator: "someone",
  imageUrl: "",
  expansionSet,
  rarity: { id: "r1", code: "C", name: "Common" },
})
const facets = {
  expansionSets: [{ ...expansionSet, seriesId: "sr1", available: true }],
  rarities: [{ id: "r1", code: "C", name: "Common", available: true }],
  categories: [{ value: "Pokémon", available: true }],
  tags: [],
}
const series = {
  series: [
    { id: "sr1", code: "SV", name: "Scarlet", expansionSets: [expansionSet] },
  ],
  ungroupedExpansionSets: [],
}

/** The envelope and error fields the stubbed API answers with. */
interface StubBody {
  data?: unknown
  meta?: { total: number; page: number; limit: number }
  status?: number
  code?: string
  detail?: string
}

const json = (status: number, body: StubBody) =>
  new Response(JSON.stringify(body), {
    status,
    headers: { "content-type": "application/json" },
  })
const loginRequired = () =>
  json(401, {
    status: 401,
    code: "login_required",
    detail: "sign in to use this part of the catalog",
  })

function stubApi() {
  const requests: URL[] = []
  vi.stubGlobal(
    "fetch",
    vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
      const url = new URL(String(input))
      requests.push(url)
      const guest = !new Headers(init?.headers).has("Authorization")
      switch (url.pathname) {
        case "/catalog/series":
          return json(200, { data: series })
        case "/catalog/facets":
          return guest ? loginRequired() : json(200, { data: facets })
        case "/collections":
          return guest ? loginRequired() : json(200, { data: [] })
        case "/catalog/cards": {
          const page = Number(url.searchParams.get("page") ?? 1)
          const asked = Number(url.searchParams.get("limit") ?? GUEST_PAGE)
          const locked = ["rarityId", "category", "tag"].some((key) =>
            url.searchParams.has(key)
          )
          if (guest && (page > 1 || locked)) return loginRequired()
          const limit = guest ? Math.min(asked, GUEST_PAGE) : asked
          const first = (page - 1) * limit
          const count = Math.max(0, Math.min(limit, TOTAL - first))
          return json(200, {
            data: Array.from({ length: count }, (_, i) =>
              cardOf(first + i + 1)
            ),
            meta: { total: TOTAL, page, limit },
          })
        }
        default:
          return json(404, { detail: "not found" })
      }
    })
  )
  return requests
}

const cardRequests = (requests: URL[]) =>
  requests.filter((url) => url.pathname === "/catalog/cards")
const pagesAsked = (requests: URL[]) =>
  cardRequests(requests).map((url) => Number(url.searchParams.get("page")))

async function visit(url: string, { signedIn }: { signedIn: boolean }) {
  vi.stubGlobal("matchMedia", () => ({
    matches: false,
    addEventListener: () => {},
    removeEventListener: () => {},
  }))
  stubGridLayout()
  const requests = stubApi()
  vi.mocked(useUser).mockReturnValue(
    clerkUserResult(signedIn ? "signed-in" : "guest")
  )
  setTokenGetter(
    signedIn ? async () => ({ token: "t", sessionId: "s_1" }) : null
  )

  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  })
  const auth: AuthGate = { isSignedIn: async () => signedIn }
  const root = createRootRouteWithContext<{
    queryClient: QueryClient
    auth: AuthGate
  }>()({ component: () => <Outlet /> })
  const routeTree = root.addChildren([
    createRoute({
      getParentRoute: () => root,
      path: "/catalog/search",
      ...pick(SearchRoute.options),
    }),
    createRoute({
      getParentRoute: () => root,
      path: "/catalog/sets/$expansionSetId",
      ...pick(SetRoute.options),
    }),
    createRoute({
      getParentRoute: () => root,
      path: "/auth/login",
      validateSearch: redirectSearchSchema,
      component: () => <h1>Sign in page</h1>,
    }),
    createRoute({
      getParentRoute: () => root,
      path: "/catalog",
      component: () => <h1>Catalog</h1>,
    }),
  ])
  const router = createRouter({
    routeTree,
    history: createMemoryHistory({ initialEntries: [url] }),
    context: { queryClient, auth },
  })
  render(
    <QueryClientProvider client={queryClient}>
      <RouterProvider router={router} />
    </QueryClientProvider>
  )
  return { router, requests }
}

/**
 * The route options a page contributes, without the path the tree assigns.
 * SAFETY: `any` because the per-route generics of a file route can't be
 * spelled for a hand-built tree; these are the options under test.
 */
function pick(options: {
  validateSearch?: unknown
  loaderDeps?: unknown
  loader?: unknown
  component?: unknown
}): any {
  const { validateSearch, loaderDeps, loader, component } = options
  return { validateSearch, loaderDeps, loader, component }
}

const isDisabled = (name: RegExp | string) =>
  screen.getByRole("button", { name }).hasAttribute("disabled")

describe("catalog search as a Guest", () => {
  it("locks rarity, category and tag with a prompt, and keeps name search and Expansion Set open", async () => {
    await visit("/catalog/search?name=Card", { signedIn: false })

    await screen.findByText("24 of 100 cards loaded")
    expect(isDisabled("Rarity")).toBe(true)
    expect(isDisabled("Category")).toBe(true)
    expect(isDisabled("Tag")).toBe(true)
    expect(isDisabled("Expansion Set")).toBe(false)
    screen.getByRole("link", { name: "Sign in to use filters" })

    // The Expansion Set dropdown is filled from the Series list, since facets are locked.
    await userEvent.click(screen.getByRole("button", { name: "Expansion Set" }))
    screen.getByRole("checkbox", { name: "Scarlet Base (SV1)" })
  })

  it("ends the first page with 'Sign in to see more' and never asks for page 2", async () => {
    const { requests } = await visit("/catalog/search?name=Card", {
      signedIn: false,
    })

    await screen.findByRole("link", { name: "Sign in to see more" })
    expect(screen.queryByRole("button", { name: "Load more" })).toBeNull()
    expect(pagesAsked(requests)).toEqual([1])
  })

  it("brings the user back to the same search after signing in, from every prompt", async () => {
    const { router } = await visit(
      "/catalog/search?name=Card&expansionSetId=s1",
      { signedIn: false }
    )
    const more = await screen.findByRole("link", {
      name: "Sign in to see more",
    })
    // The router's own address for this view (it writes arrays in its canonical form).
    const back = router.state.location.href
    expect(back).toContain("name=Card")
    const href = `/auth/login?redirect=${encodeURIComponent(back)}`
    expect(more.getAttribute("href")).toBe(href)
    expect(
      screen
        .getByRole("link", { name: "Sign in to use filters" })
        .getAttribute("href")
    ).toBe(href)
    expect(
      screen
        .getByRole("link", { name: "Sign in to add cards to a Collection" })
        .getAttribute("href")
    ).toBe(href)

    await userEvent.click(more)
    await screen.findByRole("heading", { name: "Sign in page" })
    // The login page's own same-origin rule accepted the target.
    expect(router.state.location.search).toEqual({ redirect: back })
  })

  it("shows a sign-in prompt, never an error or toast, for a locked filter in the address", async () => {
    await visit("/catalog/search?rarityId=r1", { signedIn: false })

    await screen.findByRole("link", { name: "Sign in" })
    expect(screen.queryByRole("alert")).toBeNull()
    expect(vi.mocked(toast.error)).not.toHaveBeenCalled()
    // The chip stays, so the user can still remove the filter.
    expect(screen.getByRole("button", { name: /^Remove / })).toBeTruthy()
  })

  it("keeps the card count live and the prompt reachable by keyboard", async () => {
    await visit("/catalog/search?name=Card", { signedIn: false })

    const status = await screen.findByText("24 of 100 cards loaded")
    expect(status.getAttribute("aria-live")).toBe("polite")
    const link = screen.getByRole("link", { name: "Sign in to see more" })
    link.focus()
    expect(document.activeElement).toBe(link)
    expect(link.closest("[aria-live=polite]")).not.toBeNull()
  })
})

describe("catalog search signed in", () => {
  it("has every filter, no prompts, and a Load more that fetches page 2", async () => {
    const { requests } = await visit("/catalog/search?name=Card", {
      signedIn: true,
    })

    await screen.findByText("60 of 100 cards loaded")
    expect(isDisabled("Rarity")).toBe(false)
    expect(isDisabled("Category")).toBe(false)
    expect(screen.queryByRole("link", { name: /^Sign in/ })).toBeNull()
    expect(requests.some((url) => url.pathname === "/catalog/facets")).toBe(
      true
    )

    await userEvent.click(screen.getByRole("button", { name: "Load more" }))
    await screen.findByText("100 of 100 cards loaded")
    expect(pagesAsked(requests)).toContain(2)
  })
})

describe("Expansion Set page", () => {
  it("shows a Guest 'Sign in to see more' and returns to that set", async () => {
    const { requests } = await visit("/catalog/sets/s1", { signedIn: false })

    const more = await screen.findByRole("link", {
      name: "Sign in to see more",
    })
    expect(more.getAttribute("href")).toBe(
      `/auth/login?redirect=${encodeURIComponent("/catalog/sets/s1")}`
    )
    expect(pagesAsked(requests)).toEqual([1])
  })

  it("lets a signed-in user load the next page", async () => {
    await visit("/catalog/sets/s1", { signedIn: true })

    await screen.findByRole("button", { name: "Load more" })
    expect(screen.queryByRole("link", { name: /^Sign in/ })).toBeNull()
  })
})
