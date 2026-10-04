import { afterEach, describe, expect, it, vi } from "vitest"
import { cleanup, render, screen } from "@testing-library/react"
import type * as TanStackRouter from "@tanstack/react-router"
import { CardHoldings } from "./card-holdings"
import { useListCardHoldings } from "@/generated/endpoints/inventory/inventory"
import { useSession } from "@/lib/session"

// oxlint-disable-next-line anti-slop/no-module-mocking
vi.mock("@/generated/endpoints/inventory/inventory", () => ({
  useListCardHoldings: vi.fn(),
}))
// oxlint-disable-next-line anti-slop/no-module-mocking
vi.mock("@/lib/session", () => ({ useSession: vi.fn() }))
// oxlint-disable-next-line anti-slop/no-module-mocking
vi.mock("@tanstack/react-router", async (importOriginal) => {
  const actual = await importOriginal<typeof TanStackRouter>()
  return {
    ...actual,
    Link: ({ children, params, to, search, ...props }: any) => (
      <a
        href={
          to.replace("$collectionId", params?.collectionId ?? "") +
          (search?.redirect
            ? `?redirect=${encodeURIComponent(search.redirect)}`
            : "")
        }
        {...props}
      >
        {children}
      </a>
    ),
  }
})

afterEach(() => cleanup())

interface HoldingsState {
  isPending: boolean
  isError: boolean
  data?: { status: number; data: { data?: unknown[]; detail?: string } }
}

function setup(
  session: { isAuthenticated: boolean },
  holdings: HoldingsState,
  loginRedirect?: string
) {
  // SAFETY: tests supply only the fields CardHoldings reads from these hooks.
  vi.mocked(useSession).mockReturnValue({ isLoading: false, ...session } as any)
  // SAFETY: tests supply only the fields CardHoldings reads from this hook.
  vi.mocked(useListCardHoldings).mockReturnValue(holdings as any)
  render(<CardHoldings cardId="card-1" loginRedirect={loginRedirect} />)
}
const ok = (data: unknown[]): HoldingsState => ({
  isPending: false,
  isError: false,
  data: { status: 200, data: { data } },
})

describe("CardHoldings", () => {
  it("lists each Collection with its quantity, linking to the Collection", () => {
    setup(
      { isAuthenticated: true },
      ok([
        { collection: { id: "c1", name: "Binder A" }, quantity: 3 },
        { collection: { id: "c2", name: "Binder B" }, quantity: 1 },
      ])
    )
    expect(
      screen.getByRole("link", { name: "Binder A" }).getAttribute("href")
    ).toBe("/collections/c1")
    expect(screen.getByText("×3")).toBeTruthy()
    expect(screen.getByRole("link", { name: "Binder B" })).toBeTruthy()
  })

  it("shows an explicit empty message, not an error, when held nowhere", () => {
    setup({ isAuthenticated: true }, ok([]))
    expect(screen.getByText("Not in any of your Collections")).toBeTruthy()
    expect(screen.queryByRole("alert")).toBeNull()
  })

  it("shows an error when the request fails", () => {
    setup(
      { isAuthenticated: true },
      { isPending: false, isError: true, data: undefined }
    )
    expect(screen.getByRole("alert").textContent).toContain(
      "Could not load your collections"
    )
  })

  it("shows the API error detail for a non-200 response", () => {
    setup(
      { isAuthenticated: true },
      {
        isPending: false,
        isError: false,
        data: { status: 500, data: { detail: "boom" } },
      }
    )
    expect(screen.getByRole("alert").textContent).toBe("boom")
  })

  it("shows a loading state while pending", () => {
    setup(
      { isAuthenticated: true },
      { isPending: true, isError: false, data: undefined }
    )
    expect(screen.getByLabelText("Loading your collections")).toBeTruthy()
  })

  it("sends the sign-in prompt back to the same Card page", () => {
    setup(
      { isAuthenticated: false },
      { isPending: true, isError: false, data: undefined },
      "/catalog/cards/s1/001"
    )
    expect(
      screen.getByRole("link", { name: "Sign in" }).getAttribute("href")
    ).toBe("/auth/login?redirect=%2Fcatalog%2Fcards%2Fs1%2F001")
  })

  it("prompts sign-in and does not enable the holdings query when signed out", () => {
    setup(
      { isAuthenticated: false },
      { isPending: true, isError: false, data: undefined }
    )
    expect(screen.getByRole("link", { name: "Sign in" })).toBeTruthy()
    expect(
      vi.mocked(useListCardHoldings).mock.lastCall?.[1]?.query?.enabled
    ).toBe(false)
  })
})
