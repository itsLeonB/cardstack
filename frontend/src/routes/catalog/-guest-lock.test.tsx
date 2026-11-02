import { afterEach, describe, expect, it, vi } from "vitest"
import { cleanup, screen } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { toast } from "sonner"
import { setTokenGetter } from "@/lib/http"
import { pagesAsked, visitCatalog } from "@/test-catalog"

// The catalog's guest lock (ticket 10) on its routes: one page of at most 24,
// 401 `login_required` for a later page, a rarity, category or tag filter, or
// the facets, and a sign-in prompt in place of each of them.
// oxlint-disable-next-line anti-slop/no-module-mocking
vi.mock("@clerk/react", () =>
  import("@/test-clerk").then((m) => m.clerkModule())
)
// oxlint-disable-next-line anti-slop/no-module-mocking
vi.mock("sonner", () => ({ toast: { error: vi.fn(), success: vi.fn() } }))

afterEach(() => {
  cleanup()
  setTokenGetter(null)
  vi.restoreAllMocks()
  vi.unstubAllGlobals()
})

const isDisabled = (name: RegExp | string) =>
  screen.getByRole("button", { name }).hasAttribute("disabled")

describe("catalog search as a Guest", () => {
  it("locks rarity, category and tag with a prompt, and keeps name search and Expansion Set open", async () => {
    await visitCatalog("/catalog/search?name=Card", { signedIn: false })

    await screen.findByText("24 of 100 cards loaded")
    expect(isDisabled("Rarity")).toBe(true)
    expect(isDisabled("Category")).toBe(true)
    expect(isDisabled("Tag")).toBe(true)
    expect(isDisabled("Expansion Set")).toBe(false)
    screen.getByRole("link", { name: "Sign in to use filters" })

    // The Expansion Set dropdown is filled from the Series list, since facets are locked.
    await userEvent.click(screen.getByRole("button", { name: "Expansion Set" }))
    screen.getByRole("checkbox", { name: "Scarlet Base (SV1)" })
  })

  it("ends the first page with 'Sign in to see more' and never asks for page 2", async () => {
    const { requests } = await visitCatalog("/catalog/search?name=Card", {
      signedIn: false,
    })

    await screen.findByRole("link", { name: "Sign in to see more" })
    expect(screen.queryByRole("button", { name: "Load more" })).toBeNull()
    expect(pagesAsked(requests)).toEqual([1])
  })

  it("brings the user back to the same search after signing in, from every prompt", async () => {
    const { router } = await visitCatalog(
      "/catalog/search?name=Card&expansionSetId=s1",
      { signedIn: false }
    )
    const more = await screen.findByRole("link", {
      name: "Sign in to see more",
    })
    // The router's own address for this view (it writes arrays in its canonical form).
    const back = router.state.location.href
    expect(back).toContain("name=Card")
    const href = `/auth/login?redirect=${encodeURIComponent(back)}`
    expect(more.getAttribute("href")).toBe(href)
    expect(
      screen
        .getByRole("link", { name: "Sign in to use filters" })
        .getAttribute("href")
    ).toBe(href)
    expect(
      screen
        .getByRole("link", { name: "Sign in to add cards to a Collection" })
        .getAttribute("href")
    ).toBe(href)

    await userEvent.click(more)
    await screen.findByRole("heading", { name: "Sign in page" })
    // The login page's own same-origin rule accepted the target.
    expect(router.state.location.search).toEqual({ redirect: back })
  })

  it("shows a sign-in prompt, never an error or toast, for a locked filter in the address", async () => {
    await visitCatalog("/catalog/search?rarityId=r1", { signedIn: false })

    await screen.findByRole("link", { name: "Sign in" })
    expect(screen.queryByRole("alert")).toBeNull()
    expect(vi.mocked(toast.error)).not.toHaveBeenCalled()
    // The chip stays, so the user can still remove the filter.
    expect(screen.getByRole("button", { name: /^Remove / })).toBeTruthy()
  })

  it("keeps the card count live and the prompt reachable by keyboard", async () => {
    await visitCatalog("/catalog/search?name=Card", { signedIn: false })

    const status = await screen.findByText("24 of 100 cards loaded")
    expect(status.getAttribute("aria-live")).toBe("polite")
    const link = screen.getByRole("link", { name: "Sign in to see more" })
    link.focus()
    expect(document.activeElement).toBe(link)
    expect(link.closest("[aria-live=polite]")).not.toBeNull()
  })
})

describe("catalog search signed in", () => {
  it("has every filter, no prompts, and a Load more that fetches page 2", async () => {
    const { requests } = await visitCatalog("/catalog/search?name=Card", {
      signedIn: true,
    })

    await screen.findByText("60 of 100 cards loaded")
    expect(isDisabled("Rarity")).toBe(false)
    expect(isDisabled("Category")).toBe(false)
    expect(screen.queryByRole("link", { name: /^Sign in/ })).toBeNull()
    expect(requests.some((url) => url.pathname === "/catalog/facets")).toBe(
      true
    )

    await userEvent.click(screen.getByRole("button", { name: "Load more" }))
    await screen.findByText("100 of 100 cards loaded")
    expect(pagesAsked(requests)).toContain(2)
  })
})

describe("Expansion Set page", () => {
  it("shows a Guest 'Sign in to see more' and returns to that set", async () => {
    const { requests } = await visitCatalog("/catalog/sets/s1", {
      signedIn: false,
    })

    const more = await screen.findByRole("link", {
      name: "Sign in to see more",
    })
    expect(more.getAttribute("href")).toBe(
      `/auth/login?redirect=${encodeURIComponent("/catalog/sets/s1")}`
    )
    expect(pagesAsked(requests)).toEqual([1])
  })

  it("lets a signed-in user load the next page", async () => {
    await visitCatalog("/catalog/sets/s1", { signedIn: true })

    await screen.findByRole("button", { name: "Load more" })
    expect(screen.queryByRole("link", { name: /^Sign in/ })).toBeNull()
  })
})
