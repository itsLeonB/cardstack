import { Fragment } from "react"
import { usePendingReviews } from "@/lib/use-pending-reviews"

/**
 * The Pages of this binder that have a pending Page Review, shown under the
 * jump form on every Page. Each number jumps to its Page, where the banner
 * offers Resume and Discard.
 */
export function PendingReviewNotice({
  collectionId,
  onJump,
}: {
  collectionId: string
  onJump: (page: number) => void
}) {
  const { data: reviews = [] } = usePendingReviews(collectionId)
  if (reviews.length === 0) return null

  return (
    <div role="group" aria-label="Pending scans" className="text-sm">
      <span>{reviews.length === 1 ? "Page" : "Pages"}</span>{" "}
      {reviews.map((review, index) => (
        <Fragment key={review.page}>
          {index > 0 && ", "}
          <button
            type="button"
            aria-label={`Go to Page ${review.page}`}
            className="font-medium text-foreground underline"
            onClick={() => onJump(review.page)}
          >
            {review.page}
          </button>
        </Fragment>
      ))}{" "}
      {reviews.length === 1 ? "has" : "have"} a pending scan
    </div>
  )
}
