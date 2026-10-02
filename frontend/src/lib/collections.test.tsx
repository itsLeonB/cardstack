import { describe, expect, it, vi, beforeEach } from "vitest"
import { renderHook } from "@testing-library/react"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import type { ReactNode } from "react"
import type * as CollectionsModule from "@/generated/endpoints/collections/collections"
import {
  useCreateCollection,
  useUpdateCollection,
  useDeleteCollection,
  getGetCollectionQueryKey,
  getListCollectionsQueryKey,
} from "@/generated/endpoints/collections/collections"
import type {
  createCollectionResponse,
  updateCollectionResponse,
  deleteCollectionResponse,
} from "@/generated/endpoints/collections/collections"
import {
  getListCollectionEntriesInfiniteQueryKey,
  getListCollectionEntriesQueryKey,
} from "@/generated/endpoints/inventory/inventory"
import type { InfiniteData } from "@tanstack/react-query"
import type { InventoryItem } from "@/generated/models"
import type { listCollectionEntries } from "@/generated/endpoints/inventory/inventory"
import {
  invalidateCollectionCounts,
  invalidateCollectionEntries,
  patchCollectionEntryQuantities,
  useCreateCollectionMutation,
  useUpdateCollectionMutation,
  useDeleteCollectionMutation,
} from "./collections"

// The collections endpoints are generated orval/TanStack Query hooks with no
// service layer to inject; mocking the generated module is the standard way
// to isolate these wrappers from it in tests (see session.test.tsx).
// oxlint-disable-next-line anti-slop/no-module-mocking
vi.mock("@/generated/endpoints/collections/collections", async () => {
  const actual = await vi.importActual<typeof CollectionsModule>(
    "@/generated/endpoints/collections/collections"
  )
  return {
    ...actual,
    useCreateCollection: vi.fn(),
    useUpdateCollection: vi.fn(),
    useDeleteCollection: vi.fn(),
  }
})

const mockUseCreateCollection = vi.mocked(useCreateCollection)
const mockUseUpdateCollection = vi.mocked(useUpdateCollection)
const mockUseDeleteCollection = vi.mocked(useDeleteCollection)

describe("useCreateCollectionMutation", () => {
  beforeEach(() => {
    mockUseCreateCollection.mockReset()
  })

  it("invalidates the collections list once creation succeeds (201)", () => {
    let capturedOnSuccess:
      ((response: createCollectionResponse) => void) | undefined
    mockUseCreateCollection.mockImplementation((options) => {
      // SAFETY: mutation.onSuccess is the only field this wrapper reads.
      capturedOnSuccess = options?.mutation?.onSuccess as never
      // SAFETY: partial mock; only mutation.onSuccess is exercised here.
      return {} as any
    })

    const queryClient = new QueryClient()
    const invalidateSpy = vi.spyOn(queryClient, "invalidateQueries")

    function Wrapper({ children }: { children: ReactNode }) {
      return (
        <QueryClientProvider client={queryClient}>
          {children}
        </QueryClientProvider>
      )
    }

    renderHook(() => useCreateCollectionMutation(), { wrapper: Wrapper })

    capturedOnSuccess?.({
      status: 201,
      data: {
        data: {
          id: "1",
          title: "Binder",
          description: "",
          maxCardCount: 0,
          cardCount: 0,
        },
      },
      headers: new Headers(),
    })

    expect(invalidateSpy).toHaveBeenCalledWith({
      queryKey: getListCollectionsQueryKey(),
    })
  })

  it("does not invalidate on a failed create", () => {
    let capturedOnSuccess:
      ((response: createCollectionResponse) => void) | undefined
    mockUseCreateCollection.mockImplementation((options) => {
      // SAFETY: mutation.onSuccess is the only field this wrapper reads.
      capturedOnSuccess = options?.mutation?.onSuccess as never
      // SAFETY: partial mock; only mutation.onSuccess is exercised here.
      return {} as any
    })

    const queryClient = new QueryClient()
    const invalidateSpy = vi.spyOn(queryClient, "invalidateQueries")

    function Wrapper({ children }: { children: ReactNode }) {
      return (
        <QueryClientProvider client={queryClient}>
          {children}
        </QueryClientProvider>
      )
    }

    renderHook(() => useCreateCollectionMutation(), { wrapper: Wrapper })

    capturedOnSuccess?.({
      status: 422,
      data: { detail: "Title is required" },
      headers: new Headers(),
    })

    expect(invalidateSpy).not.toHaveBeenCalled()
  })
})

describe("useUpdateCollectionMutation", () => {
  beforeEach(() => {
    mockUseUpdateCollection.mockReset()
  })

  it("invalidates the collections list once the update succeeds (200)", () => {
    let capturedOnSuccess:
      | ((
          response: updateCollectionResponse,
          variables: { id: string }
        ) => void)
      | undefined
    mockUseUpdateCollection.mockImplementation((options) => {
      // SAFETY: mutation.onSuccess is the only field this wrapper reads.
      capturedOnSuccess = options?.mutation?.onSuccess as never
      // SAFETY: partial mock; only mutation.onSuccess is exercised here.
      return {} as any
    })

    const queryClient = new QueryClient()
    const invalidateSpy = vi.spyOn(queryClient, "invalidateQueries")

    function Wrapper({ children }: { children: ReactNode }) {
      return (
        <QueryClientProvider client={queryClient}>
          {children}
        </QueryClientProvider>
      )
    }

    renderHook(() => useUpdateCollectionMutation(), { wrapper: Wrapper })

    capturedOnSuccess?.(
      {
        status: 200,
        data: {
          data: {
            id: "1",
            title: "Binder",
            description: "",
            maxCardCount: 0,
            cardCount: 0,
          },
        },
        headers: new Headers(),
      },
      { id: "1" }
    )

    expect(invalidateSpy).toHaveBeenCalledWith({
      queryKey: getListCollectionsQueryKey(),
    })
    expect(invalidateSpy).toHaveBeenCalledWith({
      queryKey: getGetCollectionQueryKey("1"),
    })
  })

  it("does not invalidate on a failed update", () => {
    let capturedOnSuccess:
      | ((
          response: updateCollectionResponse,
          variables: { id: string }
        ) => void)
      | undefined
    mockUseUpdateCollection.mockImplementation((options) => {
      // SAFETY: mutation.onSuccess is the only field this wrapper reads.
      capturedOnSuccess = options?.mutation?.onSuccess as never
      // SAFETY: partial mock; only mutation.onSuccess is exercised here.
      return {} as any
    })

    const queryClient = new QueryClient()
    const invalidateSpy = vi.spyOn(queryClient, "invalidateQueries")

    function Wrapper({ children }: { children: ReactNode }) {
      return (
        <QueryClientProvider client={queryClient}>
          {children}
        </QueryClientProvider>
      )
    }

    renderHook(() => useUpdateCollectionMutation(), { wrapper: Wrapper })

    capturedOnSuccess?.(
      {
        status: 404,
        data: { detail: "Not found" },
        headers: new Headers(),
      },
      { id: "1" }
    )

    expect(invalidateSpy).not.toHaveBeenCalled()
  })
})

describe("useDeleteCollectionMutation", () => {
  beforeEach(() => {
    mockUseDeleteCollection.mockReset()
  })

  it("invalidates the collections list once the delete succeeds (204)", () => {
    let capturedOnSuccess:
      | ((
          response: deleteCollectionResponse,
          variables: { id: string }
        ) => void)
      | undefined
    mockUseDeleteCollection.mockImplementation((options) => {
      // SAFETY: mutation.onSuccess is the only field this wrapper reads.
      capturedOnSuccess = options?.mutation?.onSuccess as never
      // SAFETY: partial mock; only mutation.onSuccess is exercised here.
      return {} as any
    })

    const queryClient = new QueryClient()
    const invalidateSpy = vi.spyOn(queryClient, "invalidateQueries")
    const removeSpy = vi.spyOn(queryClient, "removeQueries")

    function Wrapper({ children }: { children: ReactNode }) {
      return (
        <QueryClientProvider client={queryClient}>
          {children}
        </QueryClientProvider>
      )
    }

    renderHook(() => useDeleteCollectionMutation(), { wrapper: Wrapper })

    capturedOnSuccess?.(
      {
        status: 204,
        data: undefined,
        headers: new Headers(),
      },
      { id: "1" }
    )

    expect(invalidateSpy).toHaveBeenCalledWith({
      queryKey: getListCollectionsQueryKey(),
    })
    expect(removeSpy).toHaveBeenCalledWith({
      queryKey: getGetCollectionQueryKey("1"),
    })
  })

  it("does not invalidate on a failed delete", () => {
    let capturedOnSuccess:
      | ((
          response: deleteCollectionResponse,
          variables: { id: string }
        ) => void)
      | undefined
    mockUseDeleteCollection.mockImplementation((options) => {
      // SAFETY: mutation.onSuccess is the only field this wrapper reads.
      capturedOnSuccess = options?.mutation?.onSuccess as never
      // SAFETY: partial mock; only mutation.onSuccess is exercised here.
      return {} as any
    })

    const queryClient = new QueryClient()
    const invalidateSpy = vi.spyOn(queryClient, "invalidateQueries")
    const removeSpy = vi.spyOn(queryClient, "removeQueries")

    function Wrapper({ children }: { children: ReactNode }) {
      return (
        <QueryClientProvider client={queryClient}>
          {children}
        </QueryClientProvider>
      )
    }

    renderHook(() => useDeleteCollectionMutation(), { wrapper: Wrapper })

    capturedOnSuccess?.(
      {
        status: 404,
        data: { detail: "Not found" },
        headers: new Headers(),
      },
      { id: "1" }
    )

    expect(invalidateSpy).not.toHaveBeenCalled()
    expect(removeSpy).not.toHaveBeenCalled()
  })
})

describe("invalidateCollectionCounts", () => {
  it("refetches the list and the single Collection, including inactive queries", () => {
    const invalidateQueries = vi.fn()
    // SAFETY: only invalidateQueries is used.
    invalidateCollectionCounts({ invalidateQueries } as any, "col-1")
    expect(invalidateQueries.mock.calls.map(([arg]) => arg)).toEqual([
      { queryKey: getListCollectionsQueryKey(), refetchType: "all" },
      { queryKey: getGetCollectionQueryKey("col-1"), refetchType: "all" },
    ])
  })
})

// Real keys in a real cache: a prefix that doesn't match fails here, which a
// mocked invalidateQueries would never notice.
describe("invalidateCollectionEntries", () => {
  it("reaches the infinite list and the plain quantity lookups of that Collection only", () => {
    const client = new QueryClient()
    const hit = [
      getListCollectionEntriesInfiniteQueryKey("col-1", { limit: 100 }),
      getListCollectionEntriesQueryKey("col-1", { cardId: ["a"], limit: 1 }),
    ]
    const other = getListCollectionEntriesInfiniteQueryKey("col-2", {
      limit: 100,
    })
    for (const key of [...hit, other]) client.setQueryData(key, {})

    invalidateCollectionEntries(client, "col-1")

    for (const key of hit)
      expect(client.getQueryState(key)?.isInvalidated).toBe(true)
    expect(client.getQueryState(other)?.isInvalidated).toBe(false)
  })
})

type EntriesPage = Awaited<ReturnType<typeof listCollectionEntries>>
type Cached = InfiniteData<EntriesPage, number | undefined>

describe("patchCollectionEntryQuantities", () => {
  const item = (id: string, quantity: number) =>
    // SAFETY: partial card; the patch reads only its id.
    ({ card: { id }, quantity }) as InventoryItem
  const page = (rows: InventoryItem[], n: number): EntriesPage => ({
    status: 200,
    data: { data: rows, meta: { page: n, limit: 2, total: 6 } },
    headers: new Headers(),
  })
  const rowsOf = (cached: Cached, index: number) => {
    const found = cached.pages[index]
    return found?.status === 200
      ? (found.data.data ?? []).map((r) => [r.card.id, r.quantity])
      : []
  }
  const key = getListCollectionEntriesInfiniteQueryKey("col-1", { limit: 100 })

  function seed() {
    const client = new QueryClient()
    const cached: Cached = {
      pages: [
        page([item("a", 3), item("b", 3)], 1),
        page([item("c", 3), item("d", 3)], 2),
        page([item("e", 3), item("f", 3)], 3),
      ],
      pageParams: [1, 2, 3],
    }
    client.setQueryData(key, cached)
    return { client, cached }
  }
  const saved = (cardId: string, quantity: number) => ({
    cardId,
    quantity,
    status: "applied" as const,
  })
  const read = (client: QueryClient) => client.getQueryData<Cached>(key)!

  it("writes the saved quantity into the page holding the card, keeping the cache shape and every other page's identity", async () => {
    const { client, cached } = seed()

    await patchCollectionEntryQuantities(client, "col-1", [saved("c", 5)])

    const next = read(client)
    expect(next.pageParams).toEqual([1, 2, 3])
    expect(next.pages).toHaveLength(3)
    expect(next.pages[0]).toBe(cached.pages[0])
    expect(next.pages[2]).toBe(cached.pages[2])
    expect(next.pages[1]).not.toBe(cached.pages[1])
    expect(rowsOf(next, 1)).toEqual([
      ["c", 5],
      ["d", 3],
    ])
  })

  it("keeps a card saved at 0 in the cache at quantity 0", async () => {
    const { client } = seed()

    await patchCollectionEntryQuantities(client, "col-1", [
      { cardId: "e", quantity: 0, status: "removed" },
    ])

    expect(rowsOf(read(client), 2)).toEqual([
      ["e", 0],
      ["f", 3],
    ])
  })

  it("leaves the cache entry untouched when nothing changes or the card is not loaded", async () => {
    const { client, cached } = seed()

    await patchCollectionEntryQuantities(client, "col-1", [
      saved("a", 3),
      saved("zzz", 9),
    ])
    await patchCollectionEntryQuantities(client, "col-1", [])

    expect(read(client)).toBe(cached)
  })

  it("does nothing when the list is not cached", async () => {
    const client = new QueryClient()
    await patchCollectionEntryQuantities(client, "col-1", [saved("a", 5)])
    expect(client.getQueryCache().getAll()).toHaveLength(0)
  })

  it("cancels a fetch in flight so it cannot write stale pages back over the patch", async () => {
    const { client } = seed()
    const signalled = vi.fn()
    // A page append started before the save: it will resolve with the pages it saw.
    const fetching = client
      .fetchQuery({
        queryKey: key,
        queryFn: ({ signal }) => {
          signal.addEventListener("abort", signalled)
          return new Promise(() => {})
        },
      })
      .catch(() => undefined)

    await patchCollectionEntryQuantities(client, "col-1", [saved("c", 5)])
    await fetching

    expect(signalled).toHaveBeenCalled()
    expect(client.getQueryState(key)?.fetchStatus).toBe("idle")
    expect(rowsOf(read(client), 1)[0]).toEqual(["c", 5])
  })
})
