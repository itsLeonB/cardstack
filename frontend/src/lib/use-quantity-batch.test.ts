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

function respond(
  results: {
    cardId: string
    quantity: number
    status: string
    message?: string
  }[]
) {
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
      {
        cardId: "b",
        quantity: 4,
        status: "declined",
        message: "Capacity exceeded",
      },
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

  it("sends an edit made during an in-flight request in the next batch, oldest first", async () => {
    let release: () => void = () => {}
    // SAFETY: partial response; the hook reads only status and data.data.
    bulk.mockImplementationOnce(
      () =>
        new Promise(
          (resolve) =>
            (release = () =>
              resolve({ status: 200, data: { data: [] } } as any))
        )
    )
    respond([])
    const { result } = renderHook(() => useQuantityBatch("col-1"))

    act(() => result.current.setQuantity("a", 2, 1))
    await tick(QUANTITY_DEBOUNCE_MS)
    expect(bulk).toHaveBeenCalledTimes(1)

    act(() => result.current.setQuantity("b", 5, 4))
    act(() => result.current.setQuantity("c", 7, 6))
    act(() => result.current.setQuantity("b", 8, 4))
    expect(result.current.hasPending()).toBe(true)
    await tick(QUANTITY_DEBOUNCE_MS)
    // Serialized behind the first request.
    expect(bulk).toHaveBeenCalledTimes(1)

    await act(async () => release())
    await tick(0)
    expect(bulk).toHaveBeenCalledTimes(2)
    expect(bulk).toHaveBeenLastCalledWith("col-1", {
      items: [
        { cardId: "c", quantity: 7 },
        { cardId: "b", quantity: 8 },
      ],
    })
  })

  it("does not flag an outcome that a newer pending edit supersedes", async () => {
    bulk.mockRejectedValueOnce(new Error("offline"))
    const { result } = renderHook(() => useQuantityBatch("col-1"))

    act(() => result.current.setQuantity("a", 2, 1))
    await act(async () => {
      void result.current.flush()
      result.current.setQuantity("a", 3, 1)
    })
    await tick(0)

    expect(result.current.quantities.a).toBe(3)
    expect(result.current.errors).toEqual({})
  })

  it("passes the server's per-card results to onSaved", async () => {
    const results = [
      { cardId: "a", quantity: 2, status: "applied" },
      { cardId: "b", quantity: 4, status: "declined", message: "No room" },
    ]
    respond(results)
    const onSaved = vi.fn()
    const { result } = renderHook(() => useQuantityBatch("col-1", onSaved))

    act(() => result.current.setQuantity("a", 2, 1))
    act(() => result.current.setQuantity("b", 9, 4))
    await tick(QUANTITY_DEBOUNCE_MS)

    expect(onSaved).toHaveBeenCalledTimes(1)
    expect(onSaved).toHaveBeenCalledWith(results)
  })

  it("calls onSaved with no results when the response is lost, since the write may have committed", async () => {
    bulk.mockRejectedValue(new Error("offline"))
    const onSaved = vi.fn()
    const { result } = renderHook(() => useQuantityBatch("col-1", onSaved))

    act(() => result.current.setQuantity("a", 2, 1))
    await tick(QUANTITY_DEBOUNCE_MS)

    expect(onSaved).toHaveBeenCalledWith([])
  })

  it("does not call onSaved for a rejected request", async () => {
    // SAFETY: partial response; the hook reads only status and data.detail.
    bulk.mockResolvedValue({ status: 400, data: { detail: "bad" } } as any)
    const onSaved = vi.fn()
    const { result } = renderHook(() => useQuantityBatch("col-1", onSaved))

    act(() => result.current.setQuantity("a", 2, 1))
    await tick(QUANTITY_DEBOUNCE_MS)

    expect(onSaved).not.toHaveBeenCalled()
  })

  describe("prune", () => {
    async function saved(
      results: Parameters<typeof respond>[0],
      edits: [string, number, number][]
    ) {
      respond(results)
      const view = renderHook(() => useQuantityBatch("col-1"))
      for (const [cardId, quantity, server] of edits)
        act(() => view.result.current.setQuantity(cardId, quantity, server))
      await tick(QUANTITY_DEBOUNCE_MS)
      return view.result
    }

    it("drops an override once the server data differs from what was confirmed", async () => {
      const result = await saved(
        [{ cardId: "a", quantity: 5, status: "applied" }],
        [["a", 5, 1]]
      )
      expect(result.current.quantities.a).toBe(5)

      // A refetch brought a quantity changed elsewhere.
      act(() => result.current.prune(() => 7))
      expect(result.current.quantities).toEqual({})
    })

    it("drops an override whose row is gone, e.g. a card saved at 0 after a refetch", async () => {
      const result = await saved(
        [{ cardId: "a", quantity: 0, status: "removed" }],
        [["a", 0, 1]]
      )
      expect(result.current.quantities.a).toBe(0)

      act(() => result.current.prune(() => undefined))
      expect(result.current.quantities).toEqual({})
    })

    it("keeps overrides and errors the data still agrees with, as when a page appends", async () => {
      const result = await saved(
        [
          { cardId: "a", quantity: 0, status: "removed" },
          { cardId: "b", quantity: 4, status: "declined", message: "No room" },
        ],
        [
          ["a", 0, 1],
          ["b", 9, 4],
        ]
      )

      // The cache holds a patched 0 for "a" and the unchanged 4 for "b".
      act(() => result.current.prune((cardId) => ({ a: 0, b: 4 })[cardId]))

      expect(result.current.quantities).toEqual({ a: 0, b: 4 })
      expect(result.current.errors).toEqual({ b: "No room" })
    })

    it("never touches a card with an edit outstanding", () => {
      respond([])
      const { result } = renderHook(() => useQuantityBatch("col-1"))

      act(() => result.current.setQuantity("a", 5, 1))
      act(() => result.current.prune(() => 9))

      expect(result.current.quantities.a).toBe(5)
    })
  })
})
