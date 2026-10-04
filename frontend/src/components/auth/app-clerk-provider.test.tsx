import { afterEach, describe, expect, it, vi } from "vitest"
import { cleanup, render, screen } from "@testing-library/react"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import {
  RouterProvider,
  createMemoryHistory,
  createRootRoute,
  createRouter,
} from "@tanstack/react-router"
import { ClerkProvider } from "@clerk/react"
import { AppClerkProvider } from "./app-clerk-provider"

// The real provider loads Clerk's script from the network.
// oxlint-disable-next-line anti-slop/no-module-mocking
vi.mock("@clerk/react", () =>
  import("@/test-clerk").then((m) => m.clerkModule())
)

afterEach(() => {
  cleanup()
  vi.unstubAllEnvs()
  vi.clearAllMocks()
})

async function renderProvider() {
  const root = createRootRoute({
    component: () => (
      <AppClerkProvider>
        <p>the app</p>
      </AppClerkProvider>
    ),
  })
  const router = createRouter({
    routeTree: root,
    history: createMemoryHistory({ initialEntries: ["/"] }),
  })
  render(
    <QueryClientProvider client={new QueryClient()}>
      <RouterProvider router={router} />
    </QueryClientProvider>
  )
  await screen.findByText(/the app|Sign-in is not configured/)
  return router
}

describe("AppClerkProvider", () => {
  it("fails loudly instead of running the app unauthenticated when the key is missing", async () => {
    vi.stubEnv("VITE_CLERK_PUBLISHABLE_KEY", "")
    await renderProvider()

    expect(screen.getByRole("alert").textContent).toContain(
      "VITE_CLERK_PUBLISHABLE_KEY"
    )
    expect(screen.queryByText("the app")).toBeNull()
    expect(ClerkProvider).not.toHaveBeenCalled()
  })

  it("hands Clerk the publishable key, the auth addresses and the themed appearance", async () => {
    vi.stubEnv("VITE_CLERK_PUBLISHABLE_KEY", "pk_test_abc")
    await renderProvider()

    expect(screen.getByText("the app")).toBeTruthy()
    const props = vi.mocked(ClerkProvider).mock.calls[0]?.[0]
    expect(props).toMatchObject({
      publishableKey: "pk_test_abc",
      signInUrl: "/auth/login",
      signUpUrl: "/auth/register",
      afterSignOutUrl: "/",
    })
    // The app's CSS variables, so the `.dark` class themes Clerk too.
    expect(props?.appearance?.variables?.colorPrimary).toBe("var(--primary)")
    expect(props?.appearance?.variables?.colorBackground).toBe("var(--card)")
  })

  it("navigates Clerk's steps through the router, query string included", async () => {
    vi.stubEnv("VITE_CLERK_PUBLISHABLE_KEY", "pk_test_abc")
    const router = await renderProvider()
    const props = vi.mocked(ClerkProvider).mock.calls[0]?.[0]

    props?.routerPush?.("/auth/login/factor-one?redirect=%2Faccount")

    await vi.waitFor(() =>
      expect(router.history.location.href).toBe(
        "/auth/login/factor-one?redirect=%2Faccount"
      )
    )
  })
})
