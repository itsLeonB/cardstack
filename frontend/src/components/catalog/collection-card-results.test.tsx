import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"
import { act, cleanup, fireEvent, render, screen } from "@testing-library/react"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import type * as TanStackRouter from "@tanstack/react-router"
import { CollectionCardResults } from "./collection-card-results"
import { QUANTITY_DEBOUNCE_MS } from "@/lib/use-quantity-batch"
import {
  bulkUpdateCollectionEntries,
  useListCollectionEntries,
} from "@/generated/endpoints/inventory/inventory"
import type { CardSummary } from "@/generated/models"

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
  useListCollectionEntries: vi.fn(),
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
const entries = vi.mocked(useListCollectionEntries)

function setEntries(items: { card: CardSummary; quantity: number }[]) {
  // SAFETY: partial mock; the component reads only status/data and isError.
  entries.mockReturnValue({
    isError: false,
    data: {
      status: 200,
      data: { data: items, meta: { total: items.length, page: 1, limit: 24 } },
    },
  } as any)
}

function renderResults(
  cards: CardSummary[] = [card],
  client = new QueryClient()
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
    bulk.mockReset()
    entries.mockReset()
    // SAFETY: partial response; the hook reads only status and data.data.
    bulk.mockResolvedValue({ status: 200, data: { data: [] } } as any)
  })

  it("looks up quantities only for the cards on the page", () => {
    setEntries([])
    renderResults()
    expect(entries).toHaveBeenCalledWith(
      "col-1",
      { cardId: ["card-1"], limit: 1 },
      expect.objectContaining({
        query: expect.objectContaining({
          enabled: true,
          gcTime: 0,
          refetchOnMount: "always",
        }),
      })
    )
  })

  it("caps the lookup at the endpoint's 100-id limit", () => {
    setEntries([])
    renderResults(
      Array.from({ length: 120 }, (_, index) => ({
        ...card,
        id: `card-${index}`,
      }))
    )
    const [, params] = entries.mock.calls[0]!
    expect(params?.cardId).toHaveLength(100)
    expect(params?.limit).toBe(100)
  })

  it("never fires the lookup with an empty card list", () => {
    setEntries([])
    renderResults([])
    expect(entries).toHaveBeenCalledWith(
      "col-1",
      expect.anything(),
      expect.objectContaining({
        query: expect.objectContaining({ enabled: false }),
      })
    )
  })

  it("shows the Collection quantity, 0 when the card is absent", () => {
    setEntries([{ card, quantity: 3 }])
    renderResults()
    expect(quantityInput().value).toBe("3")
    cleanup()
    setEntries([])
    renderResults()
    expect(quantityInput().value).toBe("0")
  })

  it("shows no control until the quantities have loaded", () => {
    // SAFETY: partial mock; still loading.
    entries.mockReturnValue({ isError: false, data: undefined } as any)
    renderResults()
    expect(screen.queryByLabelText("Quantity of Pikachu V")).toBeNull()
  })

  it("adds a card by increasing from 0 through one bulk call", async () => {
    vi.useFakeTimers()
    setEntries([])
    renderResults()
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
    vi.useFakeTimers()
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
    fireEvent.click(
      screen.getByRole("button", { name: "Increase quantity of Pikachu V" })
    )
    await advance(QUANTITY_DEBOUNCE_MS)
    expect(screen.getByRole("alert").textContent).toBe("Capacity exceeded")
    expect(quantityInput().value).toBe("0")
  })

  it("after a save, invalidates both this lookup and the Collection page's infinite list", async () => {
    vi.useFakeTimers()
    setEntries([])
    const client = new QueryClient()
    const invalidate = vi.spyOn(client, "invalidateQueries")
    renderResults([card], client)
    fireEvent.click(
      screen.getByRole("button", { name: "Increase quantity of Pikachu V" })
    )
    await advance(QUANTITY_DEBOUNCE_MS)

    const keys = invalidate.mock.calls.map(([filters]) => filters?.queryKey)
    expect(keys).toContainEqual(["entries", "col-1"])
    expect(keys).toContainEqual(["infinite", "entries", "col-1"])
  })
})
