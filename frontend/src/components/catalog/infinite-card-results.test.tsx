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
    Link: ({ children, params: _params, to: _to, ...props }: any) => (
      <a {...props}>{children}</a>
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

  it("announces how many cards are loaded politely", () => {
    renderResults()
    const status = screen.getByText("200 of 500 cards loaded")
    expect(status.getAttribute("aria-live")).toBe("polite")
  })

  it("loads the next page from the Load more button with the keyboard", async () => {
    const onLoadMore = renderResults()
    const button = screen.getByRole("button", { name: "Load more" })
    expect(button.getAttribute("aria-disabled")).toBe("false")

    button.focus()
    await userEvent.keyboard("{Enter}")
    expect(onLoadMore).toHaveBeenCalledTimes(1)
  })

  it("keeps the button focusable but inert when there is nothing to load", async () => {
    const onLoadMore = renderResults({ hasNextPage: false, total: 200 })
    const button = screen.getByRole("button", { name: "Load more" })
    expect(button.getAttribute("aria-disabled")).toBe("true")
    expect(button.hasAttribute("disabled")).toBe(false)

    button.focus()
    await userEvent.keyboard("{Enter}")
    expect(document.activeElement).toBe(button)
    expect(onLoadMore).not.toHaveBeenCalled()
  })

  it("is inert while a fetch is in flight", async () => {
    const onLoadMore = renderResults({ isFetching: true })
    await userEvent.click(screen.getByRole("button", { name: "Load more" }))
    expect(onLoadMore).not.toHaveBeenCalled()
  })
})
