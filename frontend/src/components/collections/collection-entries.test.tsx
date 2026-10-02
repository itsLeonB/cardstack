import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"
import { act, cleanup, fireEvent, render, screen } from "@testing-library/react"
import {
  QueryClient,
  QueryClientProvider,
  focusManager,
} from "@tanstack/react-query"
import type * as TanStackRouter from "@tanstack/react-router"
import { CollectionEntries } from "./collection-entries"
import { QUANTITY_DEBOUNCE_MS } from "@/lib/use-quantity-batch"
import {
  bulkUpdateCollectionEntries,
  getListCollectionEntriesInfiniteQueryKey,
  listCollectionEntries,
  useListCollectionFacets,
} from "@/generated/endpoints/inventory/inventory"
import type * as Inventory from "@/generated/endpoints/inventory/inventory"
import { useListCatalogSeries } from "@/generated/endpoints/catalog/catalog"
import type * as Catalog from "@/generated/endpoints/catalog/catalog"
import type { CardSummary, InventoryItem } from "@/generated/models"
import type { CatalogFilters } from "@/lib/catalog-search"
import { stubGridLayout } from "@/test-grid-layout"

// Only the network is faked: the generated infinite hook, the query cache and
// the quantity batch are real, so the paging and cache-patch behaviour under
// test is the app's own.
// oxlint-disable-next-line anti-slop/no-module-mocking
vi.mock(
  "@/generated/endpoints/inventory/inventory",
  async (importOriginal) => ({
    ...(await importOriginal<typeof Inventory>()),
    listCollectionEntries: vi.fn(),
    bulkUpdateCollectionEntries: vi.fn(),
    useListCollectionFacets: vi.fn(),
  })
)
// oxlint-disable-next-line anti-slop/no-module-mocking
vi.mock("@/generated/endpoints/catalog/catalog", async (importOriginal) => ({
  ...(await importOriginal<typeof Catalog>()),
  useListCatalogSeries: vi.fn(),
}))
// CardTile links via TanStack Router's `Link`, which needs a router in the tree.
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
  vi.restoreAllMocks()
  vi.unstubAllGlobals()
})

const PAGE_SIZE = 2
const list = vi.mocked(listCollectionEntries)
const bulk = vi.mocked(bulkUpdateCollectionEntries)
const onSearchChange = vi.fn()

const entry = (n: number, quantity = 3): InventoryItem => ({
  quantity,
  card: {
    id: `card-${n}`,
    name: `Card ${n}`,
    localId: String(n),
    category: "Pokémon",
    tags: [],
    illustrator: "someone",
    imageUrl: "",
    expansionSet: { id: "set-1", code: "SCE", name: "Starter", imageUrl: "" },
    rarity: { id: "r-1", code: "RR", name: "Double Rare" },
  } satisfies CardSummary,
})

// The fake server. A page is read when answered, not when requested, so a held
// request sees the changes made while it waited, as a real one would.
let server: InventoryItem[] = []

function serveEntries(items: InventoryItem[]) {
  server = items
  const gates = new Map<number, { promise: Promise<void>; open: () => void }>()
  list.mockImplementation(async (_id, params) => {
    const page = params?.page ?? 1
    // A request takes a tick, as a real one does: the grid only asks for the
    // next page after seeing the previous fetch end.
    await new Promise((resolve) => setTimeout(resolve, 1))
    await gates.get(page)?.promise
    const rows = server.slice((page - 1) * PAGE_SIZE, page * PAGE_SIZE)
    // SAFETY: partial response; the app reads status, data and meta.
    return {
      status: 200,
      data: {
        data: rows,
        meta: { total: server.length, page, limit: PAGE_SIZE },
      },
      headers: new Headers(),
    } as any
  })
  bulk.mockImplementation(async (_id, body) => {
    const results = (body.items ?? []).map(({ cardId, quantity }) => {
      server =
        quantity === 0
          ? server.filter((item) => item.card.id !== cardId)
          : server.map((item) =>
              item.card.id === cardId ? { ...item, quantity } : item
            )
      return {
        cardId,
        quantity,
        status: quantity === 0 ? "removed" : "applied",
      }
    })
    // SAFETY: partial response; the hook reads only status and data.data.
    return { status: 200, data: { data: results } } as any
  })
  return {
    /** Page `page` is requested but not answered until `release`. */
    hold(page: number) {
      let open = () => {}
      const promise = new Promise<void>((resolve) => (open = resolve))
      gates.set(page, { promise, open })
    },
    release: (page: number) => gates.get(page)?.open(),
  }
}

let queryClient: QueryClient

function renderEntries(search: CatalogFilters = {}) {
  queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  })
  return render(
    <QueryClientProvider client={queryClient}>
      <CollectionEntries
        collectionId="col-1"
        search={search}
        onSearchChange={onSearchChange}
      />
    </QueryClientProvider>
  )
}

// SAFETY: the labelled control is an <input>.
const quantityOf = (n: number) =>
  (screen.getByLabelText(`Quantity of Card ${n}`) as HTMLInputElement).value
const setQuantityOf = (n: number, value: number) =>
  fireEvent.change(screen.getByLabelText(`Quantity of Card ${n}`), {
    target: { value: String(value) },
  })
const tile = (n: number) => screen.queryAllByLabelText(`Quantity of Card ${n}`)
const loaded = (count: number, total: number) =>
  screen.getByText(`${count} of ${total} cards loaded`)

async function advance(ms: number) {
  await act(async () => {
    await vi.advanceTimersByTimeAsync(ms)
  })
}

// Each page takes a few macrotask hops (notify, render, effect, next fetch).
const settle = () => advance(50)

async function refocusTab() {
  await act(async () => {
    focusManager.setFocused(false)
    focusManager.setFocused(true)
    await vi.advanceTimersByTimeAsync(50)
  })
}

describe("CollectionEntries", () => {
  beforeEach(() => {
    vi.useFakeTimers()
    stubGridLayout()
    onSearchChange.mockReset()
    list.mockReset()
    bulk.mockReset()
    // SAFETY: partial mock; only status/data are read.
    vi.mocked(useListCatalogSeries).mockReturnValue({} as any)
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
  })

  it("shows an empty state when the Collection has no Cards", async () => {
    serveEntries([])
    renderEntries()
    await settle()
    screen.getByText(/has no Cards yet/)
  })

  it("shows a filtered empty state when filters match nothing", async () => {
    serveEntries([])
    renderEntries({ name: "zzz" })
    await settle()
    screen.getByText(/match these filters/)
  })

  it("shows the API error detail when the list fails to load", async () => {
    // SAFETY: partial response; the app reads status and data.detail.
    list.mockResolvedValue({ status: 500, data: { detail: "boom" } } as any)
    renderEntries()
    await settle()
    expect(screen.getByRole("alert").textContent).toBe("boom")
  })

  it("renders each Card as a tile with a quantity control and no Save/Remove buttons", async () => {
    serveEntries([entry(1)])
    renderEntries()
    await settle()
    expect(quantityOf(1)).toBe("3")
    expect(screen.queryByRole("button", { name: /^Save/ })).toBeNull()
    expect(screen.queryByRole("button", { name: /^Remove/ })).toBeNull()
  })

  it("requests 100 per page with the URL filters, keyed by the filters alone", async () => {
    serveEntries([entry(1)])
    renderEntries({ name: "pika" })
    await settle()
    expect(list).toHaveBeenCalledWith(
      "col-1",
      { name: "pika", limit: 100, page: 1 },
      expect.anything()
    )
    expect(
      queryClient.getQueryCache().find({
        queryKey: getListCollectionEntriesInfiniteQueryKey("col-1", {
          name: "pika",
          limit: 100,
        }),
      })
    ).toBeDefined()
  })

  it("does not keep the list cached after leaving", async () => {
    serveEntries([entry(1)])
    const view = renderEntries()
    await settle()
    expect(queryClient.getQueryCache().getAll()).toHaveLength(1)

    view.unmount()
    await advance(1)
    expect(queryClient.getQueryCache().getAll()).toHaveLength(0)
  })

  it("loads every page by scrolling, once each, without duplicates", async () => {
    serveEntries([1, 2, 3, 4, 5].map((n) => entry(n)))
    renderEntries()
    await settle()

    expect(list.mock.calls.map(([, params]) => params?.page)).toEqual([1, 2, 3])
    loaded(5, 5)
    for (const n of [1, 2, 3, 4, 5]) expect(tile(n)).toHaveLength(1)
  })

  it("shows a row that repeats across a page boundary once", async () => {
    const gate = serveEntries([1, 2, 3, 4].map((n) => entry(n)))
    gate.hold(2)
    renderEntries()
    await settle()
    // Another tab adds a card ahead of the window: page 2 now starts with card 2 again.
    server = [entry(0), ...server]
    gate.release(2)
    await settle()

    expect(tile(2)).toHaveLength(1)
    loaded(4, 5)
  })

  it("keeps a saved quantity after more pages append and after a focus refetch", async () => {
    const gate = serveEntries([1, 2, 3, 4, 5, 6].map((n) => entry(n)))
    gate.hold(3)
    renderEntries()
    await settle()
    expect(quantityOf(3)).toBe("3")

    // Card 3 is on page 2; page 3 is still loading.
    setQuantityOf(3, 5)
    await advance(QUANTITY_DEBOUNCE_MS)
    expect(bulk).toHaveBeenCalledTimes(1)
    expect(quantityOf(3)).toBe("5")

    gate.release(3)
    await settle()
    loaded(6, 6)
    expect(quantityOf(3)).toBe("5")
    expect(quantityOf(5)).toBe("3")

    await refocusTab()
    expect(quantityOf(3)).toBe("5")
    expect(quantityOf(1)).toBe("3")
  })

  it("keeps a card saved at 0 as a tile at 0 while pages append, and drops it on the next refetch", async () => {
    const gate = serveEntries([1, 2, 3, 4, 5, 6, 7, 8].map((n) => entry(n)))
    gate.hold(4)
    renderEntries()
    await settle()

    setQuantityOf(4, 0)
    await advance(QUANTITY_DEBOUNCE_MS)
    expect(quantityOf(4)).toBe("0")

    gate.release(4)
    await settle()
    // The server shifted up by one, so page 4 starts after card 7: card 7 is
    // skipped until a refetch (accepted, see ticket 36).
    expect(tile(7)).toHaveLength(0)
    expect(quantityOf(4)).toBe("0")
    expect(quantityOf(3)).toBe("3")

    await refocusTab()
    expect(tile(4)).toHaveLength(0)
    expect(tile(7)).toHaveLength(1)
    loaded(7, 7)
  })

  it("keeps a declined card's error when a saved card in the same batch patches the cache", async () => {
    serveEntries([entry(1), entry(2)])
    // SAFETY: partial response; the hook reads only status and data.data.
    bulk.mockResolvedValue({
      status: 200,
      data: {
        data: [
          { cardId: "card-1", quantity: 5, status: "applied" },
          {
            cardId: "card-2",
            quantity: 3,
            status: "declined",
            reason: "capacity_exceeded",
            message: "Capacity exceeded",
          },
        ],
      },
    } as any)
    renderEntries()
    await settle()

    setQuantityOf(1, 5)
    setQuantityOf(2, 50)
    await advance(QUANTITY_DEBOUNCE_MS)

    expect(screen.getByRole("alert").textContent).toBe("Capacity exceeded")
    expect(quantityOf(1)).toBe("5")
    expect(quantityOf(2)).toBe("3")
  })

  it("refetches on window focus only while no quantity edits are pending or saving", async () => {
    serveEntries([entry(1)])
    renderEntries()
    await settle()
    expect(list).toHaveBeenCalledTimes(1)

    // A pending edit holds the refetch off.
    fireEvent.click(
      screen.getByRole("button", { name: "Increase quantity of Card 1" })
    )
    await refocusTab()
    expect(list).toHaveBeenCalledTimes(1)

    // So does a save in flight.
    let finish: () => void = () => {}
    bulk.mockReturnValueOnce(
      new Promise((resolve) => {
        finish = () =>
          // SAFETY: partial response; the hook reads only status and data.data.
          resolve({
            status: 200,
            data: {
              data: [{ cardId: "card-1", quantity: 4, status: "applied" }],
            },
          } as any)
      })
    )
    await advance(QUANTITY_DEBOUNCE_MS)
    await refocusTab()
    expect(list).toHaveBeenCalledTimes(1)

    finish()
    await settle()
    expect(list).toHaveBeenCalledTimes(1)
    await refocusTab()
    expect(list).toHaveBeenCalledTimes(2)
  })

  it("lets an older failed batch neither revert nor unprotect a newer, already-sent edit", async () => {
    serveEntries([entry(1)])
    renderEntries()
    await settle()
    let failFirst: () => void = () => {}
    bulk.mockReturnValueOnce(
      new Promise(
        (_, reject) => (failFirst = () => reject(new Error("offline")))
      )
    )
    bulk.mockReturnValueOnce(new Promise(() => {}))

    fireEvent.click(
      screen.getByRole("button", { name: "Increase quantity of Card 1" })
    )
    await advance(QUANTITY_DEBOUNCE_MS)
    fireEvent.click(
      screen.getByRole("button", { name: "Increase quantity of Card 1" })
    )
    await advance(QUANTITY_DEBOUNCE_MS)
    expect(bulk).toHaveBeenCalledTimes(1)

    failFirst()
    await settle()

    expect(bulk).toHaveBeenCalledTimes(2)
    expect(quantityOf(1)).toBe("5")
    await refocusTab()
    expect(list).toHaveBeenCalledTimes(1)
  })

  it("applies +, - and typed quantities optimistically, then sends one bulk call after the debounce", async () => {
    serveEntries([entry(1)])
    renderEntries()
    await settle()

    fireEvent.click(
      screen.getByRole("button", { name: "Increase quantity of Card 1" })
    )
    expect(quantityOf(1)).toBe("4")
    fireEvent.click(
      screen.getByRole("button", { name: "Decrease quantity of Card 1" })
    )
    setQuantityOf(1, 7)
    expect(quantityOf(1)).toBe("7")

    await advance(QUANTITY_DEBOUNCE_MS - 1)
    expect(bulk).not.toHaveBeenCalled()
    await advance(1)
    expect(bulk).toHaveBeenCalledTimes(1)
    expect(bulk).toHaveBeenCalledWith("col-1", {
      items: [{ cardId: "card-1", quantity: 7 }],
    })
  })

  it("raises a card at 0 in place with +", async () => {
    serveEntries([entry(1, 1)])
    renderEntries()
    await settle()

    fireEvent.click(
      screen.getByRole("button", { name: "Decrease quantity of Card 1" })
    )
    await advance(QUANTITY_DEBOUNCE_MS)
    expect(quantityOf(1)).toBe("0")
    expect(
      screen
        .getByRole("button", { name: "Decrease quantity of Card 1" })
        .hasAttribute("disabled")
    ).toBe(true)

    fireEvent.click(
      screen.getByRole("button", { name: "Increase quantity of Card 1" })
    )
    expect(quantityOf(1)).toBe("1")
    await advance(QUANTITY_DEBOUNCE_MS)
    expect(bulk).toHaveBeenLastCalledWith("col-1", {
      items: [{ cardId: "card-1", quantity: 1 }],
    })
  })

  it("reverts a capacity-declined card with an error tied to it", async () => {
    serveEntries([entry(1)])
    // SAFETY: partial response; the hook reads only status and data.data.
    bulk.mockResolvedValue({
      status: 200,
      data: {
        data: [
          {
            cardId: "card-1",
            quantity: 3,
            status: "declined",
            reason: "capacity_exceeded",
            message: "Capacity exceeded",
          },
        ],
      },
    } as any)
    renderEntries()
    await settle()

    setQuantityOf(1, 50)
    await advance(QUANTITY_DEBOUNCE_MS)

    expect(screen.getByRole("alert").textContent).toBe("Capacity exceeded")
    expect(quantityOf(1)).toBe("3")
  })

  it("flushes pending edits before changing a filter", async () => {
    serveEntries([entry(1)])
    renderEntries()
    await settle()

    setQuantityOf(1, 5)
    fireEvent.click(screen.getByRole("button", { name: "Rarity" }))
    fireEvent.click(screen.getByLabelText("Double Rare"))

    // The batch goes out without waiting for the debounce, and navigation follows it.
    await settle()
    expect(bulk).toHaveBeenCalledWith("col-1", {
      items: [{ cardId: "card-1", quantity: 5 }],
    })
    expect(onSearchChange).toHaveBeenCalledWith({ rarityId: ["r-1"] })
  })

  it("keeps both filter clicks made while a flush is pending", async () => {
    serveEntries([entry(1)])
    renderEntries()
    await settle()

    setQuantityOf(1, 5)
    fireEvent.click(screen.getByRole("button", { name: "Rarity" }))
    fireEvent.click(screen.getByLabelText("Double Rare"))
    // The search prop is still the old one: the second click must build on the first.
    fireEvent.click(screen.getByLabelText("Common"))
    await settle()

    expect(onSearchChange).toHaveBeenCalledTimes(1)
    expect(onSearchChange).toHaveBeenCalledWith({ rarityId: ["r-1", "r-2"] })
  })
})
