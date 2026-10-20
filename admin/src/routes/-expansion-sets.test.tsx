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
// check. This file tests the way from the shell home to the set screen; the
// form and its API calls have their own tests, so its hooks are stubbed here.
const api = vi.hoisted(() => ({ adminCheck: vi.fn() }))
// oxlint-disable-next-line anti-slop/no-module-mocking
vi.mock("@/generated/endpoints/admin/admin", () => ({
  getAdminCheckQueryOptions: () => ({
    queryKey: ["admin-check"],
    queryFn: api.adminCheck,
  }),
  useCreateExpansionSet: () => ({ mutate: vi.fn(), isPending: false }),
  useUpdateExpansionSet: () => ({ mutate: vi.fn(), isPending: false }),
}))
// oxlint-disable-next-line anti-slop/no-module-mocking
vi.mock("@/generated/endpoints/catalog/catalog", () => ({
  getListCatalogSeriesQueryKey: () => ["catalog-series"],
  useListCatalogSeries: () => ({ data: undefined }),
}))
// oxlint-disable-next-line anti-slop/no-module-mocking
vi.mock("@clerk/react", () =>
  import("@/test-clerk").then((m) => m.clerkModule())
)

beforeEach(() => {
  vi.stubEnv("VITE_CLERK_PUBLISHABLE_KEY", "pk_test_expansion_sets")
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

describe("the Expansion Set screen", () => {
  it("is reached from the shell home", async () => {
    const router = renderAdminAt("/")

    fireEvent.click(await screen.findByRole("link", { name: "Expansion Sets" }))

    await screen.findByRole("heading", { name: "Expansion Set" })
    expect(router.state.location.pathname).toBe("/expansion-sets")
  })
})
