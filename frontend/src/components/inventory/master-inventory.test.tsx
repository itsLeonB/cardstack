import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"
import { act, cleanup, fireEvent, render, screen } from "@testing-library/react"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import type * as TanStackRouter from "@tanstack/react-router"
import { MasterInventory } from "./master-inventory"
import {
  listMasterInventory,
  useListMasterInventoryFacets,
} from "@/generated/endpoints/inventory/inventory"
import type * as Inventory from "@/generated/endpoints/inventory/inventory"
import { useListCatalogSeries } from "@/generated/endpoints/catalog/catalog"
import type * as Catalog from "@/generated/endpoints/catalog/catalog"
import type { CardSummary, InventoryItem } from "@/generated/models"
import type { CatalogFilters } from "@/lib/catalog-search"
import { stubGridLayout } from "@/test-grid-layout"

// Only the network is faked: the generated infinite hook and the query cache are real.
// oxlint-disable-next-line anti-slop/no-module-mocking
vi.mock(
  "@/generated/endpoints/inventory/inventory",
  async (importOriginal) => ({
    ...(await importOriginal<typeof Inventory>()),
    listMasterInventory: vi.fn(),
    useListMasterInventoryFacets: vi.fn(),
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
    Link: ({ children, params, to, ...props }: any) => (
      <a
        href={to
          .replace("$expansionSetId", params?.expansionSetId)
          .replace("$localId", params?.localId)}
        {...props}
      >
        {children}
      </a>
    ),
  }
})

afterEach(() => {
  cleanup()
  vi.useRealTimers()
  vi.restoreAllMocks()
  vi.unstubAllGlobals()
})

const list = vi.mocked(listMasterInventory)

const entry = (n: number, quantity = 1): InventoryItem => ({
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

/**
 * Serves `items` two to a page. A request takes a tick, as a real one does:
 * the grid only asks for the next page after seeing the previous fetch end.
 */
function serve(items: InventoryItem[]) {
  list.mockImplementation(async (params) => {
    const page = params?.page ?? 1
    await new Promise((resolve) => setTimeout(resolve, 1))
    // SAFETY: partial response; the app reads status, data and meta.
    return {
      status: 200,
      data: {
        data: items.slice((page - 1) * 2, page * 2),
        meta: { total: items.length, page, limit: 2 },
      },
      headers: new Headers(),
    } as any
  })
}

let queryClient: QueryClient
const onSearchChange = vi.fn()

function inventory(search: CatalogFilters) {
  return (
    <QueryClientProvider client={queryClient}>
      <MasterInventory search={search} onSearchChange={onSearchChange} />
    </QueryClientProvider>
  )
}

function renderInventory(search: CatalogFilters = {}) {
  queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  })
  return render(inventory(search))
}

async function settle() {
  await act(async () => {
    await vi.advanceTimersByTimeAsync(50)
  })
}

describe("MasterInventory", () => {
  beforeEach(() => {
    vi.useFakeTimers()
    stubGridLayout()
    onSearchChange.mockReset()
    list.mockReset()
    // SAFETY: partial mock; only status/data are read.
    vi.mocked(useListCatalogSeries).mockReturnValue({} as any)
    // SAFETY: partial mock; only status/data are read.
    vi.mocked(useListMasterInventoryFacets).mockReturnValue({
      data: {
        status: 200,
        data: {
          data: {
            expansionSets: [],
            rarities: [
              { id: "r-1", code: "RR", name: "Double Rare", available: true },
            ],
            categories: [],
            tags: [],
          },
        },
      },
    } as any)
  })

  it("shows each owned Card with its total quantity, linking to the card detail page", async () => {
    serve([entry(1, 5)])
    renderInventory()
    await settle()
    expect(screen.getByText("×5")).toBeTruthy()
    expect(
      screen.getByRole("link", { name: "Card 1" }).getAttribute("href")
    ).toBe("/catalog/cards/set-1/1")
    expect(screen.queryByLabelText("Quantity of Card 1")).toBeNull()
  })

  it("shows an empty state when the user owns nothing", async () => {
    serve([])
    renderInventory()
    await settle()
    expect(screen.getByText(/don.t own any Cards yet/)).toBeTruthy()
  })

  it("shows the API error detail for a non-200 response", async () => {
    // SAFETY: partial response; the app reads status and data.detail.
    list.mockResolvedValue({ status: 500, data: { detail: "boom" } } as any)
    renderInventory()
    await settle()
    expect(screen.getByRole("alert").textContent).toBe("boom")
  })

  it("scroll-loads every page of 100 once each, without duplicates", async () => {
    serve([1, 2, 3, 4, 5].map((n) => entry(n, n)))
    renderInventory({ rarityId: ["r-1"] })
    await settle()
    await settle()

    expect(list.mock.calls.map(([params]) => params)).toEqual([
      { rarityId: ["r-1"], limit: 100, page: 1 },
      { rarityId: ["r-1"], limit: 100, page: 2 },
      { rarityId: ["r-1"], limit: 100, page: 3 },
    ])
    screen.getByText("5 of 5 cards loaded")
    for (const n of [1, 2, 3, 4, 5]) {
      expect(screen.getAllByText(`×${n}`)).toHaveLength(1)
    }
  })

  it("asks the facets for the URL filters", async () => {
    serve([])
    renderInventory({ rarityId: ["r-1"] })
    await settle()
    expect(vi.mocked(useListMasterInventoryFacets).mock.lastCall?.[0]).toEqual({
      rarityId: ["r-1"],
    })
  })

  it("uses the facets for the filter options and passes the changed filters up", async () => {
    serve([entry(1)])
    renderInventory({ name: "pika" })
    await settle()
    fireEvent.click(screen.getByRole("button", { name: "Rarity" }))
    fireEvent.click(screen.getByLabelText("Double Rare"))
    expect(onSearchChange).toHaveBeenCalledWith({
      name: "pika",
      rarityId: ["r-1"],
    })
  })

  it("shows a no-results state when filters match nothing", async () => {
    serve([])
    renderInventory({ rarityId: ["r-1"] })
    await settle()
    expect(screen.getByText(/match these filters/)).toBeTruthy()
    expect(screen.queryByText(/don.t own any Cards yet/)).toBeNull()
  })

  it("keeps the pages cached after leaving, but refetches them on return", async () => {
    serve([1, 2, 3].map((n) => entry(n)))
    const view = renderInventory()
    await settle()
    expect(list).toHaveBeenCalledTimes(2)

    view.unmount()
    await settle()
    expect(queryClient.getQueryCache().getAll()).toHaveLength(1)

    render(inventory({}))
    // The cached pages show at once (scroll restoration needs them), then refresh.
    screen.getByText("3 of 3 cards loaded")
    await settle()
    expect(list.mock.calls.length).toBeGreaterThan(2)
  })
})
