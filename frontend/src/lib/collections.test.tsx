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
  invalidateCollectionCounts,
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
