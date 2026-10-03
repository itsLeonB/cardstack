import { renderHook } from "@testing-library/react"
import { describe, expect, it, vi } from "vitest"
import { useInfiniteCardResultsProps } from "./infinite-catalog-cards"
import type { CardSummary } from "@/generated/models"

// SAFETY: partial fixtures; the code under test reads only `id`, `meta` and `status`.
const card = (id: string) => ({ id }) as CardSummary

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
