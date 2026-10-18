import { useState } from "react"
import { useQueryClient } from "@tanstack/react-query"
import {
  AlertDialog,
  AlertDialogAction,
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
import { invalidatePendingReviews } from "@/lib/use-pending-reviews"

/**
 * "Scan this Page". A Page with a pending review asks whether to resume it or
 * start over. Start over discards the review only once it is chosen.
 */
export function ScanThisPageButton({
  collectionId,
  page,
  onScan,
}: {
  collectionId: string
  page: number
  onScan: (resume?: PendingPageReview) => void
}) {
  const { user } = useSession()
  const userId = user?.id
  const queryClient = useQueryClient()
  const [pending, setPending] = useState<PendingPageReview>()
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string | null>(null)

  async function start() {
    setError(null)
    // Reads the store itself, not the list, so a list still loading cannot skip the prompt.
    const found =
      userId === undefined
        ? undefined
        : await loadPageReview({ userId, collectionId, page }).catch(
            () => undefined
          )
    if (found) setPending(found)
    else onScan(undefined)
  }

  function startOver() {
    if (pending === undefined) return
    setBusy(true)
    removePageReview(pending).then(
      () => {
        setBusy(false)
        setPending(undefined)
        void invalidatePendingReviews(queryClient)
        onScan(undefined)
      },
      () => {
        setBusy(false)
        setError("Could not discard this review. Try again.")
      }
    )
  }

  return (
    <>
      <Button variant="outline" onClick={() => void start()}>
        Scan this Page
      </Button>

      <AlertDialog
        open={pending !== undefined}
        onOpenChange={(open) => {
          if (!open) setPending(undefined)
        }}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Resume or start over?</AlertDialogTitle>
            <AlertDialogDescription>
              Starting over deletes this Page&apos;s unfinished scan.
            </AlertDialogDescription>
          </AlertDialogHeader>
          {error && (
            <p role="alert" className="text-sm text-destructive">
              {error}
            </p>
          )}
          <AlertDialogFooter>
            <Button variant="outline" disabled={busy} onClick={startOver}>
              Start over
            </Button>
            <AlertDialogAction onClick={() => onScan(pending)}>
              Resume
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </>
  )
}
