import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"
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
import type { AuthGate } from "@/lib/clerk-auth"
import { NotFound } from "@/components/layout/not-found"
import { loadDraft, saveDraft } from "@/lib/draft-addition"
import { Route as ScanRoute } from "./scan"

// The real route options in a memory router, over a faked `fetch`. jsdom has no
// camera, so the screen falls back to the file picker.
let collectionStatus = 200
beforeEach(() => {
  collectionStatus = 200
  vi.stubGlobal(
    "fetch",
    vi.fn(async () =>
      collectionStatus === 200
        ? new Response(
            JSON.stringify({
              data: {
                id: "col-1",
                title: "Binder",
                description: "",
                maxCardCount: 0,
                cardCount: 0,
              },
            })
          )
        : new Response(JSON.stringify({ title: "not found" }), {
            status: collectionStatus,
          })
    )
  )
})
afterEach(() => {
  cleanup()
  localStorage.clear()
  vi.unstubAllGlobals()
  vi.unstubAllEnvs()
})

async function renderScan() {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  })
  const root = createRootRouteWithContext<{
    queryClient: QueryClient
    auth: AuthGate
  }>()({
    component: Outlet,
    notFoundComponent: NotFound,
  })
  const scan = createRoute({
    getParentRoute: () => root,
    path: "/collections/$collectionId/scan",
    ...pick(ScanRoute.options),
  })
  const router = createRouter({
    routeTree: root.addChildren([scan]),
    history: createMemoryHistory({
      initialEntries: ["/collections/col-1/scan"],
    }),
    context: { queryClient, auth: { isSignedIn: async () => true } },
  })
  render(
    <QueryClientProvider client={queryClient}>
      <RouterProvider router={router} />
    </QueryClientProvider>
  )
}

describe("the scan route", () => {
  it("is not found while the feature flag is off", async () => {
    await renderScan()
    await screen.findByRole("heading", { name: "Page not found" })
    expect(screen.queryByText("Choose a photo")).toBeNull()
  })

  it("shows the scan screen for a Collection when the flag is on", async () => {
    vi.stubEnv("VITE_SCAN_ENABLED", "true")
    await renderScan()
    await screen.findByRole("heading", { level: 1, name: "Scan cards" })
    expect(screen.getByText("Adding to Binder")).toBeTruthy()
    expect(screen.getByLabelText("Choose a photo")).toBeTruthy()
  })

  it("drops the draft quietly when the Collection is missing", async () => {
    vi.stubEnv("VITE_SCAN_ENABLED", "true")
    collectionStatus = 404
    saveDraft("col-1", [{ cardId: "card-1", quantity: 1 }])
    await renderScan()
    await screen.findByRole("heading", { name: "Collection not found" })
    expect(loadDraft("col-1")).toEqual([])
  })
})

/**
 * The route options under test, without the path the tree assigns.
 * SAFETY: `any` because the per-route generics of a file route can't be
 * spelled for a hand-built tree.
 */
function pick(options: {
  beforeLoad?: unknown
  loader?: unknown
  component?: unknown
}): any {
  const { beforeLoad, loader, component } = options
  return { beforeLoad, loader, component }
}
