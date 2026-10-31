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
// check. This file tests the way from the shell home to the review queue; the
// queue and its photos have their own tests, so their hooks are stubbed here.
const api = vi.hoisted(() => ({ adminCheck: vi.fn() }))
// oxlint-disable-next-line anti-slop/no-module-mocking
vi.mock("@/generated/endpoints/admin/admin", () => ({
  getAdminCheckQueryOptions: () => ({
    queryKey: ["admin-check"],
    queryFn: api.adminCheck,
  }),
  getListReviewQueueQueryKey: () => ["review-queue"],
  useListReviewQueue: () => ({
    isPending: false,
    isError: false,
    data: {
      status: 200,
      data: { data: [], meta: { total: 0, page: 1, limit: 24 } },
    },
  }),
  useGetReviewPhoto: () => ({ data: undefined, refetch: vi.fn() }),
  useRejectReview: () => ({ mutate: vi.fn(), isPending: false }),
}))
// oxlint-disable-next-line anti-slop/no-module-mocking
vi.mock("@clerk/react", () =>
  import("@/test-clerk").then((m) => m.clerkModule())
)

beforeEach(() => {
  vi.stubEnv("VITE_CLERK_PUBLISHABLE_KEY", "pk_test_reviews")
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

describe("the review screen", () => {
  it("is reached from the shell home and shows the queue", async () => {
    const router = renderAdminAt("/")

    fireEvent.click(
      await screen.findByRole("link", { name: "Corrections to review" })
    )

    await screen.findByRole("heading", { name: "Corrections to review" })
    await screen.findByText("Nothing is waiting for review.")
    expect(router.state.location.pathname).toBe("/reviews")
  })
})
