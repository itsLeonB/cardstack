import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"
import { cleanup, fireEvent, render, screen } from "@testing-library/react"
import { QueryClient } from "@tanstack/react-query"
import {
  RouterProvider,
  createMemoryHistory,
  createRouter,
} from "@tanstack/react-router"
import { CrashFallback } from "@/components/layout/crash-fallback"
import type { AuthGate } from "@/lib/clerk-auth"
import { routeTree } from "@/routeTree.gen"

// The real route tree in a memory router, with an Admin let in by the access
// check. This file tests the way from the shell home to the Card screen; the
// forms and their API calls have their own tests, so their hooks are stubbed.
const api = vi.hoisted(() => ({ adminCheck: vi.fn() }))
// oxlint-disable-next-line anti-slop/no-module-mocking
vi.mock("@/generated/endpoints/admin/admin", () => ({
  getAdminCheckQueryOptions: () => ({
    queryKey: ["admin-check"],
    queryFn: api.adminCheck,
  }),
  getGetAdminCardQueryKey: () => ["admin-card"],
  useGetAdminCard: () => ({ data: undefined, isFetching: false }),
  useDeleteAdminCard: () => ({ mutate: vi.fn(), isPending: false }),
  createAdminCard: vi.fn(),
  updateAdminCard: vi.fn(),
}))
// oxlint-disable-next-line anti-slop/no-module-mocking
vi.mock("@/generated/endpoints/catalog/catalog", () => ({
  getListCatalogSeriesQueryKey: () => ["catalog-series"],
  getListCatalogSeriesQueryOptions: () => ({
    queryKey: ["catalog-series"],
    queryFn: async () => ({ status: 200, data: { data: {} } }),
  }),
  getListCatalogRaritiesQueryOptions: () => ({
    queryKey: ["catalog-rarities"],
    queryFn: async () => ({ status: 200, data: { data: [] } }),
  }),
  useListCatalogSeries: () => ({ data: undefined }),
}))
// oxlint-disable-next-line anti-slop/no-module-mocking
vi.mock("@clerk/react", () =>
  import("@/test-clerk").then((m) => m.clerkModule())
)

beforeEach(() => {
  vi.stubEnv("VITE_CLERK_PUBLISHABLE_KEY", "pk_test_cards")
  api.adminCheck.mockResolvedValue({
    status: 204,
    data: undefined,
    headers: new Headers(),
  })
})

afterEach(() => {
  cleanup()
  vi.clearAllMocks()
  vi.unstubAllEnvs()
})

function renderAdminAt(path: string) {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  })
  const auth: AuthGate = { isSignedIn: async () => true }
  const router = createRouter({
    routeTree,
    history: createMemoryHistory({ initialEntries: [path] }),
    context: { queryClient, auth },
    defaultErrorComponent: CrashFallback,
  })
  render(<RouterProvider router={router} />)
  return router
}

describe("the Card screen", () => {
  it("is reached from the shell home and holds the card and delete forms", async () => {
    const router = renderAdminAt("/")

    fireEvent.click(await screen.findByRole("link", { name: "Cards" }))

    await screen.findByRole("heading", { name: "Card" })
    screen.getByRole("heading", { name: "Delete a card" })
    expect(router.state.location.pathname).toBe("/cards")
  })
})
