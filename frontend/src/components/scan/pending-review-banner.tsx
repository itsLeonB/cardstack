import { useEffect, useState } from "react"
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog"
import { Button } from "@/components/ui/button"
import { loadPageReview, removePageReview } from "@/lib/page-review-store"
import type { PendingPageReview } from "@/lib/page-review-store"
import { useSession } from "@/lib/session"

interface PendingReviewBannerProps {
  collectionId: string
  page: number
  onResume: (review: PendingPageReview) => void
}

/**
 * Shows a Page's pending Page Review, if the signed-in user has one, with
 * Resume and Discard. It loads on mount, so the binder view remounts it after
 * the scan screen closes to pick up a review that was saved or cleared there.
 */
export function PendingReviewBanner({
  collectionId,
  page,
  onResume,
}: PendingReviewBannerProps) {
  const { user } = useSession()
  const userId = user?.id
  const [review, setReview] = useState<PendingPageReview>()
  const [discarding, setDiscarding] = useState(false)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    if (userId === undefined) return
    let current = true
    // No IndexedDB means no pending review to offer, so a failed read shows nothing.
    loadPageReview({ userId, collectionId, page }).then(
      (found) => {
        if (current) setReview(found)
      },
      () => undefined
    )
    return () => {
      current = false
    }
  }, [userId, collectionId, page])

  if (userId === undefined || !review) return null

  const discard = () => {
    removePageReview({ userId, collectionId, page }).then(
      () => {
        setReview(undefined)
        setDiscarding(false)
      },
      () => {
        setDiscarding(false)
        setError("Could not discard this review. Try again.")
      }
    )
  }

  return (
    <div className="flex flex-col gap-2 rounded-md border p-3">
      <p className="text-sm">This Page has an unfinished scan.</p>
      {error && (
        <p role="alert" className="text-sm text-destructive">
          {error}
        </p>
      )}
      <div className="flex flex-wrap gap-2">
        <Button size="sm" onClick={() => onResume(review)}>
          Resume
        </Button>
        <Button variant="outline" size="sm" onClick={() => setDiscarding(true)}>
          Discard
        </Button>
      </div>

      <AlertDialog open={discarding} onOpenChange={setDiscarding}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Discard this review?</AlertDialogTitle>
            <AlertDialogDescription>
              Its photos and matches are deleted. The Page keeps its Cards.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Keep it</AlertDialogCancel>
            <AlertDialogAction onClick={discard}>Discard</AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  )
}
