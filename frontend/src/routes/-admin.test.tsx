import { afterEach, describe, expect, it, vi } from "vitest"
import { cleanup, render, screen } from "@testing-library/react"
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
import { setTokenGetter } from "@/lib/http"
import type { AuthGate } from "@/lib/clerk-auth"
import { clerkUserResult } from "@/test-clerk"
import { Route as AdminRoute } from "./admin"

// The /admin page as a Guest, a signed-in non-Admin and an Admin meet it: the
// real route file in a memory router, Clerk's hooks faked at their edge, and
// `fetch` standing in for the API, which answers 204 to an Admin and 404 to
// everyone else (docs/adr/0019).
// oxlint-disable-next-line anti-slop/no-module-mocking
vi.mock("@clerk/react", () =>
  import("@/test-clerk").then((m) => m.clerkModule())
)

afterEach(() => {
  cleanup()
  setTokenGetter(null)
  vi.restoreAllMocks()
  vi.unstubAllGlobals()
})

async function visit(who: "guest" | "non-admin" | "admin") {
  const signedIn = who !== "guest"
  const requests: string[] = []
  vi.stubGlobal(
    "fetch",
    vi.fn(async (input: RequestInfo | URL) => {
      requests.push(new URL(String(input)).pathname)
      return who === "admin"
        ? new Response(null, { status: 204 })
        : new Response(JSON.stringify({ status: 404, detail: "Not Found" }), {
            status: 404,
            headers: { "content-type": "application/json" },
          })
    })
  )
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
  }>()({ notFoundComponent: NotFound, component: () => <Outlet /> })
  const { beforeLoad, component } = AdminRoute.options
  const routeTree = root.addChildren([
    createRoute({
      getParentRoute: () => root,
      path: "/admin",
      beforeLoad,
      component,
    }),
  ])
  const router = createRouter({
    routeTree,
    history: createMemoryHistory({ initialEntries: ["/admin"] }),
    context: { queryClient, auth },
  })
  render(
    <QueryClientProvider client={queryClient}>
      <RouterProvider router={router} />
    </QueryClientProvider>
  )
  return requests
}

describe("the /admin page", () => {
  it("shows the ordinary not-found page to a signed-in non-Admin", async () => {
    await visit("non-admin")

    await screen.findByRole("heading", { name: "Page not found" })
    expect(screen.queryByRole("heading", { name: "Catalog admin" })).toBeNull()
  })

  it("shows the ordinary not-found page to a Guest, without asking the API", async () => {
    const requests = await visit("guest")

    await screen.findByRole("heading", { name: "Page not found" })
    expect(requests).toEqual([])
  })

  it("shows the admin page to an Admin", async () => {
    await visit("admin")

    await screen.findByRole("heading", { name: "Catalog admin" })
    expect(screen.queryByRole("heading", { name: "Page not found" })).toBeNull()
  })
})
