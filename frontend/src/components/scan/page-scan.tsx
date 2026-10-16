import { useState } from "react"
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
} from "@/generated/models"
import { refreshAfterSlotWrite } from "@/lib/binder"
import { errorDetail, NETWORK_ERROR } from "@/lib/collections"
import { useCameraFrameSource } from "@/lib/frame-source"
import type { FrameSource } from "@/lib/frame-source"
import {
  cellPosition,
  cornersProblem,
  cutPage,
  frameLongSide,
} from "@/lib/page-cut"
import type { Corners, Grid } from "@/lib/page-cut"
import type { PageCell } from "@/lib/use-page-scan"
import {
  changesSlot,
  slotTarget,
  useObjectUrl,
  usePageMatches,
} from "@/lib/use-page-scan"

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
  onClose: () => void
}

/** Scans the viewed Page. Opens the camera only while it is on screen. */
export function PageScanScreen(props: ScanPageProps) {
  const source = useCameraFrameSource()
  return <ScanPageFlow {...props} source={source} />
}

/**
 * Photo, then corners, then the matched cells of the Page. The frame source
 * and the cut are passed in, so a test can fake the camera and the canvas.
 */
export function ScanPageFlow({
  collectionId,
  page,
  source,
  cut = cutPage,
  onClose,
}: ScanPageProps & { source: FrameSource; cut?: typeof cutPage }) {
  const grid: Grid = { rows: page.layout.rows, columns: page.layout.columns }
  const longSide = frameLongSide(grid)
  const [frame, setFrame] = useState<Blob>()
  const [corners, setCorners] = useState<Corners>(CENTRED)
  const [reviewing, setReviewing] = useState(false)
  const [cutting, setCutting] = useState(false)
  // Set by a mark or a candidate pick, and cleared when the photo is cut again or saved.
  const [edited, setEdited] = useState(false)
  // The in-page leave the user asked for while the review holds edits, until they confirm it.
  const [leaving, setLeaving] = useState<(() => void) | null>(null)
  const [error, setError] = useState<string | null>(null)
  // Slots stored as Unidentified because the server did not know the Card.
  const [unknownCards, setUnknownCards] = useState<BinderSlotResult[]>()
  const matches = usePageMatches()
  const frameUrl = useObjectUrl(frame)
  const queryClient = useQueryClient()
  const replace = useReplaceBinderPage()
  // A route change (browser Back, an in-app link) waits for the user while the review holds edits.
  const blocker = useBlocker({
    shouldBlockFn: () => edited,
    enableBeforeUnload: () => edited,
    withResolver: true,
  })

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
      matches.start(await cut(frame, corners, grid))
      setEdited(false)
      setReviewing(true)
    } catch {
      setError("Could not cut this photo. Move the corners and try again.")
    } finally {
      setCutting(false)
    }
  }

  // Leaving after an edit would drop it, so the leave waits for the user to confirm.
  function leave(action: () => void) {
    if (edited) setLeaving(() => action)
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
            <PageReview
              cells={matches.cells}
              grid={grid}
              changed={changed}
              onRetry={matches.retry}
              onChoose={chooseCard}
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
                Adjust corners
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
