import { useState } from "react"
import type { ReactNode } from "react"
import { Button, buttonVariants } from "@/components/ui/button"
import { ReviewStep } from "./review-step"
import type { CommitResult } from "./review-step"
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
  collectionId,
  source,
  backLink,
  onAdded,
}: {
  collectionId: string
  source: FrameSource
  /** A visible link back to the Collection, kept over the viewfinder. */
  backLink?: ReactNode
  /** Every Scan was applied: the user goes back to the Collection. */
  onAdded: () => void
}) {
  const [announcement, setAnnouncement] = useState("")
  const draft = useScanDraft(collectionId, setAnnouncement)
  const [error, setError] = useState<string | null>(null)
  const [reviewing, setReviewing] = useState(false)
  const [sheetOpen, setSheetOpen] = useState(false)
  // Reasons the Collection gave for declining a Card, from the last commit.
  const [declined, setDeclined] = useState<Record<string, string>>({})

  async function capture(takeFrame: () => Promise<Blob>) {
    setError(null)
    try {
      draft.capture(await takeFrame())
    } catch {
      setError("Could not capture that photo. Your draft is unchanged.")
    }
  }

  function finishCommit({ applied, declined: next, failed }: CommitResult) {
    draft.removeCards(applied)
    setDeclined(next)
    const left = draft.scans.some(
      (s) => !(s.status === "resolved" && applied.includes(s.cardId!))
    )
    if (!left) onAdded()
    else if (!failed && Object.keys(next).length === 0) setReviewing(false)
  }

  const reviewable =
    draft.scans.some((s) => s.status === "resolved") && draft.pendingCount === 0

  const status = (
    <p role="status" className="sr-only">
      {announcement}
    </p>
  )

  if (reviewing) {
    return (
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
    )
  }

  const hasCamera = source.camera === "starting" || source.camera === "ready"

  return (
    // The header is 3.5rem plus its 1px bottom border.
    <div className="mx-auto flex h-[calc(100svh-3.5rem-1px)] w-full max-w-md flex-col">
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
            {source.camera === "denied"
              ? "Camera access was denied. Allow it in your browser's site settings, or choose a photo instead."
              : "No camera is available here. Choose a photo instead."}
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
          <label
            className={`${buttonVariants({ variant: "outline", size: "lg" })} bg-background text-foreground has-[:focus-visible]:ring-3 has-[:focus-visible]:ring-ring/30`}
          >
            Choose a photo
            <input
              type="file"
              accept="image/*"
              className="sr-only"
              onChange={(event) => {
                const file = event.currentTarget.files?.[0]
                event.currentTarget.value = ""
                if (file) void capture(() => source.fromFile(file))
              }}
            />
          </label>
        </div>
      </div>
      <ScanSheet
        scans={draft.scans}
        seen={draft.seen}
        pendingCount={draft.pendingCount}
        open={sheetOpen}
        onOpenChange={setSheetOpen}
        reviewable={reviewable}
        onRaise={draft.raise}
        onLower={draft.lower}
        onChoose={draft.choose}
        onRetry={draft.retry}
        onRemove={draft.remove}
        onDiscard={() => {
          draft.discard()
          setDeclined({})
          setSheetOpen(false)
        }}
        onReview={() => setReviewing(true)}
      />
    </div>
  )
}
