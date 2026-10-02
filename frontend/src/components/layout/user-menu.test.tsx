import { afterEach, describe, expect, it, vi } from "vitest"
import {
  cleanup,
  fireEvent,
  render,
  screen,
  waitFor,
} from "@testing-library/react"
import { toast } from "sonner"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import {
  RouterProvider,
  createMemoryHistory,
  createRootRoute,
  createRoute,
  createRouter,
} from "@tanstack/react-router"
import type * as AuthModule from "@/generated/endpoints/auth/auth"
import { useGetCurrentUser } from "@/generated/endpoints/auth/auth"
import { ThemeProvider } from "@/components/theme-provider"
import { Toaster } from "@/components/ui/sonner"
import { UserMenu } from "./user-menu"

// Same session-probe boundary as site-header.test.tsx; logout itself goes
// through the real generated hook with `fetch` stubbed.
// oxlint-disable-next-line anti-slop/no-module-mocking
vi.mock("@/generated/endpoints/auth/auth", async () => {
  const actual = await vi.importActual<typeof AuthModule>(
    "@/generated/endpoints/auth/auth"
  )
  return { ...actual, useGetCurrentUser: vi.fn() }
})

afterEach(() => {
  cleanup()
  vi.restoreAllMocks()
  vi.unstubAllGlobals()
})

function stubFetch(respond: () => Response | Promise<Response>) {
  vi.stubGlobal(
    "fetch",
    vi.fn(async () => respond())
  )
  vi.stubGlobal("matchMedia", () => ({
    matches: false,
    addEventListener: () => {},
    removeEventListener: () => {},
  }))
}

async function openAndLogOut() {
  // SAFETY: partial mock covering only the fields useSession reads.
  vi.mocked(useGetCurrentUser).mockReturnValue({
    data: {
      status: 200,
      data: { data: { id: "1", email: "ada@example.com" } },
    },
    isPending: false,
  } as any)
  const root = createRootRoute({
    component: () => (
      <ThemeProvider>
        <UserMenu />
        <Toaster />
      </ThemeProvider>
    ),
  })
  const router = createRouter({
    routeTree: root.addChildren([
      createRoute({ getParentRoute: () => root, path: "/" }),
      createRoute({ getParentRoute: () => root, path: "/auth/login" }),
    ]),
    history: createMemoryHistory({ initialEntries: ["/"] }),
  })
  render(
    <QueryClientProvider client={new QueryClient()}>
      <RouterProvider router={router} />
    </QueryClientProvider>
  )
  fireEvent.click(await screen.findByRole("button", { name: "User menu" }))
  fireEvent.click(await screen.findByRole("menuitem", { name: "Log out" }))
  return router
}

describe("UserMenu logout", () => {
  it("shows a toast and stays put when the server rejects the logout", async () => {
    stubFetch(
      () => new Response(JSON.stringify({ detail: "nope" }), { status: 500 })
    )
    const router = await openAndLogOut()

    expect(
      await screen.findByText("Could not log out. You are still signed in.")
    ).toBeTruthy()
    expect(router.state.location.pathname).toBe("/")
  })

  it("shows a toast when the network request fails", async () => {
    stubFetch(() => Promise.reject(new TypeError("Failed to fetch")))
    await openAndLogOut()

    expect(
      await screen.findByText("Could not log out. You are still signed in.")
    ).toBeTruthy()
  })

  it("navigates to login without a toast on success", async () => {
    stubFetch(() => new Response(null, { status: 204 }))
    const error = vi.spyOn(toast, "error")
    const router = await openAndLogOut()

    await waitFor(() =>
      expect(router.state.location.pathname).toBe("/auth/login")
    )
    expect(error).not.toHaveBeenCalled()
  })
})
