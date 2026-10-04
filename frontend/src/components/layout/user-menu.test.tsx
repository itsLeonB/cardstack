import { afterEach, describe, expect, it, vi } from "vitest"
import { cleanup, fireEvent, render, screen } from "@testing-library/react"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import {
  RouterProvider,
  createMemoryHistory,
  createRootRoute,
  createRoute,
  createRouter,
} from "@tanstack/react-router"
import { toast } from "sonner"
import { useClerk, useUser } from "@clerk/react"
import { ThemeProvider } from "@/components/theme-provider"
import { Toaster } from "@/components/ui/sonner"
import { clerkUserResult } from "@/test-clerk"
import type { ClerkUserData } from "@/test-clerk"
import { UserMenu } from "./user-menu"

// Clerk's hooks need a ClerkProvider talking to Clerk's servers, so they are
// the boundary to fake; the real `useSession` and `useSignOut` run on top.
// oxlint-disable-next-line anti-slop/no-module-mocking
vi.mock("@clerk/react", () =>
  import("@/test-clerk").then((m) => m.clerkModule())
)

afterEach(() => {
  cleanup()
  vi.restoreAllMocks()
  vi.unstubAllGlobals()
})

async function renderMenu(
  signOut: () => Promise<void> = async () => {},
  user?: ClerkUserData
) {
  vi.stubGlobal("matchMedia", () => ({
    matches: false,
    addEventListener: () => {},
    removeEventListener: () => {},
  }))
  vi.mocked(useUser).mockReturnValue(clerkUserResult("signed-in", user))
  // SAFETY: partial hook result covering only `signOut`.
  vi.mocked(useClerk).mockReturnValue({ signOut } as any)
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
      createRoute({ getParentRoute: () => root, path: "/account" }),
    ]),
    history: createMemoryHistory({ initialEntries: ["/"] }),
  })
  render(
    <QueryClientProvider client={new QueryClient()}>
      <RouterProvider router={router} />
    </QueryClientProvider>
  )
  fireEvent.click(await screen.findByRole("button", { name: "User menu" }))
}

describe("UserMenu identity", () => {
  it("shows the name and email from Clerk's user data", async () => {
    await renderMenu()

    await screen.findByRole("menuitem", { name: "Account" })
    expect(screen.getAllByText("Ada Lovelace").length).toBeGreaterThan(0)
    expect(screen.getByText("ada@example.com")).toBeTruthy()
  })

  it("falls back to the email alone when the account has no name", async () => {
    await renderMenu(undefined, {
      id: "user_2",
      fullName: null,
      primaryEmailAddress: { emailAddress: "grace@example.com" },
    })

    await screen.findByRole("menuitem", { name: "Account" })
    expect(screen.getAllByText("grace@example.com").length).toBeGreaterThan(0)
    expect(screen.queryByText("Ada Lovelace")).toBeNull()
  })
})

describe("UserMenu sign-out", () => {
  it("ends the Clerk session and leaves for the login page, without a toast", async () => {
    const signOut = vi.fn(async () => {})
    const error = vi.spyOn(toast, "error")
    await renderMenu(signOut)

    fireEvent.click(await screen.findByRole("menuitem", { name: "Log out" }))

    await vi.waitFor(() =>
      expect(signOut).toHaveBeenCalledWith({ redirectUrl: "/auth/login" })
    )
    expect(error).not.toHaveBeenCalled()
  })

  it("shows a toast when Clerk fails to sign out", async () => {
    await renderMenu(async () => {
      throw new Error("Failed to fetch")
    })

    fireEvent.click(await screen.findByRole("menuitem", { name: "Log out" }))

    expect(
      await screen.findByText("Could not log out. You are still signed in.")
    ).toBeTruthy()
  })
})
