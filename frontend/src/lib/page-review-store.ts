/**
 * Pending Page Reviews in the browser's IndexedDB, which holds their photos.
 * A review is keyed by the signed-in user's id, the binder's Collection and the
 * Page number, so another login on the same browser never sees it. A save
 * overwrites the review for its key, so two tabs on one review are last write
 * wins. Nothing expires; a review ends with a successful Confirm or a Discard.
 *
 * - savePageReview(review): create or overwrite the review for its key.
 * - loadPageReview(key): the review, or undefined.
 * - listPageReviews(userId, collectionId): one binder's reviews, in Page order.
 * - removePageReview(key): delete it; removing a missing one is not an error.
 * - renamePageReview(key, page): move it to another Page number, for a Leaf
 *   added or removed. The target Page must be free, so shift the highest first.
 *
 * Every call rejects when IndexedDB cannot be used (no IndexedDB, private mode,
 * quota). Callers then fall back to a memory-only review.
 */
import type { PageCell } from "@/lib/use-page-scan"
import type { Grid } from "@/lib/page-cut"
import { transactStore } from "@/lib/indexed-db"

const DB_NAME = "cardstack-page-reviews"
const STORE = "reviews"
// The fields of a record are its key, so a review is found by its three values.
const KEY_PATH = ["userId", "collectionId", "page"]

export interface PageReviewKey {
  userId: string
  collectionId: string
  page: number
}

export interface PendingPageReview extends PageReviewKey {
  grid: Grid
  cells: PageCell[]
}

function transact<T>(
  mode: IDBTransactionMode,
  work: (store: IDBObjectStore) => IDBRequest<T>
): Promise<T> {
  return transactStore(DB_NAME, STORE, KEY_PATH, mode, work)
}

const keyOf = ({ userId, collectionId, page }: PageReviewKey) => [
  userId,
  collectionId,
  page,
]

export async function savePageReview(review: PendingPageReview): Promise<void> {
  await transact("readwrite", (store) => store.put(review))
}

export function loadPageReview(
  key: PageReviewKey
): Promise<PendingPageReview | undefined> {
  return transact("readonly", (store) => store.get(keyOf(key)))
}

export function listPageReviews(
  userId: string,
  collectionId: string
): Promise<PendingPageReview[]> {
  const range = IDBKeyRange.bound(
    [userId, collectionId],
    [userId, collectionId, Infinity]
  )
  return transact("readonly", (store) => store.getAll(range))
}

export async function removePageReview(key: PageReviewKey): Promise<void> {
  await transact("readwrite", (store) => store.delete(keyOf(key)))
}

export async function renamePageReview(
  key: PageReviewKey,
  page: number
): Promise<void> {
  if (key.page === page) return
  const review = await loadPageReview(key)
  if (!review) return
  // Save under the new key before removing the old one: a failure in between
  // leaves a duplicate, never a lost review.
  await savePageReview({ ...review, page })
  await removePageReview(key)
}
