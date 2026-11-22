import { useRef, useState } from "react"
import type { ReactNode } from "react"
import { PhotoUseNotice } from "./photo-use-notice"
import { ReviewStep } from "./review-step"
import type { CommitResult, Decline } from "./review-step"
import { ScanCamera, ScanHeader, ScanLayer } from "./scan-layer"
import { ScanFilterBar, useScanFilter } from "./scan-filter-bar"
import { ScanSheet } from "./scan-sheet"
import type { FrameSource } from "@/lib/frame-source"
import { feedbackChoice, sendScanFeedback } from "@/lib/scan-feedback"
import type { PendingFeedbackRow } from "@/lib/scan-feedback"
import { summedRows, useScanDraft } from "@/lib/use-draft-addition"
import type { Scan } from "@/lib/use-draft-addition"

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
  const scanFilter = useScanFilter()

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
      draft.capture(await takeFrame(), scanFilter.filter)
    } catch {
      setError("Could not capture that photo. Your draft is unchanged.")
    }
  }

  function finishCommit({ placed, declined: next, failed }: CommitResult) {
    const { left, removed } = draft.removeCopies(placed, !failed)
    // Not awaited: the draft clears whether or not the feedback is recorded.
    const { rows, photos } = feedbackRows(collectionId, removed)
    void sendScanFeedback(rows, photos)
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
      <ScanLayer scroll>
        <div className="mx-auto flex w-full max-w-md flex-col gap-4 px-4 py-6">
          <ScanHeader title="Scan cards" back={backLink} />
          {status}
          <PhotoUseNotice />
          <ReviewStep
            collectionId={collectionId}
            rows={summedRows(draft.scans)}
            setAside={draft.scans.filter((s) => s.mark).length}
            seen={draft.seen}
            declined={declined}
            onCommitted={finishCommit}
            onBack={() => setReviewing(false)}
          />
        </div>
      </ScanLayer>
    )
  }

  return (
    <ScanLayer>
      <div className="mx-auto flex h-full w-full max-w-md flex-col">
        {status}
        <ScanHeader
          title="Scan cards"
          back={backLink}
          className="px-4 pt-4 pb-2"
        />
        <ScanFilterBar
          filter={scanFilter.filter}
          facets={scanFilter.facets}
          collectionId={collectionId}
          onChange={scanFilter.setFilter}
        />
        <ScanCamera
          source={source}
          error={error}
          guide
          overlay
          onCapture={() => void capture(source.captureCamera)}
          onFile={(file) => void capture(() => source.fromFile(file))}
        />
        <div className="px-4">
          <PhotoUseNotice />
        </div>
        <ScanSheet
          scans={draft.scans}
          seen={draft.seen}
          facets={scanFilter.facets}
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
          onMark={(key, mark) => {
            forget(cardOf(key))
            draft.mark(key, mark)
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
    </ScanLayer>
  )
}

/**
 * The feedback rows of the Scans that left the Draft Addition, with the photo
 * of each Correction. Only a Correction names a photo, so a confirmation
 * uploads nothing.
 */
function feedbackRows(collectionId: string, scans: Scan[]) {
  const rows: PendingFeedbackRow[] = []
  const photos = new Map<string, Blob>()
  for (const scan of scans) {
    const pick = feedbackChoice(scan.detection, scan)
    if (!pick) continue
    const correction = pick.kind !== "confirmed"
    if (correction && scan.photo) photos.set(scan.key, scan.photo)
    rows.push({
      source: "card_scan",
      collectionId,
      detection: scan.detection,
      ...pick,
      photoId: correction ? scan.key : undefined,
    })
  }
  return { rows, photos }
}
