import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"
import { act, cleanup, fireEvent, render, screen } from "@testing-library/react"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import type * as TanStackRouter from "@tanstack/react-router"
import { CollectionEntries } from "./collection-entries"
import { QUANTITY_DEBOUNCE_MS } from "@/lib/use-quantity-batch"
import {
  bulkUpdateCollectionEntries,
  useListCollectionEntries,
  useListCollectionFacets,
} from "@/generated/endpoints/inventory/inventory"
import { useListCatalogSeries, useSearchCatalogCards } from "@/generated/endpoints/catalog/catalog"
import type { CardSummary } from "@/generated/models"
import type { CatalogSearch } from "@/lib/catalog-search"

// Isolates the UI from the network; the generated hooks' wire behaviour is
// orval's job.
// oxlint-disable-next-line anti-slop/no-module-mocking
vi.mock("@/generated/endpoints/inventory/inventory", () => ({
  getListCollectionEntriesQueryKey: (id: string) => ["entries", id],
  useListCollectionEntries: vi.fn(),
  useListCollectionFacets: vi.fn(),
  bulkUpdateCollectionEntries: vi.fn(),
}))
// oxlint-disable-next-line anti-slop/no-module-mocking
vi.mock("@/generated/endpoints/catalog/catalog", () => ({
  useSearchCatalogCards: vi.fn(),
  useListCatalogSeries: vi.fn(),
}))
// CardTile links via TanStack Router's `Link`, which needs a router in the tree.
// oxlint-disable-next-line anti-slop/no-module-mocking
vi.mock("@tanstack/react-router", async (importOriginal) => {
  const actual = await importOriginal<typeof TanStackRouter>()
  return {
    ...actual,
    Link: ({ children, params: _params, to: _to, ...props }: any) => <a {...props}>{children}</a>,
  }
})

afterEach(() => {
  cleanup()
  vi.useRealTimers()
})

const card: CardSummary = {
  id: "card-1",
  name: "Pikachu V",
  localId: "048",
  category: "Pokémon",
  tags: [],
  illustrator: "someone",
  imageUrl: "",
  expansionSet: { id: "set-1", code: "SCE", name: "Starter" },
  rarity: { id: "r-1", code: "RR", name: "Double Rare" },
}

const onSearchChange = vi.fn()
const bulk = vi.mocked(bulkUpdateCollectionEntries)

function setList(items: { card: CardSummary; quantity: number }[], total = items.length) {
  // SAFETY: partial mock; the component reads only status/data and isPending/isError.
  vi.mocked(useListCollectionEntries).mockReturnValue({
    isPending: false,
    isError: false,
    data: {
      status: 200,
      data: { data: items, meta: { total, page: 1, limit: 24 } },
      headers: new Headers(),
    },
  } as any)
}

function renderEntries(search: CatalogSearch = { page: 1 }) {
  render(
    <QueryClientProvider client={new QueryClient()}>
      <CollectionEntries collectionId="col-1" search={search} onSearchChange={onSearchChange} />
    </QueryClientProvider>
  )
}

// SAFETY: the labelled control is an <input>.
const quantityInput = () => screen.getByLabelText("Quantity of Pikachu V") as HTMLInputElement

async function advance(ms: number) {
  await act(async () => {
    await vi.advanceTimersByTimeAsync(ms)
  })
}

describe("CollectionEntries", () => {
  beforeEach(() => {
    onSearchChange.mockReset()
    bulk.mockReset()
    // SAFETY: partial response; the hook reads only status and data.data.
    bulk.mockResolvedValue({ status: 200, data: { data: [] } } as any)
    // SAFETY: partial mock; only status/data/isError are read.
    vi.mocked(useSearchCatalogCards).mockReturnValue({
      isError: false,
      data: { status: 200, data: { data: [card] }, headers: new Headers() },
    } as any)
    // SAFETY: partial mock; only status/data are read.
    vi.mocked(useListCatalogSeries).mockReturnValue({} as any)
    // SAFETY: partial mock; only status/data are read.
    vi.mocked(useListCollectionFacets).mockReturnValue({
      data: {
        status: 200,
        data: {
          data: {
            expansionSets: [],
            rarities: [{ id: "r-1", code: "RR", name: "Double Rare", available: true }],
            categories: [],
            tags: [],
          },
        },
        headers: new Headers(),
      },
    } as any)
  })

  it("shows an empty state when the Collection has no Cards", () => {
    setList([])
    renderEntries()
    screen.getByText(/has no Cards yet/)
  })

  it("shows a filtered empty state when filters match nothing", () => {
    setList([])
    renderEntries({ page: 1, name: "zzz" })
    screen.getByText(/match these filters/)
  })

  it("renders each Card as a tile with a quantity control and no Save/Remove buttons", () => {
    setList([{ card, quantity: 3 }])
    renderEntries()
    expect(screen.getAllByText("Pikachu V").length).toBeGreaterThan(0)
    expect(quantityInput().value).toBe("3")
    expect(screen.queryByRole("button", { name: /^Save/ })).toBeNull()
    expect(screen.queryByRole("button", { name: /^Remove/ })).toBeNull()
  })

  it("refetches on mount and does not keep the list cached after leaving", () => {
    setList([{ card, quantity: 3 }])
    renderEntries()
    expect(useListCollectionEntries).toHaveBeenCalledWith(
      "col-1",
      { page: 1 },
      expect.objectContaining({
        query: expect.objectContaining({ refetchOnMount: "always", gcTime: 0 }),
      })
    )
  })

  it("applies +, - and typed quantities optimistically, then sends one bulk call after the debounce", async () => {
    vi.useFakeTimers()
    setList([{ card, quantity: 3 }])
    renderEntries()

    fireEvent.click(screen.getByRole("button", { name: "Increase quantity of Pikachu V" }))
    expect(quantityInput().value).toBe("4")
    fireEvent.click(screen.getByRole("button", { name: "Decrease quantity of Pikachu V" }))
    fireEvent.change(quantityInput(), { target: { value: "7" } })
    expect(quantityInput().value).toBe("7")

    await advance(QUANTITY_DEBOUNCE_MS - 1)
    expect(bulk).not.toHaveBeenCalled()
    await advance(1)
    expect(bulk).toHaveBeenCalledTimes(1)
    expect(bulk).toHaveBeenCalledWith("col-1", { items: [{ cardId: "card-1", quantity: 7 }] })
  })

  it("keeps the tile visible at 0 after the removal is saved", async () => {
    vi.useFakeTimers()
    setList([{ card, quantity: 1 }])
    // SAFETY: partial response; the hook reads only status and data.data.
    bulk.mockResolvedValue({
      status: 200,
      data: { data: [{ cardId: "card-1", quantity: 0, status: "removed" }] },
    } as any)
    renderEntries()

    fireEvent.click(screen.getByRole("button", { name: "Decrease quantity of Pikachu V" }))
    await advance(QUANTITY_DEBOUNCE_MS)

    expect(bulk).toHaveBeenCalledWith("col-1", { items: [{ cardId: "card-1", quantity: 0 }] })
    expect(screen.getAllByText("Pikachu V").length).toBeGreaterThan(0)
    expect(quantityInput().value).toBe("0")
    fireEvent.click(screen.getByRole("button", { name: "Increase quantity of Pikachu V" }))
    expect(quantityInput().value).toBe("1")
  })

  it("reverts a capacity-declined card with an error tied to it", async () => {
    vi.useFakeTimers()
    setList([{ card, quantity: 3 }])
    // SAFETY: partial response; the hook reads only status and data.data.
    bulk.mockResolvedValue({
      status: 200,
      data: {
        data: [
          { cardId: "card-1", quantity: 3, status: "declined", reason: "capacity_exceeded", message: "Capacity exceeded" },
        ],
      },
    } as any)
    renderEntries()

    fireEvent.change(quantityInput(), { target: { value: "50" } })
    await advance(QUANTITY_DEBOUNCE_MS)

    expect(screen.getByRole("alert").textContent).toBe("Capacity exceeded")
    expect(quantityInput().value).toBe("3")
  })

  it("flushes pending edits before changing a filter", async () => {
    vi.useFakeTimers()
    setList([{ card, quantity: 3 }])
    renderEntries()

    fireEvent.change(quantityInput(), { target: { value: "5" } })
    fireEvent.click(screen.getByRole("button", { name: "Rarity" }))
    fireEvent.click(screen.getByLabelText("Double Rare"))

    // The batch goes out without waiting for the debounce, and navigation follows it.
    await advance(0)
    expect(bulk).toHaveBeenCalledWith("col-1", { items: [{ cardId: "card-1", quantity: 5 }] })
    expect(onSearchChange).toHaveBeenCalledWith({ page: 1, rarityId: ["r-1"] })
  })

  it("flushes pending edits before changing page", async () => {
    vi.useFakeTimers()
    setList([{ card, quantity: 3 }], 50)
    renderEntries()

    fireEvent.change(quantityInput(), { target: { value: "5" } })
    fireEvent.click(screen.getByRole("button", { name: /next/i }))

    await advance(0)
    expect(bulk).toHaveBeenCalledTimes(1)
    expect(onSearchChange).toHaveBeenCalledWith({ page: 2 })
  })

  it("raises a card at 0 in place with +", async () => {
    vi.useFakeTimers()
    setList([{ card, quantity: 0 }])
    renderEntries()
    expect(screen.getByRole("button", { name: "Decrease quantity of Pikachu V" }).hasAttribute("disabled")).toBe(true)
    fireEvent.click(screen.getByRole("button", { name: "Increase quantity of Pikachu V" }))
    expect(quantityInput().value).toBe("1")
    await advance(QUANTITY_DEBOUNCE_MS)
    expect(bulk).toHaveBeenCalledWith("col-1", { items: [{ cardId: "card-1", quantity: 1 }] })
  })

  it("keeps both filter clicks made while a flush is pending", async () => {
    vi.useFakeTimers()
    setList([{ card, quantity: 3 }])
    // SAFETY: partial mock; only status/data are read.
    vi.mocked(useListCollectionFacets).mockReturnValue({
      data: {
        status: 200,
        data: {
          data: {
            expansionSets: [],
            rarities: [
              { id: "r-1", code: "RR", name: "Double Rare", available: true },
              { id: "r-2", code: "C", name: "Common", available: true },
            ],
            categories: [],
            tags: [],
          },
        },
        headers: new Headers(),
      },
    } as any)
    renderEntries()

    fireEvent.change(quantityInput(), { target: { value: "5" } })
    fireEvent.click(screen.getByRole("button", { name: "Rarity" }))
    fireEvent.click(screen.getByLabelText("Double Rare"))
    // The search prop is still the old one: the second click must build on the first.
    fireEvent.click(screen.getByLabelText("Common"))
    await advance(0)

    expect(onSearchChange).toHaveBeenCalledTimes(1)
    expect(onSearchChange).toHaveBeenCalledWith({ page: 1, rarityId: ["r-1", "r-2"] })
  })
})
