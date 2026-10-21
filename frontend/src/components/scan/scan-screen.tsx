import { useRef, useState } from "react"
import type { ReactNode } from "react"
import { Button } from "@/components/ui/button"
import { cameraMessage, ChoosePhotoButton } from "./photo-source-controls"
import { ReviewStep } from "./review-step"
import type { CommitResult, Decline } from "./review-step"
import { ScanSheet } from "./scan-sheet"
import { CARD_ASPECT, GUIDE_HEIGHT, VIEW_ASPECT } from "@/lib/frame-crop"
import type { FrameSource } from "@/lib/frame-source"
import { summedRows, useScanDraft } from "@/lib/use-draft-addition"

/**
 * The scan screen: a full-screen viewfinder with the Draft Addition as a
 * bottom sheet over it, and the Review step in place of both. Captures never
 * wait for a match, so each one lands in the sheet at once.
 */
export function ScanScreen({
  userId,
  collectionId,
  source,
  backLink,
  onAdded,
}: {
  /** Owns the saved Draft Addition, so another user never sees it. */
  userId: string
  collectionId: string
  source: FrameSource
  /** A visible link back to the Collection, kept over the viewfinder. */
  backLink?: ReactNode
  /** Every Scan was applied: the user goes back to the Collection. */
  onAdded: () => void
}) {
  // Each message is a new node, so a repeat of the same text is read again.
  const [announcement, setAnnouncement] = useState<{
    id: number
    text: string
  }>()
  const announcementId = useRef(0)
  const draft = useScanDraft(userId, collectionId, (text) =>
    setAnnouncement({ id: ++announcementId.current, text })
  )
  const [error, setError] = useState<string | null>(null)
  const [reviewing, setReviewing] = useState(false)
  const [sheetOpen, setSheetOpen] = useState(false)
  // Why the Collection declined a Card in the last commit, by card id.
  const [declined, setDeclined] = useState<Record<string, Decline>>({})

  // A reason describes its Card's Scans as they were, so an edit to them makes it stale.
  function forget(...cardIds: (string | undefined)[]) {
    setDeclined((reasons) =>
      Object.fromEntries(
        Object.entries(reasons).filter(([id]) => !cardIds.includes(id))
      )
    )
  }
  const cardOf = (key: string) => draft.scans.find((s) => s.key === key)?.cardId

  async function capture(takeFrame: () => Promise<Blob>) {
    setError(null)
    try {
      draft.capture(await takeFrame())
    } catch {
      setError("Could not capture that photo. Your draft is unchanged.")
    }
  }

  function finishCommit({ placed, declined: next, failed }: CommitResult) {
    const left = draft.removeCopies(placed)
    setDeclined(next)
    if (left === 0) onAdded()
    else if (!failed && Object.keys(next).length === 0) setReviewing(false)
  }

  const reviewable =
    draft.scans.some((s) => s.status === "resolved") && draft.pendingCount === 0

  const status = (
    <p role="status" className="sr-only">
      {announcement && <span key={announcement.id}>{announcement.text}</span>}
    </p>
  )

  // A fixed layer under the site header (3.5rem plus its 1px border), so the
  // page's footer never shows behind the scan screen.
  if (reviewing) {
    return (
      <div className="fixed inset-x-0 top-[calc(3.5rem+1px)] bottom-0 z-30 overflow-y-auto bg-background">
        <div className="mx-auto flex w-full max-w-md flex-col gap-4 px-4 py-6">
          {backLink}
          {status}
          <ReviewStep
            collectionId={collectionId}
            rows={summedRows(draft.scans)}
            seen={draft.seen}
            declined={declined}
            onCommitted={finishCommit}
            onBack={() => setReviewing(false)}
          />
        </div>
      </div>
    )
  }

  const hasCamera = source.camera === "starting" || source.camera === "ready"

  return (
    <div className="fixed inset-x-0 top-[calc(3.5rem+1px)] bottom-0 z-30 bg-background">
      <div className="mx-auto flex h-full w-full max-w-md flex-col">
        {status}
        <div className="relative min-h-0 flex-1 overflow-hidden bg-black text-white">
          {hasCamera ? (
            // The box and the guide are sized from the constants `lib/frame-crop.ts` crops to.
            <div
              className="relative w-full overflow-hidden"
              style={{ aspectRatio: VIEW_ASPECT }}
            >
              <video
                ref={source.videoRef}
                playsInline
                muted
                aria-label="Camera viewfinder"
                className="size-full object-cover"
              />
              <div
                aria-hidden
                className="absolute top-1/2 left-1/2 -translate-x-1/2 -translate-y-1/2 rounded-xl border-2 border-white/90 shadow-[0_0_0_9999px_rgb(0_0_0/0.45)]"
                style={{
                  height: `${GUIDE_HEIGHT * 100}%`,
                  aspectRatio: CARD_ASPECT,
                }}
              />
            </div>
          ) : (
            <p role="alert" className="px-4 pt-20 text-center text-sm">
              {cameraMessage(source.camera)}
            </p>
          )}
          {backLink && <div className="absolute top-3 left-3">{backLink}</div>}
          {error && (
            <p
              role="alert"
              className="absolute inset-x-3 top-14 rounded-lg bg-background p-2 text-sm text-destructive"
            >
              {error}
            </p>
          )}
          <div className="absolute inset-x-0 bottom-3 flex flex-wrap justify-center gap-2 px-4">
            {hasCamera && (
              <Button
                size="lg"
                disabled={source.camera !== "ready"}
                onClick={() => void capture(source.captureCamera)}
              >
                Capture
              </Button>
            )}
            <ChoosePhotoButton
              onFile={(file) => void capture(() => source.fromFile(file))}
            />
          </div>
        </div>
        <ScanSheet
          scans={draft.scans}
          seen={draft.seen}
          pendingCount={draft.pendingCount}
          open={sheetOpen}
          onOpenChange={setSheetOpen}
          reviewable={reviewable}
          onRaise={(key) => {
            forget(cardOf(key))
            draft.raise(key)
          }}
          onLower={(key) => {
            forget(cardOf(key))
            draft.lower(key)
          }}
          onChoose={(key, cardId, reprintPosition) => {
            forget(cardOf(key), cardId)
            draft.choose(key, cardId, reprintPosition)
          }}
          onRetry={draft.retry}
          onRemove={(key) => {
            forget(cardOf(key))
            draft.remove(key)
          }}
          onDiscard={() => {
            draft.discard()
            setDeclined({})
            setSheetOpen(false)
          }}
          onReview={() => setReviewing(true)}
        />
      </div>
    </div>
  )
}
