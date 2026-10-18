import { beforeEach, describe, expect, it, vi } from "vitest"
import { IDBFactory, IDBKeyRange } from "fake-indexeddb"
import { Blob as NodeBlob } from "node:buffer"
import {
  applyLeafChange,
  discardMismatchedReviews,
  planPageReviewShift,
} from "./page-review-leaf"
import type { PageReviewShift } from "./page-review-leaf"
import { listPageReviews, savePageReview } from "./page-review-store"
import type { PendingPageReview } from "./page-review-store"

// SAFETY: Node's Blob reads the same as the DOM Blob that a cell's photo is typed as.
const photo = () => new NodeBlob(["cell"]) as Blob

// A review with `cells` cells, on a binder of one row and `cells` columns.
const review = (
  page: number,
  { userId = "user-1", collectionId = "col-1", cells = 2 } = {}
): PendingPageReview => ({
  userId,
  collectionId,
  page,
  grid: { rows: 1, columns: cells },
  cells: Array.from({ length: cells }, () => ({
    photo: photo(),
    match: { status: "failed" as const },
    mark: "empty" as const,
  })),
})

// The Pages left after a plan, the way the applier runs it: discards first,
// then renames in order, where each target must be free when it is written.
function applyPlan(pages: number[], plan: PageReviewShift): number[] {
  const taken = new Set(pages.filter((page) => !plan.discard.includes(page)))
  for (const { from, to } of plan.renames) {
    expect(taken.has(from)).toBe(true)
    expect(taken.has(to)).toBe(false)
    taken.delete(from)
    taken.add(to)
  }
  return [...taken].sort((a, b) => a - b)
}

describe("planPageReviewShift", () => {
  it("shifts every Page from the new Leaf's front on by two when a Leaf is added", () => {
    // A new Leaf 2 pushes Leaf 2 (Pages 3 and 4) and everything after it back.
    const plan = planPageReviewShift([1, 2, 3, 4, 5, 6], {
      kind: "add",
      leaf: 2,
    })
    expect(plan.discard).toEqual([])
    expect(applyPlan([1, 2, 3, 4, 5, 6], plan)).toEqual([1, 2, 5, 6, 7, 8])
  })

  it("leaves the Pages before an added Leaf where they are", () => {
    // A Leaf added after Leaf 2 starts at Page 5.
    const plan = planPageReviewShift([3, 4], { kind: "add", leaf: 3 })
    expect(applyPlan([3, 4], plan)).toEqual([3, 4])
  })

  it("discards the review of a removed Leaf's own Pages and shifts later ones down", () => {
    const pages = [1, 2, 3, 4, 5, 6]
    const plan = planPageReviewShift(pages, { kind: "remove", leaf: 2 })
    expect(plan.discard).toEqual([3, 4])
    expect(applyPlan(pages, plan)).toEqual([1, 2, 3, 4])
  })

  it("discards both Pages of the first Leaf when it is removed with no Leaf before it", () => {
    const plan = planPageReviewShift([1, 2, 3], { kind: "remove", leaf: 1 })
    expect(applyPlan([1, 2, 3], plan)).toEqual([1])
  })
})

beforeEach(() => {
  // A fresh in-memory database for every test.
  vi.stubGlobal("indexedDB", new IDBFactory())
  vi.stubGlobal("IDBKeyRange", IDBKeyRange)
})

describe("applyLeafChange", () => {
  const scope = { userId: "user-1", collectionId: "col-1" }

  it("moves this user's and this binder's reviews and returns the discarded Pages", async () => {
    await savePageReview(review(3))
    await savePageReview(review(4))
    await savePageReview(review(3, { userId: "user-2" }))
    await savePageReview(review(3, { collectionId: "col-2" }))

    const discarded = await applyLeafChange(scope, { kind: "remove", leaf: 2 })

    expect(discarded).toEqual([3, 4])
    expect(
      (await listPageReviews("user-1", "col-1")).map((r) => r.page)
    ).toEqual([])
    expect(
      (await listPageReviews("user-2", "col-1")).map((r) => r.page)
    ).toEqual([3])
    expect(
      (await listPageReviews("user-1", "col-2")).map((r) => r.page)
    ).toEqual([3])
  })

  it("shifts a review after an added Leaf to its new Page", async () => {
    await savePageReview(review(5))

    const discarded = await applyLeafChange(scope, { kind: "add", leaf: 2 })

    expect(discarded).toEqual([])
    const [moved] = await listPageReviews("user-1", "col-1")
    expect(moved.page).toBe(7)
    expect(moved.cells).toHaveLength(2)
  })
})

describe("discardMismatchedReviews", () => {
  const scope = { userId: "user-1", collectionId: "col-1" }

  it("discards this binder's reviews whose cell count is not rows x columns", async () => {
    await savePageReview(review(1, { cells: 2 }))
    await savePageReview(review(2, { cells: 3 }))
    await savePageReview(review(3, { cells: 2, userId: "user-2" }))

    // The Layout is now 1 row by 3 columns.
    const discarded = await discardMismatchedReviews(scope, {
      rows: 1,
      columns: 3,
    })

    expect(discarded).toEqual([1])
    expect(
      (await listPageReviews("user-1", "col-1")).map((r) => r.page)
    ).toEqual([2])
    expect(
      (await listPageReviews("user-2", "col-1")).map((r) => r.page)
    ).toEqual([3])
  })
})
