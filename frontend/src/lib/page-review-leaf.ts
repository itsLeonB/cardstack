/**
 * Keeps pending Page Reviews on the Page they were saved for when a Leaf is
 * added or removed. Each Leaf is two Pages, so every Page from the changed
 * Leaf onward moves by two. Only the signed-in user's reviews of this binder
 * are touched.
 */
import {
  listPageReviews,
  removePageReview,
  renamePageReview,
} from "@/lib/page-review-store"

/** `leaf` is the Leaf number the new Leaf takes, or the removed Leaf's number. */
export type LeafChange =
  { kind: "add"; leaf: number } | { kind: "remove"; leaf: number }

export interface PageReviewShift {
  /** Pages whose reviews go: the removed Leaf's own two Pages. */
  discard: number[]
  /** Renames in the order to run them, so each target is free when written. */
  renames: { from: number; to: number }[]
}

export interface BinderScope {
  userId: string
  collectionId: string
}

export function planPageReviewShift(
  pages: number[],
  change: LeafChange
): PageReviewShift {
  const front = 2 * change.leaf - 1
  if (change.kind === "add") {
    // Pages from the new Leaf's front on move up, the highest first so each
    // target is free.
    const moved = pages.filter((page) => page >= front).sort((a, b) => b - a)
    return {
      discard: [],
      renames: moved.map((page) => ({ from: page, to: page + 2 })),
    }
  }
  // The removed Leaf's Pages go first, so the Pages after it can move down,
  // the lowest first.
  const moved = pages.filter((page) => page > front + 1).sort((a, b) => a - b)
  return {
    discard: pages.filter((page) => page === front || page === front + 1),
    renames: moved.map((page) => ({ from: page, to: page - 2 })),
  }
}

/** Applies a Leaf change to the store. Resolves with the Pages whose reviews were discarded. */
export async function applyLeafChange(
  scope: BinderScope,
  change: LeafChange
): Promise<number[]> {
  const reviews = await listPageReviews(scope.userId, scope.collectionId)
  const plan = planPageReviewShift(
    reviews.map((review) => review.page),
    change
  )
  for (const page of plan.discard) {
    await removePageReview({ ...scope, page })
  }
  for (const { from, to } of plan.renames) {
    await renamePageReview({ ...scope, page: from }, to)
  }
  return plan.discard
}

/**
 * Discards the reviews of this binder whose cell count no longer matches the
 * Layout's rows and columns. Resolves with the Pages discarded.
 */
export async function discardMismatchedReviews(
  scope: BinderScope,
  layout: { rows: number; columns: number }
): Promise<number[]> {
  const reviews = await listPageReviews(scope.userId, scope.collectionId)
  const stale = reviews.filter(
    (review) => review.cells.length !== layout.rows * layout.columns
  )
  for (const review of stale) {
    await removePageReview({ ...scope, page: review.page })
  }
  return stale.map((review) => review.page)
}
