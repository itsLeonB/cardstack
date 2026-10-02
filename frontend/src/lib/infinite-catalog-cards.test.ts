import { renderHook } from "@testing-library/react"
import { describe, expect, it, vi } from "vitest"
import {
  mergeCatalogPages,
  useInfiniteCardResultsProps,
} from "./infinite-catalog-cards"
import type { searchCatalogCards } from "@/generated/endpoints/catalog/catalog"
import type { CardSummary } from "@/generated/models"

type Page = Awaited<ReturnType<typeof searchCatalogCards>>

// SAFETY: partial fixtures; the code under test reads only `id`, `meta` and `status`.
const card = (id: string) => ({ id }) as CardSummary

describe("mergeCatalogPages", () => {
  it("returns the deduped cards and the latest total", () => {
    const page = (ids: string[], total: number) =>
      // SAFETY: a 200 page; headers are unused by the code under test.
      ({
        status: 200,
        data: { data: ids.map(card), meta: { page: 1, limit: 2, total } },
        headers: new Headers(),
      }) as Page
    const merged = mergeCatalogPages([page(["a", "b"], 4), page(["b", "c"], 5)])
    expect(merged.cards.map((c) => c.id)).toEqual(["a", "b", "c"])
    expect(merged.total).toBe(5)
  })
})

describe("useInfiniteCardResultsProps", () => {
  it("loads the next page without cancelling a background refetch", () => {
    const fetchNextPage = vi.fn()
    const query = {
      data: { cards: [card("a")], total: 5 },
      isPending: false,
      isError: true,
      error: new Error("boom"),
      hasNextPage: true,
      isFetching: false,
      fetchNextPage,
    }

    const { result } = renderHook(() => useInfiniteCardResultsProps(query))
    result.current.onLoadMore()

    expect(fetchNextPage).toHaveBeenCalledWith({ cancelRefetch: false })
    expect(result.current).toMatchObject({
      total: 5,
      errorMessage: "boom",
      hasNextPage: true,
    })
  })

  it("defaults to an empty list before data arrives", () => {
    const query = {
      data: undefined,
      isPending: true,
      isError: false,
      error: null,
      hasNextPage: false,
      isFetching: true,
      fetchNextPage: vi.fn(),
    }
    const { result } = renderHook(() => useInfiniteCardResultsProps(query))
    expect(result.current).toMatchObject({ cards: [], total: 0 })
  })
})
