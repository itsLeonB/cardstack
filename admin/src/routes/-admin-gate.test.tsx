import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"
import { cleanup, render, screen } from "@testing-library/react"
import { QueryClient } from "@tanstack/react-query"
import {
  RouterProvider,
  createMemoryHistory,
  createRouter,
} from "@tanstack/react-router"
import { CrashFallback } from "@/components/layout/crash-fallback"
import type { AuthGate } from "@/lib/clerk-auth"
import { routeTree } from "@/routeTree.gen"

// The real route tree in a memory router, with the generated client standing
// in for the API: `admin-check` answers with the status a test sets, and a
// Guest must never reach it (docs/adr/0019).
const api = vi.hoisted(() => ({ adminCheck: vi.fn() }))
// oxlint-disable-next-line anti-slop/no-module-mocking
vi.mock("@/generated/endpoints/admin/admin", () => ({
  getAdminCheckQueryOptions: () => ({
    queryKey: ["admin-check"],
    queryFn: api.adminCheck,
  }),
}))
// Clerk is faked at its edge: with no real instance the app's provider would
// render its "not configured" error instead of the routes under test.
// oxlint-disable-next-line anti-slop/no-module-mocking
vi.mock("@clerk/react", () =>
  import("@/test-clerk").then((m) => m.clerkModule())
)

beforeEach(() => {
  vi.stubEnv("VITE_CLERK_PUBLISHABLE_KEY", "pk_test_gate")
})

afterEach(() => {
  cleanup()
  vi.clearAllMocks()
  vi.unstubAllEnvs()
})

interface Visitor {
  signedIn: boolean
  checkStatus?: number
}

async function visit(who: Visitor) {
  if (who.checkStatus !== undefined) {
    api.adminCheck.mockResolvedValue({
      status: who.checkStatus,
      data: undefined,
      headers: new Headers(),
    })
  }
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  })
  const auth: AuthGate = { isSignedIn: async () => who.signedIn }
  const router = createRouter({
    routeTree,
    history: createMemoryHistory({ initialEntries: ["/"] }),
    context: { queryClient, auth },
    defaultErrorComponent: CrashFallback,
  })
  render(<RouterProvider router={router} />)
}

describe("the admin gate", () => {
  it("lets an Admin into the shell", async () => {
    await visit({ signedIn: true, checkStatus: 204 })

    await screen.findByRole("heading", { name: "Catalog admin" })
    expect(screen.getByRole("button", { name: "Sign out" })).toBeTruthy()
    expect(api.adminCheck).toHaveBeenCalledTimes(1)
  })

  it("shows the ordinary not-found page to a Guest, without asking the API", async () => {
    await visit({ signedIn: false })

    await screen.findByRole("heading", { name: "Page not found" })
    expect(api.adminCheck).not.toHaveBeenCalled()
  })

  it("shows the ordinary not-found page to a signed-in user who is not listed", async () => {
    await visit({ signedIn: true, checkStatus: 404 })

    await screen.findByRole("heading", { name: "Page not found" })
    expect(screen.queryByRole("heading", { name: "Catalog admin" })).toBeNull()
  })

  it.each([429, 500, 503])(
    "shows an error, not the not-found page, when the access check answers %i",
    async (status) => {
      await visit({ signedIn: true, checkStatus: status })

      await screen.findByRole("heading", { name: "Something went wrong" })
      expect(
        screen.queryByRole("heading", { name: "Page not found" })
      ).toBeNull()
    }
  )
})
