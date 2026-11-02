import { afterEach, describe, expect, it, vi } from "vitest"
import { cleanup, screen } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { setTokenGetter } from "@/lib/http"
import { visitCatalog, wishlistLookups } from "@/test-catalog"

// "Add to Wishlist" from the catalog (ticket 8): the action on a search result
// and on the Card's page, its wanted quantity, replacing an existing one, and
// the Guest's sign-in prompt. The API stub answers the wanted-quantity lookups
// and records every write, so a test sees one PUT per action and no second item.
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

// The action appears once the wanted-quantity lookup answers, which the grid's
// 60 cards queue behind, so these waits are longer than the default second.
const LOOKUP_TIMEOUT = 5000

describe("Add to Wishlist from the catalog", () => {
  it("adds a Card from a search result with the wanted quantity and shows its state", async () => {
    const { requests, writes } = await visitCatalog(
      "/catalog/search?name=Card",
      {
        signedIn: true,
      }
    )

    await userEvent.click(
      await screen.findByRole(
        "button",
        { name: "Increase wanted quantity of Card 1" },
        { timeout: LOOKUP_TIMEOUT }
      )
    )
    await userEvent.click(
      screen.getByRole("button", { name: "Add Card 1 to your Wishlist" })
    )

    await screen.findByText("Want 2 of Card 1.")
    expect(writes).toEqual([
      { method: "PUT", path: "/wishlist/items/card-1", body: { quantity: 2 } },
    ])
    // The control shows the saved quantity, and the Wishlist's data was refreshed.
    await screen.findByText("Wanted ×2")
    screen.getByRole("button", {
      name: "Update the wanted quantity of Card 1",
    })
    expect(wishlistLookups(requests)).toHaveLength(2)
  })

  it("replaces the wanted quantity of a Card already on the Wishlist", async () => {
    const { writes } = await visitCatalog("/catalog/search?name=Card", {
      signedIn: true,
      wishlist: new Map([["card-1", 3]]),
    })

    await screen.findByText("Wanted ×3", undefined, {
      timeout: LOOKUP_TIMEOUT,
    })
    const increase = screen.getByRole("button", {
      name: "Increase wanted quantity of Card 1",
    })
    await userEvent.click(increase)
    await userEvent.click(increase)
    await userEvent.click(
      screen.getByRole("button", {
        name: "Update the wanted quantity of Card 1",
      })
    )

    await screen.findByText("Want 5 of Card 1.")
    // One write for one item: the API replaces the wanted quantity.
    expect(writes).toEqual([
      { method: "PUT", path: "/wishlist/items/card-1", body: { quantity: 5 } },
    ])
    expect(await screen.findAllByText("Wanted ×5")).toHaveLength(1)
  })

  it("sends a Guest to sign in instead of offering the action", async () => {
    await visitCatalog("/catalog/search?name=Card", { signedIn: false })

    await screen.findByText("24 of 100 cards loaded")
    const prompt = screen.getByRole("link", {
      name: "Sign in to add Cards to your Wishlist",
    })
    expect(prompt.getAttribute("href")).toMatch(/^\/auth\/login\?redirect=/)
    expect(screen.queryByRole("button", { name: /Wishlist/ })).toBeNull()
  })

  it("adds a Card from its catalog page", async () => {
    const { writes } = await visitCatalog("/catalog/cards/s1/1", {
      signedIn: true,
    })

    await userEvent.click(
      await screen.findByRole("button", {
        name: "Add Card 1 to your Wishlist",
      })
    )

    await screen.findByText("Want 1 of Card 1.")
    expect(writes).toEqual([
      { method: "PUT", path: "/wishlist/items/card-1", body: { quantity: 1 } },
    ])
  })

  it("prompts a Guest on the Card's page to sign in", async () => {
    const { router } = await visitCatalog("/catalog/cards/s1/1", {
      signedIn: false,
    })

    const back = router.state.location.href
    const prompts = await screen.findAllByRole("link", { name: "Sign in" })
    expect(prompts).toHaveLength(2)
    for (const prompt of prompts) {
      expect(prompt.getAttribute("href")).toBe(
        `/auth/login?redirect=${encodeURIComponent(back)}`
      )
    }
    expect(screen.queryByRole("button", { name: /Wishlist/ })).toBeNull()
  })
})
