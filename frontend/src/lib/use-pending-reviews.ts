import { useQuery } from "@tanstack/react-query"
import type { QueryClient } from "@tanstack/react-query"
import { listPageReviews } from "@/lib/page-review-store"
import { useSession } from "@/lib/session"

const PENDING_REVIEWS = "pending-page-reviews"

/**
 * One binder's pending Page Reviews for the signed-in user, in Page order.
 * Every write to the store calls `invalidatePendingReviews`, so the list follows
 * a save, a Confirm or a Discard. Sign-out drops it with the rest of the cache.
 */
export function usePendingReviews(collectionId: string) {
  const { user } = useSession()
  const userId = user?.id
  return useQuery({
    queryKey: [PENDING_REVIEWS, userId, collectionId],
    queryFn: () => {
      // No IndexedDB means no pending review to list, so a failed read lists none.
      if (userId === undefined) return []
      return listPageReviews(userId, collectionId).catch(() => [])
    },
  })
}

export function invalidatePendingReviews(queryClient: QueryClient) {
  return queryClient.invalidateQueries({ queryKey: [PENDING_REVIEWS] })
}
