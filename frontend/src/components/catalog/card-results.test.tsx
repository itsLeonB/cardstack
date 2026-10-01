import { afterEach, describe, expect, it, vi } from "vitest"
import { cleanup, fireEvent, render, screen } from "@testing-library/react"
import type * as TanStackRouter from "@tanstack/react-router"
import { CardResults } from "./card-results"
import type { CardSummary } from "@/generated/models"

// This project's vitest config doesn't enable RTL's automatic per-test
// cleanup (no `globals: true`), and several tests below use ambiguous
// role queries (e.g. "Previous"/"Next" buttons) that collide once more
// than one render is left mounted in the same test file.
afterEach(() => cleanup())

// CardTile links to the card's Expansion Set via TanStack Router's `Link`,
// which needs a `<RouterProvider>` in the tree to resolve routes/hrefs.
// These tests render CardResults in isolation from the router, so `Link` is
// swapped for a plain anchor — the same isolation approach health-check.test.tsx
// uses for the generated query hooks.
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

const card: CardSummary = {
  id: "card-1",
  name: "Pikachu V",
  localId: "048",
  category: "Pokémon",
  tags: ["V"],
  illustrator: "someone",
  imageUrl: "https://example.com/pikachu-v.png",
  expansionSet: { id: "set-1", code: "SCE", name: "V Starter Deck Kebangkitan" },
  rarity: { id: "rarity-1", code: "RR", name: "Double Rare" },
}

describe("CardResults", () => {
  it("renders loading skeletons while pending", () => {
    render(
      <CardResults
        cards={[]}
        total={0}
        page={1}
        limit={24}
        isPending
        isError={false}
        emptyMessage="No cards"
        onPageChange={vi.fn()}
      />
    )

    expect(screen.getByLabelText("Loading cards")).toBeTruthy()
  })

  it("renders an error message when the query failed", () => {
    render(
      <CardResults
        cards={[]}
        total={0}
        page={1}
        limit={24}
        isPending={false}
        isError
        errorMessage="Could not reach the backend."
        emptyMessage="No cards"
        onPageChange={vi.fn()}
      />
    )

    expect(screen.getByRole("alert").textContent).toBe("Could not reach the backend.")
  })

  it("renders the empty message when there are no results", () => {
    render(
      <CardResults
        cards={[]}
        total={0}
        page={1}
        limit={24}
        isPending={false}
        isError={false}
        emptyMessage="No cards match these filters."
        onPageChange={vi.fn()}
      />
    )

    expect(screen.getByText("No cards match these filters.")).toBeTruthy()
  })

  it("renders cards and omits pagination when everything fits on one page", () => {
    render(
      <CardResults
        cards={[card]}
        total={1}
        page={1}
        limit={24}
        isPending={false}
        isError={false}
        emptyMessage="No cards"
        onPageChange={vi.fn()}
      />
    )

    expect(screen.getByText("Pikachu V")).toBeTruthy()
    expect(screen.queryByRole("navigation", { name: "Pagination" })).toBeNull()
  })

  it("renders Previous/Next controls and reports page changes", () => {
    const onPageChange = vi.fn()

    render(
      <CardResults
        cards={[card]}
        total={50}
        page={2}
        limit={24}
        isPending={false}
        isError={false}
        emptyMessage="No cards"
        onPageChange={onPageChange}
      />
    )

    expect(screen.getByText("Page 2 of 3")).toBeTruthy()

    fireEvent.click(screen.getByRole("button", { name: /previous/i }))
    expect(onPageChange).toHaveBeenCalledWith(1)

    fireEvent.click(screen.getByRole("button", { name: /next/i }))
    expect(onPageChange).toHaveBeenCalledWith(3)
  })

  it("disables Previous on the first page and Next on the last page", () => {
    render(
      <CardResults
        cards={[card]}
        total={50}
        page={1}
        limit={24}
        isPending={false}
        isError={false}
        emptyMessage="No cards"
        onPageChange={vi.fn()}
      />
    )

    expect(
      screen.getByRole("button", { name: /previous/i }).hasAttribute("disabled")
    ).toBe(true)
    expect(
      screen.getByRole("button", { name: /next/i }).hasAttribute("disabled")
    ).toBe(false)
  })

  it("renders the per-card control slot inside each tile", () => {
    render(
      <CardResults
        cards={[card]}
        total={1}
        page={1}
        limit={24}
        isPending={false}
        isError={false}
        emptyMessage="No cards"
        onPageChange={vi.fn()}
        renderControl={(c) => <button type="button">Control for {c.name}</button>}
      />
    )
    expect(screen.getByRole("button", { name: "Control for Pikachu V" })).toBeTruthy()
  })
})
