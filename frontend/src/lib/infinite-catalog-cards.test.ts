import { describe, expect, it } from "vitest"
import {
  mergeCatalogPages,
  nextCatalogPageParam,
} from "./infinite-catalog-cards"
import type { searchCatalogCards } from "@/generated/endpoints/catalog/catalog"
import type { CardSummary } from "@/generated/models"

type Page = Awaited<ReturnType<typeof searchCatalogCards>>

// SAFETY: partial fixtures; the code under test reads only `id`, `meta` and `status`.
const card = (id: string) => ({ id }) as CardSummary

function ok(
  ids: string[],
  meta: { page: number; limit: number; total: number }
) {
  // SAFETY: a 200 page; headers are unused by the code under test.
  return {
    status: 200,
    data: { data: ids.map(card), meta },
    headers: new Headers(),
  } as Page
}

// SAFETY: a failed page; the error body is unused by the code under test.
const failed = { status: 500, data: {}, headers: new Headers() } as Page

describe("nextCatalogPageParam", () => {
  it("returns the next page while rows remain", () => {
    expect(
      nextCatalogPageParam(ok(["a", "b"], { page: 1, limit: 2, total: 5 }))
    ).toBe(2)
  })

  it("stops once total is reached", () => {
    expect(
      nextCatalogPageParam(ok(["a", "b"], { page: 3, limit: 2, total: 6 }))
    ).toBeUndefined()
  })

  it("stops on a short page even if total says more", () => {
    expect(
      nextCatalogPageParam(ok(["a"], { page: 1, limit: 2, total: 9 }))
    ).toBeUndefined()
  })

  it("stops on an error response", () => {
    expect(nextCatalogPageParam(failed)).toBeUndefined()
  })
})

describe("mergeCatalogPages", () => {
  it("flattens pages in order and reports the latest total", () => {
    const merged = mergeCatalogPages([
      ok(["a", "b"], { page: 1, limit: 2, total: 4 }),
      ok(["c", "d"], { page: 2, limit: 2, total: 5 }),
    ])
    expect(merged.cards.map((c) => c.id)).toEqual(["a", "b", "c", "d"])
    expect(merged.total).toBe(5)
  })

  it("drops a card repeated across a page boundary", () => {
    const merged = mergeCatalogPages([
      ok(["a", "b"], { page: 1, limit: 2, total: 4 }),
      ok(["b", "c"], { page: 2, limit: 2, total: 4 }),
    ])
    expect(merged.cards.map((c) => c.id)).toEqual(["a", "b", "c"])
  })

  it("ignores non-200 pages and null rows", () => {
    // SAFETY: a 200 page with a null row list, as the API can return.
    const empty = {
      status: 200,
      data: { data: null, meta: { page: 1, limit: 2, total: 0 } },
      headers: new Headers(),
    } as Page
    expect(mergeCatalogPages([failed, empty])).toEqual({ cards: [], total: 0 })
  })
})
