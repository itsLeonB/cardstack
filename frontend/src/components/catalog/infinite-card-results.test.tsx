import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"
import { cleanup, render, screen } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import type * as TanStackRouter from "@tanstack/react-router"
import { InfiniteCardResults } from "./infinite-card-results"
import type { InfiniteCardResultsProps } from "./infinite-card-results"
import { stubGridLayout } from "@/test-grid-layout"
import type { CardSummary } from "@/generated/models"

// CardTile links through TanStack Router's `Link`, which needs a router.
// oxlint-disable-next-line anti-slop/no-module-mocking
vi.mock("@tanstack/react-router", async (importOriginal) => {
  const actual = await importOriginal<typeof TanStackRouter>()
  return {
    ...actual,
    Link: ({ children, params: _params, to, search, ...props }: any) => (
      <a
        href={
          search?.redirect
            ? `${to}?redirect=${encodeURIComponent(search.redirect)}`
            : to
        }
        {...props}
      >
        {children}
      </a>
    ),
  }
})

beforeEach(() => {
  stubGridLayout()
})

afterEach(() => {
  cleanup()
  vi.restoreAllMocks()
  vi.unstubAllGlobals()
})

const makeCard = (index: number): CardSummary => ({
  id: `card-${index}`,
  name: `Card ${index}`,
  localId: String(index),
  category: "Pokémon",
  tags: [],
  illustrator: "someone",
  imageUrl: "",
  expansionSet: { id: "set-1", code: "SCE", name: "Starter", imageUrl: "" },
  rarity: { id: "r-1", code: "C", name: "Common" },
})

// 200 cards is far more rows than the window renders, so the scroll trigger
// stays quiet and only the button can call `onLoadMore`.
function renderResults(overrides: Partial<InfiniteCardResultsProps> = {}) {
  const onLoadMore = vi.fn()
  render(
    <InfiniteCardResults
      cards={Array.from({ length: 200 }, (_, index) => makeCard(index))}
      total={500}
      isPending={false}
      isError={false}
      emptyMessage="No cards"
      hasNextPage
      isFetching={false}
      onLoadMore={onLoadMore}
      {...overrides}
    />
  )
  return onLoadMore
}

describe("InfiniteCardResults", () => {
  it("shows skeletons while the first page is pending", () => {
    renderResults({ isPending: true, cards: [] })
    expect(screen.getByLabelText("Loading cards")).toBeTruthy()
  })

  it("shows the error when the first page failed", () => {
    renderResults({ isError: true, cards: [], errorMessage: "Boom" })
    expect(screen.getByRole("alert").textContent).toBe("Boom")
  })

  it("shows the empty message", () => {
    renderResults({ cards: [], total: 0 })
    expect(screen.getByText("No cards")).toBeTruthy()
  })

  it("keeps the loaded cards and the button when a later page fails", () => {
    renderResults({ isError: true, errorMessage: "Boom" })
    expect(screen.getByRole("alert").textContent).toBe("Boom")
    expect(screen.getByTitle("Card 0")).toBeTruthy()
    expect(screen.getByRole("button", { name: "Load more" })).toBeTruthy()
  })

  it("reports each tile's place in the whole result set, not just the loaded part", () => {
    renderResults()
    const tile = screen.getAllByRole("listitem")[0]!
    expect(tile.getAttribute("aria-posinset")).toBe("1")
    expect(tile.getAttribute("aria-setsize")).toBe("500")
  })

  it("announces how many cards are loaded politely", () => {
    renderResults()
    const status = screen.getByText("200 of 500 cards loaded")
    expect(status.getAttribute("aria-live")).toBe("polite")
  })

  it("keeps the same live region mounted from pending to loaded", () => {
    const props = {
      total: 500,
      isError: false,
      emptyMessage: "No cards",
      hasNextPage: true,
      isFetching: false,
      onLoadMore: vi.fn(),
    }
    const { container, rerender } = render(
      <InfiniteCardResults {...props} cards={[]} isPending />
    )
    const region = container.querySelector("[aria-live=polite]")
    expect(region).not.toBeNull()
    expect(region?.textContent).toBe("")

    rerender(
      <InfiniteCardResults
        {...props}
        cards={[makeCard(0), makeCard(1)]}
        isPending={false}
      />
    )
    expect(container.querySelector("[aria-live=polite]")).toBe(region)
    expect(region?.textContent).toBe("2 of 500 cards loaded")
  })

  it("loads the next page from the Load more button with the keyboard", async () => {
    const onLoadMore = renderResults()
    const button = screen.getByRole("button", { name: "Load more" })
    expect(button.getAttribute("aria-disabled")).toBe("false")

    button.focus()
    await userEvent.keyboard("{Enter}")
    expect(onLoadMore).toHaveBeenCalledTimes(1)
  })

  it("hides the button once every page is loaded", () => {
    renderResults({ hasNextPage: false, total: 200 })
    expect(screen.queryByRole("button", { name: "Load more" })).toBeNull()
  })

  it("keeps the button, focusable but inert, while a fetch is in flight", async () => {
    const onLoadMore = renderResults({ isFetching: true })
    const button = screen.getByRole("button", { name: "Load more" })
    expect(button.getAttribute("aria-disabled")).toBe("true")
    expect(button.hasAttribute("disabled")).toBe(false)

    await userEvent.click(button)
    expect(onLoadMore).not.toHaveBeenCalled()
  })

  describe("for a Guest", () => {
    const guest = { guest: true, signInRedirect: "/catalog/search?name=pika" }
    const tenCards = Array.from({ length: 10 }, (_, index) => makeCard(index))

    it("shows 'Sign in to see more' in place of Load more, returning to the same view", () => {
      renderResults(guest)
      const link = screen.getByRole("link", { name: "Sign in to see more" })
      expect(link.getAttribute("href")).toBe(
        "/auth/login?redirect=%2Fcatalog%2Fsearch%3Fname%3Dpika"
      )
      expect(screen.queryByRole("button", { name: "Load more" })).toBeNull()
    })

    it("never asks for the next page, not even when scrolled to the end", () => {
      const control = renderResults({ cards: tenCards, total: 500 })
      expect(control).toHaveBeenCalled()

      const locked = renderResults({ ...guest, cards: tenCards, total: 500 })
      expect(locked).not.toHaveBeenCalled()
    })

    it("shows no prompt when the first page is the whole result", () => {
      renderResults({ ...guest, hasNextPage: false, total: 200 })
      expect(screen.queryByRole("link", { name: /sign in/i })).toBeNull()
    })

    it("announces the prompt through a live region that is always mounted", () => {
      const props = {
        ...guest,
        total: 500,
        isError: false,
        emptyMessage: "No cards",
        hasNextPage: true,
        isFetching: false,
        onLoadMore: vi.fn(),
      }
      const { container, rerender } = render(
        <InfiniteCardResults {...props} cards={[]} isPending />
      )
      // The first live region is the card count; the second is the prompt's.
      const regionOf = () => container.querySelectorAll("[aria-live=polite]")[1]
      const region = regionOf()
      expect(region).toBeDefined()
      expect(region?.textContent).toBe("")

      rerender(
        <InfiniteCardResults {...props} cards={tenCards} isPending={false} />
      )
      expect(regionOf()).toBe(region)
      expect(region?.textContent).toBe("Sign in to see more")
    })

    it("keeps the prompt reachable by keyboard", async () => {
      renderResults(guest)
      const link = screen.getByRole("link", { name: "Sign in to see more" })
      link.focus()
      expect(document.activeElement).toBe(link)
    })
  })

  describe("when the API answers login_required", () => {
    it("shows a sign-in prompt, not an error, when the first page is locked", () => {
      renderResults({
        loginRequired: true,
        isError: true,
        cards: [],
        errorMessage: "sign in to use this part of the catalog",
        signInRedirect: "/catalog/search?rarityId=r1",
      })
      expect(screen.queryByRole("alert")).toBeNull()
      const link = screen.getByRole("link", { name: "Sign in" })
      expect(link.getAttribute("href")).toContain("redirect=")
    })

    it("replaces the load-more error with the prompt once cards are showing", () => {
      renderResults({ loginRequired: true, isError: true })
      expect(screen.queryByRole("alert")).toBeNull()
      expect(screen.getByTitle("Card 0")).toBeTruthy()
      screen.getByRole("link", { name: "Sign in" })
    })
  })
})
