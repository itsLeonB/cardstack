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
const LOOKUP_TIMEOUT = 15000

describe("Add to Wishlist from the catalog", () => {
  it("adds a Card from a search result with the wanted quantity and shows its state", async () => {
    const { requests, writes } = await visitCatalog(
      "/catalog/search?name=Card",
      {
        signedIn: true,
      }
    )

    await userEvent.click(
      (
        await screen.findAllByRole(
          "button",
          { name: /^Edit Wishlist quantity for Card 1(?:, wanted \d+)?$/ },
          { timeout: LOOKUP_TIMEOUT }
        )
      )[0]
    )
    await userEvent.click(
      screen.getByRole("button", { name: "Increase wanted quantity of Card 1" })
    )
    expect(writes).toEqual([])
    await userEvent.click(screen.getByRole("button", { name: "Save" }))

    await screen.findByText("Want 2 of Card 1.")
    expect(writes).toEqual([
      { method: "PUT", path: "/wishlist/items/card-1", body: { quantity: 2 } },
    ])
    // The control shows the saved quantity, and the Wishlist's data was refreshed.
    await screen.findByText("Wanted ×2")
    screen.getByRole("button", {
      name: /^Edit Wishlist quantity for Card 1(?:, wanted \d+)?$/,
    })
    expect(wishlistLookups(requests)).toHaveLength(2)
  })

  it("replaces the wanted quantity of a Card already on the Wishlist", async () => {
    const { writes } = await visitCatalog("/catalog/search?name=Card", {
      signedIn: true,
      wishlist: new Map([["card-1", 3]]),
    })

    await userEvent.click(
      (
        await screen.findAllByRole(
          "button",
          { name: /^Edit Wishlist quantity for Card 1(?:, wanted \d+)?$/ },
          { timeout: LOOKUP_TIMEOUT }
        )
      )[0]
    )
    expect(screen.getByText("Wanted quantity: 3")).toBeTruthy()
    const increase = screen.getByRole("button", {
      name: "Increase wanted quantity of Card 1",
    })
    await userEvent.click(increase)
    await userEvent.click(increase)
    expect(writes).toEqual([])
    await userEvent.click(screen.getByRole("button", { name: "Save" }))

    await screen.findByText("Want 5 of Card 1.")
    // One write for one item: the API replaces the wanted quantity.
    expect(writes).toEqual([
      { method: "PUT", path: "/wishlist/items/card-1", body: { quantity: 5 } },
    ])
    expect(await screen.findAllByText("Wanted ×5")).toHaveLength(1)
  })

  it("discards an unsaved quantity when the popup is dismissed", async () => {
    const { writes } = await visitCatalog("/catalog/search?name=Card", {
      signedIn: true,
    })
    const opener = (
      await screen.findAllByRole(
        "button",
        { name: /^Edit Wishlist quantity for Card 1(?:, wanted \d+)?$/ },
        { timeout: LOOKUP_TIMEOUT }
      )
    )[0]
    await userEvent.click(opener)
    await userEvent.click(
      screen.getByRole("button", { name: "Increase wanted quantity of Card 1" })
    )
    await userEvent.keyboard("{Escape}")

    expect(writes).toEqual([])
    expect(screen.queryByText("Wanted quantity: 2")).toBeNull()
    await userEvent.click(opener)
    expect(screen.getByText("Wanted quantity: 1")).toBeTruthy()
  })

  it("keeps the popup open after a failed save so the user can retry", async () => {
    const { writes } = await visitCatalog("/catalog/search?name=Card", {
      signedIn: true,
    })
    const opener = (
      await screen.findAllByRole(
        "button",
        { name: /^Edit Wishlist quantity for Card 1(?:, wanted \d+)?$/ },
        { timeout: LOOKUP_TIMEOUT }
      )
    )[0]
    const fetchMock = vi.mocked(fetch)
    fetchMock.mockImplementationOnce(async (input) => {
      if (new URL(String(input)).pathname.startsWith("/wishlist/items/")) {
        return new Response(JSON.stringify({ detail: "Save failed." }), {
          status: 500,
          headers: { "content-type": "application/json" },
        })
      }
      return new Response(null, { status: 500 })
    })
    await userEvent.click(opener)
    await userEvent.click(screen.getByRole("button", { name: "Save" }))

    expect((await screen.findByRole("alert")).textContent).toContain(
      "Save failed."
    )
    expect(screen.getByText("Wishlist for Card 1")).toBeTruthy()
    expect(writes).toEqual([])
    await userEvent.click(screen.getByRole("button", { name: "Save" }))
    expect(await screen.findByText("Want 1 of Card 1.")).toBeTruthy()
    expect(writes).toEqual([
      { method: "PUT", path: "/wishlist/items/card-1", body: { quantity: 1 } },
    ])
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
        name: /^Edit Wishlist quantity for Card 1(?:, wanted \d+)?$/,
      })
    )
    expect(screen.getByText("Wanted quantity: 1")).toBeTruthy()
    await userEvent.click(screen.getByRole("button", { name: "Save" }))

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
