import { afterEach, describe, expect, it, vi } from "vitest"
import { cleanup, render, screen, within } from "@testing-library/react"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import {
  Outlet,
  RouterProvider,
  createMemoryHistory,
  createRootRoute,
  createRoute,
  createRouter,
} from "@tanstack/react-router"
import type * as AuthModule from "@/generated/endpoints/auth/auth"
import { useGetCurrentUser } from "@/generated/endpoints/auth/auth"
import { ThemeProvider } from "@/components/theme-provider"
import { SiteFooter } from "./site-footer"
import { SiteHeader } from "./site-header"

// Network boundary: the session probe is a generated orval hook with no
// service layer to inject (same approach as lib/session.test.tsx).
// oxlint-disable-next-line anti-slop/no-module-mocking
vi.mock("@/generated/endpoints/auth/auth", async () => {
  const actual = await vi.importActual<typeof AuthModule>(
    "@/generated/endpoints/auth/auth"
  )
  return { ...actual, useGetCurrentUser: vi.fn() }
})

const mockUseGetCurrentUser = vi.mocked(useGetCurrentUser)

afterEach(cleanup)

// jsdom has no matchMedia, which ThemeProvider reads.
vi.stubGlobal("matchMedia", () => ({
  matches: false,
  addEventListener: () => {},
  removeEventListener: () => {},
}))

function mockSession(session: "guest" | "signed-in" | "loading") {
  // SAFETY: partial mock covering only the fields useSession reads.
  mockUseGetCurrentUser.mockReturnValue(
    session === "loading"
      ? ({ data: undefined, isPending: true } as any)
      : session === "guest"
        ? ({ data: { status: 401 }, isPending: false } as any)
        : ({
            data: {
              status: 200,
              data: { data: { id: "1", email: "ada@example.com" } },
            },
            isPending: false,
          } as any)
  )
}

async function renderShell(path = "/") {
  const root = createRootRoute({
    component: () => (
      <ThemeProvider>
        <SiteHeader />
        <Outlet />
        <SiteFooter />
      </ThemeProvider>
    ),
  })
  const routeTree = root.addChildren(
    [
      "/",
      "/catalog",
      "/collections",
      "/inventory",
      "/auth/login",
      "/auth/register",
      "/account",
    ].map((p) => createRoute({ getParentRoute: () => root, path: p }))
  )
  const router = createRouter({
    routeTree,
    history: createMemoryHistory({ initialEntries: [path] }),
  })
  render(
    <QueryClientProvider client={new QueryClient()}>
      <RouterProvider router={router} />
    </QueryClientProvider>
  )
  return screen.findByRole("banner")
}

function linkNames(nav: HTMLElement) {
  return within(nav)
    .getAllByRole("link")
    .map((link) => link.textContent)
}

describe("SiteHeader", () => {
  it("shows Catalog, Log in and Register to guests", async () => {
    mockSession("guest")
    const header = await renderShell()

    expect(linkNames(within(header).getByRole("navigation"))).toEqual([
      "Catalog",
      "Log in",
      "Register",
    ])
    expect(
      within(header).queryByRole("button", { name: "User menu" })
    ).toBeNull()
  })

  it("shows Catalog, Collections and Master Inventory plus a user menu when signed in", async () => {
    mockSession("signed-in")
    const header = await renderShell()

    expect(linkNames(within(header).getByRole("navigation"))).toEqual([
      "Catalog",
      "Collections",
      "Master Inventory",
    ])
    expect(
      within(header).getByRole("button", { name: "User menu" })
    ).toBeTruthy()
  })

  it("shows only Catalog while the session is loading", async () => {
    mockSession("loading")
    const header = await renderShell()

    expect(linkNames(within(header).getByRole("navigation"))).toEqual([
      "Catalog",
    ])
  })

  it("marks the current section as the current page", async () => {
    mockSession("signed-in")
    const header = await renderShell("/collections")

    const current = within(header).getByRole("link", { name: "Collections" })
    expect(current.getAttribute("aria-current")).toBe("page")
    expect(
      within(header)
        .getByRole("link", { name: "Catalog" })
        .getAttribute("aria-current")
    ).toBeNull()
  })

  it("renders the footer description without repeating nav links", async () => {
    mockSession("guest")
    await renderShell()

    const footer = screen.getByRole("contentinfo")
    expect(within(footer).queryByRole("navigation")).toBeNull()
    expect(footer.textContent).toContain("personal MVP")
  })
})
