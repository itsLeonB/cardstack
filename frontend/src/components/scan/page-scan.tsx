import { useEffect, useState } from "react"
import { useQueryClient } from "@tanstack/react-query"
import { useBlocker } from "@tanstack/react-router"
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
import { PageCorners } from "@/components/scan/page-corners"
import { PageReview } from "@/components/scan/page-review"
import {
  cameraMessage,
  ChoosePhotoButton,
} from "@/components/scan/photo-source-controls"
import { useReplaceBinderPage } from "@/generated/endpoints/binder/binder"
import type {
  BinderPage,
  BinderPageSlotBody,
  BinderSlotResult,
  CardSummary,
} from "@/generated/models"
import { refreshAfterSlotWrite } from "@/lib/binder"
import { errorDetail, NETWORK_ERROR } from "@/lib/collections"
import { removePageReview, savePageReview } from "@/lib/page-review-store"
import type { PendingPageReview } from "@/lib/page-review-store"
import { invalidatePendingReviews } from "@/lib/use-pending-reviews"
import { useCameraFrameSource } from "@/lib/frame-source"
import type { FrameSource } from "@/lib/frame-source"
import {
  cellPosition,
  cornersProblem,
  cutPage,
  frameLongSide,
} from "@/lib/page-cut"
import type { Corners, Grid } from "@/lib/page-cut"
import { useSession } from "@/lib/session"
import type { PageCell } from "@/lib/use-page-scan"
import {
  cellMark,
  changesSlot,
  slotTarget,
  useObjectUrl,
  usePageMatches,
} from "@/lib/use-page-scan"
import { feedbackChoice, sendScanFeedback } from "@/lib/scan-feedback"
import type { Choice, PendingFeedbackRow } from "@/lib/scan-feedback"

// Corners start as a centred rectangle, a little inside the photo's edges.
const CENTRED: Corners = [
  { x: 0.15, y: 0.1 },
  { x: 0.85, y: 0.1 },
  { x: 0.85, y: 0.9 },
  { x: 0.15, y: 0.9 },
]

interface ScanPageProps {
  /** The binder's Collection, which the Confirm writes to. */
  collectionId: string
  /** The Page being viewed: its number, Layout and Slots. */
  page: BinderPage
  /** A pending review to pick up where it was left, instead of starting at the photo. */
  resume?: PendingPageReview
  onClose: () => void
}

/** Scans the viewed Page. Opens the camera only while it is on screen. */
export function PageScanScreen(props: ScanPageProps) {
  const source = useCameraFrameSource()
  const { user } = useSession()
  // The binder is only reached signed in, so a review always has an owner.
  if (!user) return null
  return <ScanPageFlow {...props} userId={user.id} source={source} />
}

// The Page photo is one upload shared by every Correction on the Page (ADR-0022).
const PAGE_PHOTO = "page"

// The choice a cell stands at, as slotTarget saves it: its mark, its Reprint or its Card. A cell with neither is Unidentified.
function cellChoice(cell: PageCell): Choice {
  const mark = cellMark(cell)
  if (mark) return { mark }
  if (cell.cardId === undefined) return { mark: "unidentified" }
  return { cardId: cell.cardId, reprintPosition: cell.reprintPosition }
}

/**
 * The feedback rows of a confirmed Page Review, built from the review as it
 * stood at Confirm. The API refuses a binder row without its grid and
 * corners, so a review saved before they were kept records nothing.
 */
function pageFeedbackRows(
  collectionId: string,
  page: number,
  grid: Grid,
  cells: PageCell[],
  cut: { photo?: Blob; corners?: Corners }
) {
  const rows: PendingFeedbackRow[] = []
  const photos = new Map<string, Blob>()
  if (!cut.corners) return { rows, photos }
  cells.forEach((cell, index) => {
    const pick = feedbackChoice(cell.detection, cellChoice(cell))
    if (!pick) return
    const { row, column } = cellPosition(index, grid.columns)
    const correction = pick.kind !== "confirmed"
    rows.push({
      source: "binder_page",
      collectionId,
      detection: cell.detection,
      page,
      slotRow: row,
      slotColumn: column,
      gridRows: grid.rows,
      gridColumns: grid.columns,
      corners: cut.corners,
      ...pick,
      photoId: correction && cut.photo ? PAGE_PHOTO : undefined,
    })
  })
  if (cut.photo && rows.some((row) => row.photoId))
    photos.set(PAGE_PHOTO, cut.photo)
  return { rows, photos }
}

/**
 * Photo, then corners, then the matched cells of the Page. The frame source
 * and the cut are passed in, so a test can fake the camera and the canvas.
 */
export function ScanPageFlow({
  collectionId,
  page,
  resume,
  userId,
  source,
  cut = cutPage,
  onClose,
}: ScanPageProps & {
  userId: string
  source: FrameSource
  cut?: typeof cutPage
}) {
  const grid: Grid = { rows: page.layout.rows, columns: page.layout.columns }
  const longSide = frameLongSide(grid)
  const [frame, setFrame] = useState<Blob>()
  const [corners, setCorners] = useState<Corners>(CENTRED)
  // The photo and corners the cells were cut from. Only a successful cut sets them, so they always match the cells, even when a new photo is taken before cutting.
  const [cutFrom, setCutFrom] = useState<{ photo?: Blob; corners?: Corners }>({
    photo: resume?.photo,
    corners: resume?.corners,
  })
  const [reviewing, setReviewing] = useState(resume !== undefined)
  // Set once a save fails (no IndexedDB): the review is then kept in memory only.
  const [memoryOnly, setMemoryOnly] = useState(false)
  const [cutting, setCutting] = useState(false)
  // Set by a mark or a candidate pick, and cleared when the photo is cut again or saved.
  const [edited, setEdited] = useState(false)
  // The in-page leave the user asked for while the review holds edits, until they confirm it.
  const [leaving, setLeaving] = useState<(() => void) | null>(null)
  const [error, setError] = useState<string | null>(null)
  // Slots stored as Unidentified because the server did not know the Card.
  const [unknownCards, setUnknownCards] = useState<BinderSlotResult[]>()
  const matches = usePageMatches(resume?.cells)
  const frameUrl = useObjectUrl(frame)
  const queryClient = useQueryClient()
  const replace = useReplaceBinderPage()
  // A route change (browser Back, an in-app link) waits for the user only while a memory-only review holds edits.
  const blocker = useBlocker({
    shouldBlockFn: () => memoryOnly && edited,
    enableBeforeUnload: () => memoryOnly && edited,
    withResolver: true,
  })

  // Every change to the cells is saved, so the review survives leaving or a reload.
  useEffect(() => {
    if (memoryOnly || matches.cells.length === 0) return
    savePageReview({
      userId,
      collectionId,
      page: page.page,
      grid,
      cells: matches.cells,
      photo: cutFrom.photo,
      corners: cutFrom.corners,
    })
      .then(() => invalidatePendingReviews(queryClient))
      .catch(() => setMemoryOnly(true))
  }, [matches.cells])

  const targets = matches.cells.map(slotTarget)

  function slotAt(index: number) {
    const { row, column } = cellPosition(index, grid.columns)
    return page.slots?.find(
      (slot) => slot.row === row && slot.column === column
    )
  }
  const changed = targets.map(
    (target, index) => target !== null && changesSlot(target, slotAt(index))
  )
  const cornersError = cornersProblem(corners)

  async function takeFrame(take: () => Promise<Blob>) {
    setError(null)
    try {
      setFrame(await take())
    } catch {
      setError("Could not capture that photo. Try again, or choose a photo.")
    }
  }

  async function cutAndMatch() {
    if (!frame || cornersProblem(corners)) return
    setError(null)
    setCutting(true)
    try {
      const cells = await cut(frame, corners, grid)
      setCutFrom({ photo: frame, corners })
      matches.start(cells)
      setEdited(false)
      setReviewing(true)
    } catch {
      setError("Could not cut this photo. Move the corners and try again.")
    } finally {
      setCutting(false)
    }
  }

  // A memory-only review loses its edits on leave, so the leave waits for the user to confirm.
  function leave(action: () => void) {
    if (memoryOnly && edited) setLeaving(() => action)
    else action()
  }

  function markCell(index: number, mark: PageCell["mark"]) {
    setEdited(true)
    matches.mark(index, mark)
  }

  function chooseCard(index: number, cardId: string) {
    setEdited(true)
    matches.choose(index, cardId)
  }

  function reprintCell(index: number, card: CardSummary, position: number) {
    setEdited(true)
    matches.reprint(index, card, position)
  }

  function confirm() {
    // A cell still queued or loading holds the Confirm, marked or not.
    if (matches.pending) return
    const slots = targets.filter(
      (target): target is BinderPageSlotBody => target !== null
    )
    setError(null)
    replace.mutate(
      { id: collectionId, page: page.page, data: { slots } },
      {
        onSuccess: (response) => {
          if (response.status !== 200) {
            setError(errorDetail(response, "Could not save this Page.") ?? null)
            return
          }
          setEdited(false)
          // Built before the review is removed, and not awaited: the save never waits for it.
          const { rows, photos } = pageFeedbackRows(
            collectionId,
            page.page,
            grid,
            matches.cells,
            cutFrom
          )
          void sendScanFeedback(rows, photos)
          // A review that cannot be cleared stays listed, where it can be discarded.
          void removePageReview({
            userId,
            collectionId,
            page: page.page,
          })
            .catch(() => undefined)
            .then(() => invalidatePendingReviews(queryClient))
          refreshAfterSlotWrite(queryClient, collectionId)
          const unknown = (response.data.data.slots ?? []).filter(
            (slot) => slot.reason === "unknown_card"
          )
          if (unknown.length > 0) setUnknownCards(unknown)
          else onClose()
        },
        onError: () => setError(NETWORK_ERROR),
      }
    )
  }

  // Closing the dialog keeps the review; leaving takes the action or the route that asked.
  const dialogOpen = leaving !== null || blocker.status === "blocked"
  function dismissLeave() {
    setLeaving(null)
    if (blocker.status === "blocked") blocker.reset()
  }
  function confirmLeave() {
    leaving?.()
    setLeaving(null)
    if (blocker.status === "blocked") blocker.proceed()
  }

  const hasCamera = source.camera === "starting" || source.camera === "ready"

  return (
    <div className="fixed inset-x-0 top-[calc(3.5rem+1px)] bottom-0 z-30 overflow-y-auto bg-background">
      <div className="mx-auto flex w-full max-w-md flex-col gap-4 px-4 py-6">
        <div className="flex items-center justify-between gap-2">
          <h2 className="text-lg font-medium">Scan Page {page.page}</h2>
          <Button variant="outline" size="sm" onClick={() => leave(onClose)}>
            Back to binder
          </Button>
        </div>
        {error && (
          <p role="alert" className="text-sm text-destructive">
            {error}
          </p>
        )}

        {unknownCards ? (
          <div className="flex flex-col gap-3">
            <p role="status" className="text-sm">
              Saved. The app did not recognise the Card in these Slots, so they
              are Unidentified:
            </p>
            {unknownCards.map((slot) => (
              <p key={`${slot.row},${slot.column}`} className="text-sm">
                Row {slot.row}, column {slot.column}
              </p>
            ))}
            <div>
              <Button onClick={onClose}>Done</Button>
            </div>
          </div>
        ) : reviewing ? (
          <div className="flex flex-col gap-3">
            <p role="status" className="text-sm">
              {matches.pending ? "Matching the cells…" : "Check each cell."}
            </p>
            {memoryOnly && (
              <p role="status" className="text-sm">
                This review is not being saved, because this browser cannot
                store it. Leaving this screen discards it.
              </p>
            )}
            <PageReview
              cells={matches.cells}
              grid={grid}
              changed={changed}
              onRetry={matches.retry}
              onChoose={chooseCard}
              onReprint={reprintCell}
              onMark={markCell}
            />
            <div className="flex flex-wrap gap-2">
              <Button
                disabled={matches.pending || replace.isPending}
                onClick={confirm}
              >
                {replace.isPending ? "Saving…" : "Confirm"}
              </Button>
              <Button
                variant="outline"
                disabled={replace.isPending}
                onClick={() =>
                  leave(() => {
                    setReviewing(false)
                    setEdited(false)
                  })
                }
              >
                {frame ? "Adjust corners" : "Scan again"}
              </Button>
            </div>
          </div>
        ) : frame && frameUrl ? (
          <div className="flex flex-col gap-3">
            <p className="text-sm">
              Drag the four corners onto the outer corners of the pockets.
            </p>
            <PageCorners
              src={frameUrl}
              corners={corners}
              onChange={setCorners}
            />
            {cornersError && (
              <p role="alert" className="text-sm text-destructive">
                {cornersError}
              </p>
            )}
            <div className="flex flex-wrap gap-2">
              <Button
                disabled={cutting || cornersError !== null}
                onClick={() => void cutAndMatch()}
              >
                {cutting ? "Cutting…" : "Cut and match"}
              </Button>
              <Button
                variant="outline"
                disabled={cutting}
                onClick={() => setFrame(undefined)}
              >
                Retake photo
              </Button>
            </div>
          </div>
        ) : (
          <div className="flex flex-col gap-3">
            {hasCamera ? (
              <div className="aspect-[3/4] overflow-hidden bg-black">
                <video
                  ref={source.videoRef}
                  playsInline
                  muted
                  aria-label="Camera viewfinder"
                  className="size-full object-contain"
                />
              </div>
            ) : (
              <p className="text-sm">{cameraMessage(source.camera)}</p>
            )}
            <div className="flex flex-wrap gap-2">
              {hasCamera && (
                <Button
                  size="lg"
                  disabled={source.camera !== "ready"}
                  onClick={() =>
                    void takeFrame(() => source.captureFrame(longSide))
                  }
                >
                  Capture
                </Button>
              )}
              <ChoosePhotoButton
                onFile={(file) =>
                  void takeFrame(() => source.frameFromFile(file, longSide))
                }
              />
            </div>
          </div>
        )}

        <AlertDialog
          open={dialogOpen}
          onOpenChange={(open) => {
            if (!open) dismissLeave()
          }}
        >
          <AlertDialogContent>
            <AlertDialogHeader>
              <AlertDialogTitle>Leave this review?</AlertDialogTitle>
              <AlertDialogDescription>
                Your edits to these cells are not saved, and leaving discards
                them.
              </AlertDialogDescription>
            </AlertDialogHeader>
            <AlertDialogFooter>
              <AlertDialogCancel>Keep editing</AlertDialogCancel>
              <AlertDialogAction onClick={confirmLeave}>
                Leave
              </AlertDialogAction>
            </AlertDialogFooter>
          </AlertDialogContent>
        </AlertDialog>
      </div>
    </div>
  )
}
