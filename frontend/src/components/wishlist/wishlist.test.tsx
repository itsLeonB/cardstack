import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"
import { act, cleanup, fireEvent, render, screen } from "@testing-library/react"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import type * as TanStackRouter from "@tanstack/react-router"
import { Wishlist } from "./wishlist"
import { useListCatalogSeries } from "@/generated/endpoints/catalog/catalog"
import type * as Catalog from "@/generated/endpoints/catalog/catalog"
import type * as WishlistApi from "@/generated/endpoints/wishlist/wishlist"
import type { CardSummary, WishlistItem } from "@/generated/models"
import type { WishlistFilters } from "@/lib/wishlist"
import { stubGridLayout } from "@/test-grid-layout"

// Only the network is faked. The list query keeps its real generated infinite
// hook; the mutation hooks are replaced by the smallest fake that delivers the
// canned answer through the per-call callbacks the real ones invoke, because
// the generated hook and its request live in the same module (mocking the
// request alone would not reach it).
const mocks = vi.hoisted(() => {
  // SAFETY: partial responses; the component reads status and data only.
  const responses = {
    set: { status: 204, data: undefined } as any,
    remove: { status: 204, data: undefined } as any,
    clear: { status: 200, data: { data: { removed: 1 } } } as any,
  }
  const fakeHook =
    <Variables, Response>(
      spy: (variables: Variables) => void,
      answer: () => Response
    ) =>
    () => ({
      isPending: false,
      mutate: (
        variables: Variables,
        callbacks?: { onSuccess?: (response: Response) => void }
      ) => {
        spy(variables)
        void Promise.resolve(answer()).then((response) =>
          callbacks?.onSuccess?.(response)
        )
      },
    })
  return {
    responses,
    list: vi.fn(),
    facets: vi.fn(),
    setItem: vi.fn(),
    removeItem: vi.fn(),
    clear: vi.fn(),
    fakeHook,
  }
})

// oxlint-disable-next-line anti-slop/no-module-mocking
vi.mock("@/generated/endpoints/wishlist/wishlist", async (importOriginal) => ({
  ...(await importOriginal<typeof WishlistApi>()),
  listWishlist: mocks.list,
  useListWishlistFacets: mocks.facets,
  useSetWishlistItem: mocks.fakeHook(mocks.setItem, () => mocks.responses.set),
  useRemoveWishlistItem: mocks.fakeHook(
    mocks.removeItem,
    () => mocks.responses.remove
  ),
  useClearFulfilledWishlist: mocks.fakeHook(
    mocks.clear,
    () => mocks.responses.clear
  ),
}))

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

const list = mocks.list

const item = (n: number, owned: number, wanted: number): WishlistItem => ({
  fulfilled: owned >= wanted,
  ownedQuantity: owned,
  wantedQuantity: wanted,
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
function serve(items: WishlistItem[]) {
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

function renderWishlist(search: WishlistFilters = {}) {
  queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  })
  return render(
    <QueryClientProvider client={queryClient}>
      <Wishlist search={search} onSearchChange={onSearchChange} />
    </QueryClientProvider>
  )
}

async function settle() {
  await act(async () => {
    await vi.advanceTimersByTimeAsync(50)
  })
}

describe("Wishlist", () => {
  beforeEach(() => {
    vi.useFakeTimers()
    stubGridLayout()
    onSearchChange.mockReset()
    list.mockReset()
    mocks.setItem.mockReset()
    mocks.removeItem.mockReset()
    mocks.clear.mockReset()
    mocks.responses.set = { status: 204, data: undefined }
    mocks.responses.remove = { status: 204, data: undefined }
    mocks.responses.clear = { status: 200, data: { data: { removed: 1 } } }
    // SAFETY: partial mock; only status/data are read.
    vi.mocked(useListCatalogSeries).mockReturnValue({} as any)
    // SAFETY: partial mock; only status/data are read.
    mocks.facets.mockReturnValue({
      data: {
        status: 200,
        data: {
          data: {
            expansionSets: [],
            rarities: [],
            categories: [],
            tags: [],
          },
        },
      },
    })
  })

  it("shows owned of wanted on every item, and Fulfilled as a word", async () => {
    serve([item(1, 2, 2), item(2, 1, 3)])
    renderWishlist()
    await settle()

    expect(screen.getByText("2 / 2")).toBeTruthy()
    expect(screen.getByText("2 owned of 2 wanted")).toBeTruthy()
    expect(screen.getByText("1 / 3")).toBeTruthy()
    expect(screen.getByText("1 owned of 3 wanted")).toBeTruthy()
    // The status filter's button, plus the one fulfilled item's word.
    expect(screen.getAllByText("Fulfilled")).toHaveLength(2)
  })

  it("shows an empty state when nothing is wanted", async () => {
    serve([])
    renderWishlist()
    await settle()
    expect(screen.getByText(/Wishlist is empty/)).toBeTruthy()
  })

  it("marks the current status and asks for the chosen one", async () => {
    serve([item(1, 1, 1)])
    renderWishlist({ name: "pika", status: "fulfilled" })
    await settle()

    expect(
      screen
        .getByRole("button", { name: "Fulfilled" })
        .getAttribute("aria-pressed")
    ).toBe("true")
    expect(
      screen.getByRole("button", { name: "All" }).getAttribute("aria-pressed")
    ).toBe("false")

    fireEvent.click(screen.getByRole("button", { name: "Still needed" }))
    expect(onSearchChange).toHaveBeenCalledWith({
      name: "pika",
      status: "needed",
    })
  })

  it("asks the API for the status filter", async () => {
    serve([])
    renderWishlist({ status: "needed" })
    await settle()
    expect(list.mock.lastCall?.[0]).toMatchObject({ status: "needed" })
  })

  it("clears every fulfilled item in one call and announces the count", async () => {
    serve([item(1, 2, 2), item(2, 1, 3)])
    renderWishlist()
    await settle()

    const before = list.mock.calls.length
    fireEvent.click(screen.getByRole("button", { name: "Clear fulfilled" }))
    await settle()

    expect(mocks.clear).toHaveBeenCalledTimes(1)
    expect(screen.getByRole("status").textContent).toBe(
      "Cleared 1 fulfilled Card."
    )
    // The list is refetched rather than patched: status is derived on read.
    expect(list.mock.calls.length).toBeGreaterThan(before)
  })

  it("shows the API error when Clear fulfilled is refused", async () => {
    serve([item(1, 2, 2)])
    mocks.responses.clear = { status: 500, data: { detail: "boom" } }
    renderWishlist()
    await settle()

    fireEvent.click(screen.getByRole("button", { name: "Clear fulfilled" }))
    await settle()

    expect(screen.getByRole("alert").textContent).toBe("boom")
  })

  it("replaces a wanted quantity one step at a time", async () => {
    serve([item(1, 0, 2)])
    renderWishlist()
    await settle()

    fireEvent.click(
      screen.getByRole("button", {
        name: "Increase wanted quantity of Card 1",
      })
    )
    await settle()

    expect(mocks.setItem).toHaveBeenCalledWith({
      cardId: "card-1",
      data: { quantity: 3 },
    })
    expect(screen.getByRole("status").textContent).toBe("Want 3 of Card 1.")
    expect(screen.getByText("0 / 3")).toBeTruthy()
  })

  it("never wants less than one", async () => {
    serve([item(1, 0, 1)])
    renderWishlist()
    await settle()

    expect(
      screen.getByRole<HTMLButtonElement>("button", {
        name: "Decrease wanted quantity of Card 1",
      }).disabled
    ).toBe(true)
  })

  it("removes an item and announces it", async () => {
    serve([item(1, 1, 2)])
    renderWishlist()
    await settle()

    fireEvent.click(
      screen.getByRole("button", {
        name: "Remove Card 1 from your Wishlist",
      })
    )
    await settle()

    expect(mocks.removeItem).toHaveBeenCalledWith({ cardId: "card-1" })
    expect(screen.getByRole("status").textContent).toBe(
      "Removed Card 1 from your Wishlist."
    )
  })

  it("surfaces a refused quantity change inline, on the item", async () => {
    serve([item(1, 0, 2)])
    mocks.responses.set = { status: 500, data: { detail: "nope" } }
    renderWishlist()
    await settle()

    fireEvent.click(
      screen.getByRole("button", {
        name: "Increase wanted quantity of Card 1",
      })
    )
    await settle()

    expect(screen.getByRole("alert").textContent).toBe("nope")
    expect(screen.getByText("0 / 2")).toBeTruthy()
  })
})
