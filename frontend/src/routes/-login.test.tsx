import { afterEach, describe, expect, it, vi } from "vitest"
import { cleanup, fireEvent, render, screen, waitFor } from "@testing-library/react"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import {
  RouterProvider,
  createMemoryHistory,
  createRootRouteWithContext,
  createRoute,
  createRouter,
} from "@tanstack/react-router"
import type * as AuthModule from "@/generated/endpoints/auth/auth"
import { useLogin } from "@/generated/endpoints/auth/auth"
import { Route as LoginRoute } from "./login"

// Network boundary: login is a generated orval hook with no service layer to
// inject (same approach as lib/session.test.tsx).
// oxlint-disable-next-line anti-slop/no-module-mocking
vi.mock("@/generated/endpoints/auth/auth", async () => {
  const actual =
    await vi.importActual<typeof AuthModule>("@/generated/endpoints/auth/auth")
  return { ...actual, useLogin: vi.fn() }
})

afterEach(cleanup)

// Mounts the real login route (its search validation and component) under a
// bare root, with a guest session seeded so the guest guard lets it through.
interface LoginResponse {
  status: number
  data: { data?: { csrfToken: string }; detail?: string }
}

async function renderLogin(url: string, loginStatus = 200) {
  const response: LoginResponse =
    loginStatus === 200
      ? { status: 200, data: { data: { csrfToken: "t" } } }
      : { status: 401, data: { detail: "Invalid credentials." } }
  // SAFETY: partial mutation result covering only what LoginPage reads.
  vi.mocked(useLogin).mockReturnValue({
    isPending: false,
    mutate: (_vars: { data: { email: string; password: string } }, options: { onSuccess: (r: LoginResponse) => void }) =>
      options.onSuccess(response),
  } as any)

  const queryClient = new QueryClient()
  queryClient.setQueryData(["/auth/me"], { status: 401, data: {} })
  const root = createRootRouteWithContext<{ queryClient: QueryClient }>()()
  const login = createRoute({
    getParentRoute: () => root,
    path: "/login",
    validateSearch: LoginRoute.options.validateSearch,
    beforeLoad: LoginRoute.options.beforeLoad,
    component: LoginRoute.options.component,
  })
  const router = createRouter({
    routeTree: root.addChildren([login]),
    history: createMemoryHistory({ initialEntries: [url] }),
    context: { queryClient },
  })
  const push = vi.spyOn(router.history, "push").mockImplementation(() => {})
  render(
    <QueryClientProvider client={queryClient}>
      <RouterProvider router={router} />
    </QueryClientProvider>
  )
  await screen.findByRole("heading", { level: 1, name: "Log in" })
  return { push }
}

function submit() {
  fireEvent.change(screen.getByLabelText("Email"), { target: { value: "ada@example.com" } })
  fireEvent.change(screen.getByLabelText("Password", { selector: "input" }), {
    target: { value: "correct horse" },
  })
  fireEvent.click(screen.getByRole("button", { name: "Log in" }))
}

describe("login redirect", () => {
  it("sends the user back to the attempted path, query string included", async () => {
    const { push } = await renderLogin("/login?redirect=%2Fcollections%3Fq%3Dbinder")
    submit()
    await waitFor(() => expect(push).toHaveBeenCalledWith("/collections?q=binder"))
  })

  it("falls back to /account without a redirect", async () => {
    const { push } = await renderLogin("/login")
    submit()
    await waitFor(() => expect(push).toHaveBeenCalledWith("/account"))
  })

  // The schema/predicate edge cases live in route-guard.test.ts; this only
  // checks that login navigation really falls back.
  it("drops an external redirect target and falls back to /account", async () => {
    const { push } = await renderLogin(
      `/login?redirect=${encodeURIComponent("https://evil.example/")}`
    )
    submit()
    await waitFor(() => expect(push).toHaveBeenCalledWith("/account"))
  })

  it("carries the redirect through the Register link", async () => {
    await renderLogin("/login?redirect=%2Fcollections%3Fq%3Dbinder")
    expect(screen.getByRole("link", { name: "Register" }).getAttribute("href")).toBe(
      "/register?redirect=%2Fcollections%3Fq%3Dbinder"
    )
  })
})

describe("login form", () => {
  it("shows the account-created notice", async () => {
    await renderLogin("/login?registered=true")
    expect(screen.getByRole("status").textContent).toBe("Account created. Log in below.")
  })

  it("shows field errors tied to their inputs without calling the API", async () => {
    const { push } = await renderLogin("/login")
    fireEvent.click(screen.getByRole("button", { name: "Log in" }))

    const email = screen.getByLabelText("Email")
    expect(email.getAttribute("aria-invalid")).toBe("true")
    const errorId = email.getAttribute("aria-describedby") ?? ""
    expect(document.getElementById(errorId)?.textContent).toBe("Enter a valid email address.")
    expect(push).not.toHaveBeenCalled()
  })

  it("shows a rejected login inline", async () => {
    await renderLogin("/login", 401)
    submit()
    expect((await screen.findByRole("alert")).textContent).toBe("Invalid credentials.")
  })
})
