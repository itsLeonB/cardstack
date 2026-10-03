import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"
import { act, cleanup, fireEvent, render, screen } from "@testing-library/react"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import type * as TanStackRouter from "@tanstack/react-router"
import { CollectionCardResults } from "./collection-card-results"
import { QUANTITY_DEBOUNCE_MS } from "@/lib/use-quantity-batch"
import {
  bulkUpdateCollectionEntries,
  getListCollectionEntriesQueryOptions,
} from "@/generated/endpoints/inventory/inventory"
import type {
  CardSummary,
  ListCollectionEntriesParams,
} from "@/generated/models"

const lookup = vi.hoisted(() => vi.fn())

// Isolates the UI from the network.
// oxlint-disable-next-line anti-slop/no-module-mocking
vi.mock("@/generated/endpoints/inventory/inventory", () => ({
  getListCollectionEntriesQueryKey: (id: string) => ["entries", id],
  getListCollectionEntriesInfiniteQueryKey: (id: string) => [
    "infinite",
    "entries",
    id,
  ],
  getListMasterInventoryQueryKey: () => ["inventory"],
  getListMasterInventoryInfiniteQueryKey: () => ["infinite", "inventory"],
  getListMasterInventoryFacetsQueryKey: () => ["inventory-facets"],
  getListCollectionFacetsQueryKey: (id: string) => ["facets", id],
  getListCollectionEntriesQueryOptions: vi.fn(
    (
      id: string,
      params: ListCollectionEntriesParams,
      options?: { query?: object }
    ) => ({
      queryKey: ["entries", id, params],
      queryFn: () => lookup(params),
      ...options?.query,
    })
  ),
  bulkUpdateCollectionEntries: vi.fn(),
}))
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
  expansionSet: { id: "set-1", code: "SCE", name: "Starter", imageUrl: "" },
  rarity: { id: "r-1", code: "RR", name: "Double Rare" },
}

const bulk = vi.mocked(bulkUpdateCollectionEntries)
const entries = vi.mocked(getListCollectionEntriesQueryOptions)

function setEntries(items: { card: CardSummary; quantity: number }[]) {
  lookup.mockResolvedValue({
    status: 200,
    data: { data: items, meta: { total: items.length, page: 1, limit: 24 } },
  })
}

async function settle() {
  await act(async () => {
    await vi.advanceTimersByTimeAsync(0)
  })
}

function renderResults(
  cards: CardSummary[] = [card],
  client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
) {
  render(
    <QueryClientProvider client={client}>
      <CollectionCardResults
        collectionId="col-1"
        cards={cards}
        total={cards.length}
        isPending={false}
        isError={false}
        emptyMessage="none"
        hasNextPage={false}
        isFetching={false}
        onLoadMore={vi.fn()}
      />
    </QueryClientProvider>
  )
}

// SAFETY: the labelled control is an <input>.
const quantityInput = () =>
  screen.getByLabelText("Quantity of Pikachu V") as HTMLInputElement

async function advance(ms: number) {
  await act(async () => {
    await vi.advanceTimersByTimeAsync(ms)
  })
}

describe("CollectionCardResults", () => {
  beforeEach(() => {
    vi.useFakeTimers()
    bulk.mockReset()
    entries.mockClear()
    lookup.mockReset()
    // SAFETY: partial response; the hook reads only status and data.data.
    bulk.mockResolvedValue({ status: 200, data: { data: [] } } as any)
  })

  it("looks up quantities only for the cards on the page", async () => {
    setEntries([])
    renderResults()
    await settle()
    expect(entries).toHaveBeenCalledWith(
      "col-1",
      { cardId: ["card-1"], limit: 1 },
      expect.objectContaining({
        query: expect.objectContaining({
          gcTime: 0,
          refetchOnMount: "always",
        }),
      })
    )
  })

  it("looks up each page's worth of cards separately, none over the endpoint's 100-id limit", async () => {
    setEntries([])
    renderResults(
      Array.from({ length: 130 }, (_, index) => ({
        ...card,
        id: `card-${index}`,
      }))
    )
    await settle()
    const sizes = lookup.mock.calls.map(([params]) => params.cardId.length)
    expect(sizes).toEqual([60, 60, 10])
    for (const [params] of lookup.mock.calls)
      expect(params.limit).toBe(params.cardId.length)
  })

  it("never fires the lookup with an empty card list", async () => {
    setEntries([])
    renderResults([])
    await settle()
    expect(lookup).not.toHaveBeenCalled()
  })

  it("shows the Collection quantity, 0 when the card is absent", async () => {
    setEntries([{ card, quantity: 3 }])
    renderResults()
    await settle()
    expect(quantityInput().value).toBe("3")
    cleanup()
    setEntries([])
    renderResults()
    await settle()
    expect(quantityInput().value).toBe("0")
  })

  it("shows no control until the quantities have loaded", async () => {
    lookup.mockReturnValue(new Promise(() => {}))
    renderResults()
    await settle()
    expect(screen.queryByLabelText("Quantity of Pikachu V")).toBeNull()
  })

  it("shows an alert and no control when the lookup fails", async () => {
    lookup.mockRejectedValue(new Error("offline"))
    renderResults()
    await settle()
    expect(screen.getByRole("alert").textContent).toContain(
      "Could not load this Collection"
    )
    expect(screen.queryByLabelText("Quantity of Pikachu V")).toBeNull()
  })

  it("adds a card by increasing from 0 through one bulk call", async () => {
    setEntries([])
    renderResults()
    await settle()
    fireEvent.click(
      screen.getByRole("button", { name: "Increase quantity of Pikachu V" })
    )
    expect(quantityInput().value).toBe("1")
    await advance(QUANTITY_DEBOUNCE_MS)
    expect(bulk).toHaveBeenCalledWith("col-1", {
      items: [{ cardId: "card-1", quantity: 1 }],
    })
  })

  it("reverts a capacity-declined addition with an error", async () => {
    setEntries([])
    // SAFETY: partial response; the hook reads only status and data.data.
    bulk.mockResolvedValue({
      status: 200,
      data: {
        data: [
          {
            cardId: "card-1",
            quantity: 0,
            status: "declined",
            reason: "capacity_exceeded",
            message: "Capacity exceeded",
          },
        ],
      },
    } as any)
    renderResults()
    await settle()
    fireEvent.click(
      screen.getByRole("button", { name: "Increase quantity of Pikachu V" })
    )
    await advance(QUANTITY_DEBOUNCE_MS)
    expect(screen.getByRole("alert").textContent).toBe("Capacity exceeded")
    expect(quantityInput().value).toBe("0")
  })

  it("after a save, invalidates both this lookup and the Collection page's infinite list", async () => {
    setEntries([])
    const client = new QueryClient()
    const invalidate = vi.spyOn(client, "invalidateQueries")
    renderResults([card], client)
    await settle()
    fireEvent.click(
      screen.getByRole("button", { name: "Increase quantity of Pikachu V" })
    )
    await advance(QUANTITY_DEBOUNCE_MS)

    const keys = invalidate.mock.calls.map(([filters]) => filters?.queryKey)
    expect(keys).toContainEqual(["entries", "col-1"])
    expect(keys).toContainEqual(["infinite", "entries", "col-1"])
  })
})
