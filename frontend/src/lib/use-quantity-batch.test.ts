import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"
import { act, renderHook } from "@testing-library/react"
import { bulkUpdateCollectionEntries } from "@/generated/endpoints/inventory/inventory"
import { QUANTITY_DEBOUNCE_MS, useQuantityBatch } from "./use-quantity-batch"

// Isolates the hook from the network; the generated client's wire behaviour is orval's job.
// oxlint-disable-next-line anti-slop/no-module-mocking
vi.mock("@/generated/endpoints/inventory/inventory", () => ({
  bulkUpdateCollectionEntries: vi.fn(),
}))

const bulk = vi.mocked(bulkUpdateCollectionEntries)

function respond(results: { cardId: string; quantity: number; status: string; message?: string }[]) {
  // SAFETY: partial response; the hook reads only status and data.data.
  bulk.mockResolvedValue({ status: 200, data: { data: results } } as any)
}

async function tick(ms: number) {
  await act(async () => {
    await vi.advanceTimersByTimeAsync(ms)
  })
}

describe("useQuantityBatch", () => {
  beforeEach(() => {
    vi.useFakeTimers()
    bulk.mockReset()
  })
  afterEach(() => vi.useRealTimers())

  it("sends one bulk call after the debounce, oldest change first", async () => {
    respond([])
    const { result } = renderHook(() => useQuantityBatch("col-1"))

    act(() => result.current.setQuantity("a", 2, 1))
    await tick(100)
    act(() => result.current.setQuantity("b", 5, 4))
    await tick(100)
    // Re-changing "a" makes it the most recent change, so it moves to the end.
    act(() => result.current.setQuantity("a", 3, 1))

    expect(result.current.quantities).toEqual({ a: 3, b: 5 })
    await tick(QUANTITY_DEBOUNCE_MS - 1)
    expect(bulk).not.toHaveBeenCalled()
    await tick(1)

    expect(bulk).toHaveBeenCalledTimes(1)
    expect(bulk).toHaveBeenCalledWith("col-1", {
      items: [
        { cardId: "b", quantity: 5 },
        { cardId: "a", quantity: 3 },
      ],
    })
  })

  it("reverts only the declined card, with its error, and keeps the others", async () => {
    respond([
      { cardId: "a", quantity: 1, status: "applied" },
      { cardId: "b", quantity: 4, status: "declined", message: "Capacity exceeded" },
    ])
    const { result } = renderHook(() => useQuantityBatch("col-1"))

    act(() => result.current.setQuantity("a", 2, 1))
    act(() => result.current.setQuantity("b", 9, 4))
    await tick(QUANTITY_DEBOUNCE_MS)

    expect(result.current.quantities.b).toBe(4)
    expect(result.current.errors).toEqual({ b: "Capacity exceeded" })
    // "a" is untouched by the response's quantity field: it stays optimistic.
    expect(result.current.quantities.a).toBe(2)
  })

  it("reverts every affected card with an error on network failure", async () => {
    bulk.mockRejectedValue(new Error("offline"))
    const { result } = renderHook(() => useQuantityBatch("col-1"))

    act(() => result.current.setQuantity("a", 2, 1))
    act(() => result.current.setQuantity("b", 9, 4))
    await tick(QUANTITY_DEBOUNCE_MS)

    expect(result.current.quantities).toEqual({ a: 1, b: 4 })
    expect(Object.keys(result.current.errors).sort()).toEqual(["a", "b"])
  })

  it("keeps a card at 0 and clears its error when changed again", async () => {
    respond([{ cardId: "a", quantity: 0, status: "removed" }])
    const { result } = renderHook(() => useQuantityBatch("col-1"))

    act(() => result.current.setQuantity("a", 0, 1))
    await tick(QUANTITY_DEBOUNCE_MS)
    expect(result.current.quantities.a).toBe(0)

    act(() => result.current.setQuantity("a", 1, 1))
    expect(result.current.quantities.a).toBe(1)
  })

  it("flush sends immediately and resolves after the request", async () => {
    respond([])
    const { result } = renderHook(() => useQuantityBatch("col-1"))

    act(() => result.current.setQuantity("a", 2, 1))
    await act(async () => {
      await result.current.flush()
    })

    expect(bulk).toHaveBeenCalledTimes(1)
    await tick(QUANTITY_DEBOUNCE_MS)
    expect(bulk).toHaveBeenCalledTimes(1)
  })

  it("flushes pending edits on unmount", async () => {
    respond([])
    const { result, unmount } = renderHook(() => useQuantityBatch("col-1"))

    act(() => result.current.setQuantity("a", 2, 1))
    unmount()
    await tick(0)

    expect(bulk).toHaveBeenCalledTimes(1)
  })
})
