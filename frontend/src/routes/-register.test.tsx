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
import { useRegister } from "@/generated/endpoints/auth/auth"
import { Route as LoginRoute } from "./login"
import { Route as RegisterRoute } from "./register"

// Network boundary: register is a generated orval hook with no service layer
// to inject (same approach as -login.test.tsx).
// oxlint-disable-next-line anti-slop/no-module-mocking
vi.mock("@/generated/endpoints/auth/auth", async () => {
  const actual =
    await vi.importActual<typeof AuthModule>("@/generated/endpoints/auth/auth")
  return { ...actual, useRegister: vi.fn() }
})

afterEach(cleanup)

// Mounts the real register route plus a stub /login that renders its search,
// so the post-register navigation can be read back off the router.
async function renderRegister(url: string) {
  // SAFETY: partial mutation result covering only what RegisterPage reads.
  vi.mocked(useRegister).mockReturnValue({
    isPending: false,
    mutate: (_vars: { data: { email: string; password: string } }, options: { onSuccess: (r: { status: number }) => void }) =>
      options.onSuccess({ status: 201 }),
  } as any)

  const queryClient = new QueryClient()
  queryClient.setQueryData(["/auth/me"], { status: 401, data: {} })
  const root = createRootRouteWithContext<{ queryClient: QueryClient }>()()
  const register = createRoute({
    getParentRoute: () => root,
    path: "/register",
    validateSearch: RegisterRoute.options.validateSearch,
    beforeLoad: RegisterRoute.options.beforeLoad,
    component: RegisterRoute.options.component,
  })
  const login = createRoute({
    getParentRoute: () => root,
    path: "/login",
    validateSearch: LoginRoute.options.validateSearch,
    component: () => <h1>Login stub</h1>,
  })
  const router = createRouter({
    routeTree: root.addChildren([register, login]),
    history: createMemoryHistory({ initialEntries: [url] }),
    context: { queryClient },
  })
  render(
    <QueryClientProvider client={queryClient}>
      <RouterProvider router={router} />
    </QueryClientProvider>
  )
  await screen.findByRole("heading", { level: 1, name: "Create an account" })
  return router
}

describe("register redirect", () => {
  it("carries the redirect through the Log in link", async () => {
    await renderRegister("/register?redirect=%2Fcollections%3Fq%3Dbinder")
    expect(screen.getByRole("link", { name: "Log in" }).getAttribute("href")).toBe(
      "/login?redirect=%2Fcollections%3Fq%3Dbinder"
    )
  })

  it("drops an external redirect from the Log in link", async () => {
    await renderRegister(`/register?redirect=${encodeURIComponent("https://evil.example/")}`)
    expect(screen.getByRole("link", { name: "Log in" }).getAttribute("href")).toBe("/login")
  })

  it("passes the redirect to login after a successful registration", async () => {
    const router = await renderRegister("/register?redirect=%2Fcollections%3Fq%3Dbinder")
    fireEvent.change(screen.getByLabelText("Email"), { target: { value: "ada@example.com" } })
    fireEvent.change(screen.getByLabelText("Password", { selector: "input" }), {
      target: { value: "correct horse" },
    })
    fireEvent.change(screen.getByLabelText("Confirm password", { selector: "input" }), {
      target: { value: "correct horse" },
    })
    fireEvent.click(screen.getByRole("button", { name: "Create account" }))

    await waitFor(() => expect(router.state.location.pathname).toBe("/login"))
    expect(router.state.location.search).toEqual({
      registered: true,
      redirect: "/collections?q=binder",
    })
  })
})
