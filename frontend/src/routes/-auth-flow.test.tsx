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
import { ThemeProvider } from "@/components/theme-provider"
import { useClerk, useUser } from "@clerk/react"
import type { AuthGate } from "@/lib/clerk-auth"
import { clerkUserResult } from "@/test-clerk"
import { Route as AuthenticatedRoute } from "./_authenticated"
import { Route as AccountRoute } from "./_authenticated/account"
import { Route as AuthRoute } from "./auth/route"
import { Route as LoginRoute } from "./auth/login/index"
import { Route as LoginStepRoute } from "./auth/login/$"
import { Route as RegisterRoute } from "./auth/register/index"
import { Route as RegisterStepRoute } from "./auth/register/$"
import { Route as AuthIndexRoute } from "./auth/index"

// Sign-in and sign-up flows through the real route files (guards, redirect
// validation, splat routes) mounted at their real paths, with Clerk faked at
// its edge: its hooks, and its prebuilt components, which are stubbed to show
// the props they were given. The shell around them has its own tests.
// oxlint-disable-next-line anti-slop/no-module-mocking
vi.mock("@clerk/react", () => ({
  useUser: vi.fn(),
  useClerk: vi.fn(),
  SignIn: (props: ClerkComponentProps) => (
    <div data-testid="sign-in" data-props={JSON.stringify(props)} />
  ),
  SignUp: (props: ClerkComponentProps) => (
    <div data-testid="sign-up" data-props={JSON.stringify(props)} />
  ),
}))

afterEach(() => {
  cleanup()
  vi.restoreAllMocks()
  vi.unstubAllGlobals()
})

interface RouteBits {
  beforeLoad?: unknown
  head?: unknown
  validateSearch?: unknown
  component?: unknown
  notFoundComponent?: unknown
  ssr?: unknown
}

/**
 * The route options a page contributes, without the path the tree assigns.
 * SAFETY: `any` because the per-route generics of a file route can't be
 * spelled for a hand-built tree; these are the options under test.
 */
function pick(options: RouteBits): any {
  const {
    beforeLoad,
    head,
    validateSearch,
    component,
    notFoundComponent,
    ssr,
  } = options
  return { beforeLoad, head, validateSearch, component, notFoundComponent, ssr }
}

interface ClerkComponentProps {
  routing?: string
  path?: string
  forceRedirectUrl?: string
  signUpUrl?: string
  signInUrl?: string
}

function clerkProps(testId: "sign-in" | "sign-up"): ClerkComponentProps {
  const props = screen.getByTestId(testId).getAttribute("data-props")
  // SAFETY: data-props is the JSON the stubbed Clerk component wrote below.
  return JSON.parse(props ?? "{}") as ClerkComponentProps
}

async function visit(
  url: string,
  { signedIn }: { signedIn: boolean | Promise<boolean> }
) {
  vi.stubGlobal("matchMedia", () => ({
    matches: false,
    addEventListener: () => {},
    removeEventListener: () => {},
  }))
  // The signed-in home page and account page fetch; none of that is under test.
  vi.stubGlobal(
    "fetch",
    vi.fn(() => new Promise<Response>(() => {}))
  )
  const state = signedIn === true ? "signed-in" : "guest"
  vi.mocked(useUser).mockReturnValue(clerkUserResult(state))
  // SAFETY: partial hook result covering only `signOut`.
  vi.mocked(useClerk).mockReturnValue({ signOut: async () => {} } as any)

  const queryClient = new QueryClient()
  const auth: AuthGate = { isSignedIn: async () => signedIn }
  const root = createRootRouteWithContext<{
    queryClient: QueryClient
    auth: AuthGate
  }>()({
    // ThemeProvider (the auth shell's toggle reads it) needs router context.
    component: () => (
      <ThemeProvider>
        <Outlet />
      </ThemeProvider>
    ),
  })
  const stub = (path: string, name: string) =>
    createRoute({
      getParentRoute: () => root,
      path,
      component: () => <h1>{name}</h1>,
    })
  const auth_ = createRoute({
    getParentRoute: () => root,
    path: "/auth",
    ...pick(AuthRoute.options),
  })
  const authed = createRoute({
    getParentRoute: () => root,
    id: "_authenticated",
    ...pick(AuthenticatedRoute.options),
  })
  const routeTree = root.addChildren([
    stub("/", "Home"),
    auth_.addChildren([
      createRoute({
        getParentRoute: () => auth_,
        path: "/",
        ...pick(AuthIndexRoute.options),
      }),
      createRoute({
        getParentRoute: () => auth_,
        path: "/login",
        ...pick(LoginRoute.options),
      }),
      createRoute({
        getParentRoute: () => auth_,
        path: "/login/$",
        ...pick(LoginStepRoute.options),
      }),
      createRoute({
        getParentRoute: () => auth_,
        path: "/register",
        ...pick(RegisterRoute.options),
      }),
      createRoute({
        getParentRoute: () => auth_,
        path: "/register/$",
        ...pick(RegisterStepRoute.options),
      }),
    ]),
    authed.addChildren([
      createRoute({
        getParentRoute: () => authed,
        path: "/account",
        ...pick(AccountRoute.options),
      }),
      createRoute({
        getParentRoute: () => authed,
        path: "/collections",
        component: () => <h1>Collections</h1>,
      }),
    ]),
  ])
  const router = createRouter({
    routeTree,
    history: createMemoryHistory({ initialEntries: [url] }),
    context: { queryClient, auth },
  })
  render(
    <QueryClientProvider client={queryClient}>
      <RouterProvider router={router} />
    </QueryClientProvider>
  )
  return router
}

describe("guest on a protected page", () => {
  it("is sent to login, remembering the page and its query for Clerk to return to", async () => {
    const router = await visit("/collections?q=binder", { signedIn: false })

    await screen.findByTestId("sign-in")
    expect(router.state.location.pathname).toBe("/auth/login")
    expect(router.state.location.search).toEqual({
      redirect: "/collections?q=binder",
    })
    expect(clerkProps("sign-in")).toMatchObject({
      routing: "path",
      path: "/auth/login",
      forceRedirectUrl: "/collections?q=binder",
      signUpUrl: "/auth/register?redirect=%2Fcollections%3Fq%3Dbinder",
    })
    expect(screen.getByRole("heading", { level: 1 }).textContent).toBe("Log in")
  })

  it("defaults the post-sign-in destination to /account", async () => {
    await visit("/auth/login", { signedIn: false })

    await screen.findByTestId("sign-in")
    expect(clerkProps("sign-in")).toMatchObject({
      forceRedirectUrl: "/account",
      signUpUrl: "/auth/register",
    })
  })

  it.each(["https://evil.example/", "//evil.example", "/\\evil.example"])(
    "ignores the off-origin redirect %s",
    async (target) => {
      await visit(`/auth/login?redirect=${encodeURIComponent(target)}`, {
        signedIn: false,
      })

      await screen.findByTestId("sign-in")
      expect(clerkProps("sign-in")).toMatchObject({
        forceRedirectUrl: "/account",
        signUpUrl: "/auth/register",
      })
    }
  )
})

describe("sign-in and sign-up addresses", () => {
  it("renders Clerk's sign-up at /auth/register and carries the redirect to the login link", async () => {
    await visit("/auth/register?redirect=%2Finventory", { signedIn: false })

    await screen.findByTestId("sign-up")
    expect(clerkProps("sign-up")).toMatchObject({
      routing: "path",
      path: "/auth/register",
      forceRedirectUrl: "/inventory",
      signInUrl: "/auth/login?redirect=%2Finventory",
    })
    expect(screen.getByRole("heading", { level: 1 }).textContent).toBe(
      "Create an account"
    )
  })

  it.each([
    ["/auth/login/factor-one", "sign-in"],
    ["/auth/login/sso-callback", "sign-in"],
    ["/auth/register/verify-email-address", "sign-up"],
    ["/auth/register/sso-callback", "sign-up"],
  ] as const)("serves Clerk's step %s", async (url, testId) => {
    await visit(url, { signedIn: false })

    expect(await screen.findByTestId(testId)).toBeTruthy()
  })

  it("keeps the minimal auth shell and the noindex tag on those pages", async () => {
    const router = await visit("/auth/login", { signedIn: false })

    await screen.findByTestId("sign-in")
    expect(screen.getAllByRole("main")).toHaveLength(1)
    expect(screen.queryByRole("navigation")).toBeNull()
    expect(screen.getByRole("link", { name: "Cardstack" })).toBeTruthy()
    const meta = router.state.matches.flatMap((match) => match.meta ?? [])
    expect(meta).toContainEqual({ title: "Log in · Cardstack" })
    expect(meta).toContainEqual({ name: "robots", content: "noindex" })
  })

  it("does not serve the bare /auth address", async () => {
    await visit("/auth", { signedIn: false })

    expect(
      await screen.findByRole("heading", { level: 1, name: "Page not found" })
    ).toBeTruthy()
    expect(screen.queryByTestId("sign-in")).toBeNull()
  })
})

describe("signed-in user", () => {
  it.each(["/auth/login", "/auth/register", "/auth/login/factor-one"])(
    "is sent from %s to /",
    async (url) => {
      const router = await visit(url, { signedIn: true })

      await vi.waitFor(() => expect(router.state.location.pathname).toBe("/"))
      expect(screen.queryByTestId("sign-in")).toBeNull()
      expect(screen.queryByTestId("sign-up")).toBeNull()
    }
  )

  it("reaches a protected page, which shows their Clerk name and email", async () => {
    const router = await visit("/account", { signedIn: true })

    expect(
      await screen.findByRole("heading", { level: 1, name: "Account" })
    ).toBeTruthy()
    expect(router.state.location.pathname).toBe("/account")
    expect(screen.getByText("ada@example.com", { selector: "dd" })).toBeTruthy()
    expect(screen.getByText("Ada Lovelace", { selector: "dd" })).toBeTruthy()
    expect(screen.getByRole("button", { name: "Log out" })).toBeTruthy()
  })
})

describe("while Clerk is still loading", () => {
  it("neither redirects nor shows the protected page until Clerk answers", async () => {
    let finishLoading: (signedIn: boolean) => void = () => {}
    const loading = new Promise<boolean>((resolve) => {
      finishLoading = resolve
    })
    const router = await visit("/account", { signedIn: loading })

    await new Promise((resolve) => setTimeout(resolve, 50))
    expect(screen.queryByRole("heading", { name: "Account" })).toBeNull()
    expect(screen.queryByTestId("sign-in")).toBeNull()
    expect(router.state.location.pathname).not.toBe("/auth/login")

    finishLoading(true)
    expect(
      await screen.findByRole("heading", { level: 1, name: "Account" })
    ).toBeTruthy()
  })
})
