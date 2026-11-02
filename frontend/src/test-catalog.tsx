import { vi } from "vitest"
import { render } from "@testing-library/react"
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
import { setTokenGetter } from "@/lib/http"
import { redirectSearchSchema } from "@/lib/route-guard"
import type { AuthGate } from "@/lib/clerk-auth"
import type { WishlistItem } from "@/generated/models"
import { stubGridLayout } from "@/test-grid-layout"
import { clerkUserResult } from "@/test-clerk"
import { Route as SearchRoute } from "@/routes/catalog/search"
import { Route as SetRoute } from "@/routes/catalog/sets/$expansionSetId"
import { Route as CardRoute } from "@/routes/catalog/cards.$expansionSetId.$localId"

// The catalog's real route files in a memory router, Clerk's hooks faked at
// their edge, and `fetch` standing in for the API with its guest lock (ticket
// 10): one page of at most 24, 401 `login_required` for a later page, a rarity,
// category or tag filter, or the facets. The stubbed Wishlist answers the
// wanted-quantity lookups and records every write, for "Add to Wishlist" from
// the catalog (ticket 8). Page and request assertions read the request log.
// A test file using this helper mocks `@clerk/react` from `@/test-clerk` and
// `sonner` itself, and clears the token getter after each test.

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
    {
      id: "sr1",
      code: "SV",
      name: "Scarlet",
      imageUrl: "",
      expansionSets: [expansionSet],
    },
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

/** One Card's row from the stubbed Wishlist. */
const wishlistItemOf = (cardId: string, quantity: number): WishlistItem => ({
  card: cardOf(Number(cardId.replace("card-", ""))),
  wantedQuantity: quantity,
  ownedQuantity: 0,
  fulfilled: false,
})

/** A write to the stubbed Wishlist, as the API received it. */
export interface WishlistWrite {
  method: string
  path: string
  body?: unknown
}

/**
 * The API, with a Wishlist of `cardId` → wanted quantity that a write changes,
 * so a test can watch a quantity be added and then replaced.
 */
function stubApi(wishlist = new Map<string, number>()) {
  const requests: URL[] = []
  const writes: WishlistWrite[] = []
  vi.stubGlobal(
    "fetch",
    vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
      const url = new URL(String(input))
      requests.push(url)
      const guest = !new Headers(init?.headers).has("Authorization")
      const method = init?.method ?? "GET"

      if (url.pathname === "/wishlist") {
        if (guest) return loginRequired()
        const ids = url.searchParams.getAll("cardId")
        const data: WishlistItem[] = []
        for (const id of ids) {
          const quantity = wishlist.get(id)
          if (quantity !== undefined) data.push(wishlistItemOf(id, quantity))
        }
        return json(200, {
          data,
          meta: { total: data.length, page: 1, limit: ids.length },
        })
      }

      const written = /^\/wishlist\/items\/(.+)$/.exec(url.pathname)
      if (written) {
        const cardId = written[1]
        if (!cardId) return json(404, { detail: "not found" })
        const body = init?.body ? JSON.parse(String(init.body)) : undefined
        writes.push({ method, path: url.pathname, body })
        if (guest) return loginRequired()
        if (method === "PUT") wishlist.set(cardId, body.quantity)
        else wishlist.delete(cardId)
        return new Response(null, { status: 204 })
      }

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
  return { requests, writes }
}

const cardRequests = (requests: URL[]) =>
  requests.filter((url) => url.pathname === "/catalog/cards")
export const pagesAsked = (requests: URL[]) =>
  cardRequests(requests).map((url) => Number(url.searchParams.get("page")))
export const wishlistLookups = (requests: URL[]) =>
  requests.filter((url) => url.pathname === "/wishlist")

/** Renders one catalog route in a fresh cache, signed in or as a Guest. */
export async function visitCatalog(
  url: string,
  { signedIn, wishlist }: { signedIn: boolean; wishlist?: Map<string, number> }
) {
  vi.stubGlobal("matchMedia", () => ({
    matches: false,
    addEventListener: () => {},
    removeEventListener: () => {},
  }))
  stubGridLayout()
  const { requests, writes } = stubApi(wishlist)
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
      path: "/catalog/cards/$expansionSetId/$localId",
      ...pick(CardRoute.options),
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
  return { router, requests, writes }
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
