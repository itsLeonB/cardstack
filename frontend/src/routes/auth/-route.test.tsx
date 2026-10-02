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
import { getGetCurrentUserQueryKey } from "@/generated/endpoints/auth/auth"
import { ThemeProvider } from "@/components/theme-provider"
import { Route as AuthRoute } from "./route"

afterEach(cleanup)

// jsdom has no matchMedia, which ThemeProvider reads.
vi.stubGlobal("matchMedia", () => ({
  matches: false,
  addEventListener: () => {},
  removeEventListener: () => {},
}))

// Mounts the real auth layout (guard + shell) over a stub child and a stub `/`.
async function renderAuth(session: { status: number; data: unknown }) {
  const queryClient = new QueryClient()
  queryClient.setQueryData(getGetCurrentUserQueryKey(), session)
  const root = createRootRouteWithContext<{ queryClient: QueryClient }>()({
    // ThemeProvider needs router context (ScriptOnce), so it sits inside the root.
    component: () => (
      <ThemeProvider>
        <Outlet />
      </ThemeProvider>
    ),
  })
  const layout = createRoute({
    getParentRoute: () => root,
    path: "/auth",
    beforeLoad: AuthRoute.options.beforeLoad,
    component: AuthRoute.options.component,
  })
  const child = createRoute({
    getParentRoute: () => layout,
    path: "/child",
    component: () => <h1>Child page</h1>,
  })
  const home = createRoute({ getParentRoute: () => root, path: "/", component: () => <h1>Home</h1> })
  const router = createRouter({
    routeTree: root.addChildren([layout.addChildren([child]), home]),
    history: createMemoryHistory({ initialEntries: ["/auth/child"] }),
    context: { queryClient },
  })
  render(
    <QueryClientProvider client={queryClient}>
      <RouterProvider router={router} />
    </QueryClientProvider>
  )
  return router
}

describe("auth layout", () => {
  it("renders a minimal shell: wordmark and theme toggle, one main, no nav or footer", async () => {
    await renderAuth({ status: 401, data: {} })
    await screen.findByRole("heading", { level: 1, name: "Child page" }, { timeout: 5000 })

    expect(screen.getByRole("link", { name: "Cardstack" }).getAttribute("href")).toBe("/")
    expect(screen.getByRole("button", { name: "Theme" })).toBeTruthy()
    expect(screen.getAllByRole("main")).toHaveLength(1)
    expect(screen.queryByRole("navigation")).toBeNull()
    expect(screen.queryByRole("contentinfo")).toBeNull()
  })

  it("sends a signed-in user to /", async () => {
    const router = await renderAuth({ status: 200, data: { data: { id: "1", email: "a@b.com" } } })
    await screen.findByRole("heading", { level: 1, name: "Home" }, { timeout: 5000 })
    expect(router.state.location.pathname).toBe("/")
  })
})
