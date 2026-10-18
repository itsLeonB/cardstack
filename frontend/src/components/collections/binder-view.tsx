import { useEffect, useRef, useState } from "react"
import { keepPreviousData, useQueryClient } from "@tanstack/react-query"
import { toast } from "sonner"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import {
  useAddBinderLeaf,
  useGetBinderPage,
  useRemoveBinder,
  useRemoveBinderLeaf,
  useUpdateBinderGrid,
} from "@/generated/endpoints/binder/binder"
import type { BinderSlot, BinderSlotTarget } from "@/generated/models"
import { BinderLayoutForm } from "@/components/collections/binder-layout-form"
import type { BinderLayoutValues } from "@/components/collections/binder-layout-form"
import { BinderMoveBanner } from "@/components/collections/binder-move-banner"
import { BinderPageGrid } from "@/components/collections/binder-page-grid"
import { SlotPanel } from "@/components/collections/binder-slot-panel"
import { useBinderMove } from "@/components/collections/use-binder-move"
import { PageScanScreen } from "@/components/scan/page-scan"
import { PendingReviewBanner } from "@/components/scan/pending-review-banner"
import { PendingReviewNotice } from "@/components/scan/pending-review-notice"
import { ScanThisPageButton } from "@/components/scan/scan-this-page-button"
import { invalidatePendingReviews } from "@/lib/use-pending-reviews"
import type { PendingPageReview } from "@/lib/page-review-store"
import {
  applyLeafChange,
  discardMismatchedReviews,
} from "@/lib/page-review-leaf"
import type { LeafChange } from "@/lib/page-review-leaf"
import { useSession } from "@/lib/session"
import { errorDetail, NETWORK_ERROR } from "@/lib/collections"
import {
  invalidateBinder,
  pageOf,
  refreshAfterRemoveBinder,
  slotKey,
} from "@/lib/binder"
import type { SlotRef } from "@/lib/binder"

// A swipe shorter than this is a scroll or a tap, not a page turn.
const SWIPE_MIN_PX = 50

/** Turns through one binder Page at a time, with its Leaf actions. */
export function BinderView({
  collectionId,
  jumpTo,
  onJumpApplied,
}: {
  collectionId: string
  /** A Slot to show: turns to its Page and highlights it. A new object per tap. */
  jumpTo?: BinderSlotTarget | null
  /** Clears `jumpTo` once applied, so a remount does not jump again. */
  onJumpApplied?: () => void
}) {
  const queryClient = useQueryClient()
  const [requestedPage, setRequestedPage] = useState(1)
  const [jumpText, setJumpText] = useState("")
  const [jumpError, setJumpError] = useState<string | null>(null)
  const [leafError, setLeafError] = useState<string | null>(null)
  const [editingLayout, setEditingLayout] = useState(false)
  const [layoutError, setLayoutError] = useState<string | null>(null)
  const [confirmingRemove, setConfirmingRemove] = useState(false)
  const [scanning, setScanning] = useState(false)
  const [resuming, setResuming] = useState<PendingPageReview>()
  // Bumped once the stored reviews change, so the banner reads them again.
  const [reviewsVersion, setReviewsVersion] = useState(0)
  const { user } = useSession()
  const userId = user?.id
  // Set once the Layout is gone, so the Page queries stop before the
  // Collection refetch unmounts this view.
  const [detached, setDetached] = useState(false)
  const touchStart = useRef<{ x: number; y: number } | null>(null)
  // The Slot whose actions are open. Only the Page it was tapped on shows it.
  const [selectedSlot, setSelectedSlot] = useState<SlotRef | null>(null)
  // The tapped tile, so focus can go back to it when the panel closes.
  const openerRef = useRef<HTMLButtonElement | null>(null)
  // A Slot a jump brought into view. It is only highlighted, not opened.
  const [highlighted, setHighlighted] = useState<SlotRef | null>(null)
  const pageRef = useRef<HTMLElement>(null)
  // Set by a jump, so the Slot takes focus once its Page is on screen.
  const jumpFocus = useRef(false)
  const { moving, moveError, startMove, cancelMove, moveTo } =
    useBinderMove(collectionId)

  // Keep the previous Page on screen while the next one loads, so a turn does
  // not flash the grid away.
  const query = useGetBinderPage(collectionId, requestedPage, {
    query: { placeholderData: keepPreviousData, enabled: !detached },
  })
  const binderPage =
    query.data?.status === 200 ? query.data.data.data : undefined
  const totalPages = binderPage
    ? pageOf(binderPage.layout.leafCount, "back")
    : 0
  const loadError = errorDetail(query.data, NETWORK_ERROR)
  const currentPage = binderPage?.page ?? requestedPage

  const addLeafMutation = useAddBinderLeaf()
  const removeLeafMutation = useRemoveBinderLeaf()
  const updateGridMutation = useUpdateBinderGrid()
  const removeBinderMutation = useRemoveBinder()

  // The entries list sits below the binder, so the grid scrolls back up to it.
  useEffect(() => {
    if (!jumpTo) return
    jumpFocus.current = true
    setRequestedPage(jumpTo.page)
    setHighlighted(jumpTo)
    pageRef.current?.scrollIntoView({ block: "center" })
    onJumpApplied?.()
  }, [jumpTo, onJumpApplied])

  // Waits for the jumped-to Page: the previous one stays on screen while the
  // next one loads.
  useEffect(() => {
    if (!jumpFocus.current || !binderPage) return
    if (highlighted?.page !== binderPage.page) return
    jumpFocus.current = false
    pageRef.current
      ?.querySelector<HTMLElement>('[aria-pressed="true"]')
      ?.focus()
  }, [binderPage, highlighted])

  // A review saved for another Layout no longer fits its cells, so it goes when
  // the binder opens.
  const rows = binderPage?.layout.rows
  const columns = binderPage?.layout.columns
  useEffect(() => {
    if (userId === undefined || rows === undefined || columns === undefined) {
      return
    }
    // Without IndexedDB nothing was saved, so there is nothing to discard.
    discardMismatchedReviews({ userId, collectionId }, { rows, columns }).then(
      (pages) => {
        if (pages.length === 0) return
        toast.info(
          `Discarded the pending review of Page ${pages.join(" and Page ")}: the Layout changed.`
        )
        setReviewsVersion((version) => version + 1)
        void invalidatePendingReviews(queryClient)
      },
      () => undefined
    )
  }, [queryClient, userId, collectionId, rows, columns])

  // Steps from the requested Page, not the shown one, so two quick taps move
  // two Pages even before the first has loaded.
  function stepPage(delta: number) {
    setRequestedPage((page) => Math.min(Math.max(page + delta, 1), totalPages))
  }

  function handleJump(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault()
    setJumpError(null)
    const page = Number(jumpText)
    if (!Number.isInteger(page) || page < 1 || page > totalPages) {
      setJumpError(`Enter a Page from 1 to ${totalPages}.`)
      return
    }
    setRequestedPage(page)
  }

  function handleTouchStart(event: React.TouchEvent) {
    const touch = event.touches[0]
    touchStart.current = touch ? { x: touch.clientX, y: touch.clientY } : null
  }

  function handleTouchEnd(event: React.TouchEvent) {
    const start = touchStart.current
    touchStart.current = null
    const end = event.changedTouches[0]
    if (!start || !end) return
    const dx = end.clientX - start.x
    const dy = end.clientY - start.y
    // A mostly vertical drag is a scroll, not a page turn.
    if (Math.abs(dx) < SWIPE_MIN_PX || Math.abs(dx) <= Math.abs(dy)) return
    stepPage(dx < 0 ? 1 : -1)
  }

  // Pages renumber after a Leaf change, so a Slot picked or highlighted before
  // it no longer names the same Slot.
  function clearSlotMarks() {
    cancelMove()
    setHighlighted(null)
    setSelectedSlot(null)
  }

  // Renumbers the stored reviews of this binder. Without IndexedDB there are none.
  function moveReviews(change: LeafChange): Promise<number[]> {
    if (userId === undefined) return Promise.resolve([])
    return applyLeafChange({ userId, collectionId }, change).catch(() => [])
  }

  function addLeaf(placement: "before" | "after") {
    if (!binderPage) return
    setLeafError(null)
    addLeafMutation.mutate(
      { id: collectionId, data: { leaf: binderPage.leaf, placement } },
      {
        onSuccess: (response) => {
          if (response.status !== 201) {
            toast.error(errorDetail(response, "Could not add a Leaf.") ?? "")
            return
          }
          invalidateBinder(queryClient, collectionId)
          clearSlotMarks()
          // Turn to the new Leaf's front Page, so the blank Leaf is on screen.
          const leaf =
            placement === "after" ? binderPage.leaf + 1 : binderPage.leaf
          // The reviews move first, so the Page turned to shows its own review.
          void moveReviews({ kind: "add", leaf }).then(() => {
            setReviewsVersion((version) => version + 1)
            void invalidatePendingReviews(queryClient)
            setRequestedPage(pageOf(leaf, "front"))
          })
        },
        onError: () => toast.error(NETWORK_ERROR),
      }
    )
  }

  function removeLeaf() {
    if (!binderPage) return
    setLeafError(null)
    removeLeafMutation.mutate(
      { id: collectionId, leaf: binderPage.leaf },
      {
        onSuccess: (response) => {
          // A refusal names the Page holding the card, so it stays beside the
          // controls rather than in a toast.
          if (response.status !== 200) {
            setLeafError(
              errorDetail(response, "Could not remove this Leaf.") ?? null
            )
            return
          }
          invalidateBinder(queryClient, collectionId)
          clearSlotMarks()
          // The Leaf after the removed one takes its place, or the last Leaf
          // when the removed one was the last.
          const nextPage = Math.min(
            pageOf(binderPage.leaf, "front"),
            pageOf(response.data.data.leafCount, "back")
          )
          void moveReviews({ kind: "remove", leaf: binderPage.leaf }).then(
            (pages) => {
              if (pages.length > 0) {
                toast.info(
                  `Discarded the pending review of Page ${pages.join(" and Page ")} with its Leaf.`
                )
              }
              setReviewsVersion((version) => version + 1)
              void invalidatePendingReviews(queryClient)
              setRequestedPage(nextPage)
            }
          )
        },
        onError: () => toast.error(NETWORK_ERROR),
      }
    )
  }

  function saveLayout(values: BinderLayoutValues) {
    setLayoutError(null)
    updateGridMutation.mutate(
      {
        id: collectionId,
        data: { rows: values.rows, columns: values.columns },
      },
      {
        onSuccess: (response) => {
          if (response.status !== 200) {
            setLayoutError(
              errorDetail(response, "Could not change this Layout.") ?? null
            )
            return
          }
          invalidateBinder(queryClient, collectionId)
          setEditingLayout(false)
        },
        onError: () => setLayoutError(NETWORK_ERROR),
      }
    )
  }

  function removeBinder() {
    removeBinderMutation.mutate(
      { id: collectionId },
      {
        onSuccess: (response) => {
          if (response.status !== 204) {
            toast.error(
              errorDetail(response, "Could not remove this binder.") ?? ""
            )
            return
          }
          setDetached(true)
          refreshAfterRemoveBinder(queryClient, collectionId)
        },
        onError: () => toast.error(NETWORK_ERROR),
      }
    )
  }

  function startSlotMove(slot: BinderSlot, source: SlotRef) {
    setSelectedSlot(null)
    startMove(slot, source)
  }

  if (!binderPage) {
    return (
      <div className="flex flex-col gap-3">
        {query.isError || loadError ? (
          <div role="alert" className="flex flex-col items-start gap-2">
            <p className="text-sm text-destructive">
              {loadError ?? NETWORK_ERROR}
            </p>
            <Button variant="outline" onClick={() => void query.refetch()}>
              Retry
            </Button>
          </div>
        ) : (
          <p className="text-sm">Loading this Page…</p>
        )}
      </div>
    )
  }

  if (scanning) {
    return (
      <PageScanScreen
        collectionId={collectionId}
        page={binderPage}
        resume={resuming}
        onClose={() => {
          setScanning(false)
          setResuming(undefined)
        }}
      />
    )
  }

  const { layout, cardTotal, unidentifiedCount, leaf, side } = binderPage
  const empty = cardTotal === 0
  const openSlot =
    selectedSlot?.page === binderPage.page ? selectedSlot : undefined
  const shownSlot =
    openSlot &&
    binderPage.slots?.find(
      (slot) => slot.row === openSlot.row && slot.column === openSlot.column
    )
  const highlightedHere =
    highlighted?.page === binderPage.page ? highlighted : undefined
  const movingHere =
    moving?.source.page === binderPage.page ? moving.source : undefined
  function closeSlot() {
    setSelectedSlot(null)
    openerRef.current?.focus()
  }
  const leafBusy = addLeafMutation.isPending || removeLeafMutation.isPending

  return (
    <div className="flex flex-col gap-6">
      <div className="flex flex-col gap-1 text-sm">
        <p>
          Cards and Unidentified: {cardTotal} of {layout.capacity} slots
        </p>
        <p>Unidentified: {unidentifiedCount}</p>
        <p>
          {layout.rows} × {layout.columns} slots per Page · {layout.leafCount}{" "}
          Leaves
        </p>
      </div>

      <div className="flex flex-wrap items-center gap-2">
        <Button
          variant="outline"
          aria-label="Previous page"
          disabled={requestedPage <= 1}
          onClick={() => stepPage(-1)}
        >
          Previous
        </Button>
        <p className="text-sm font-medium">
          Page {currentPage} of {totalPages}
        </p>
        <Button
          variant="outline"
          aria-label="Next page"
          disabled={requestedPage >= totalPages}
          onClick={() => stepPage(1)}
        >
          Next
        </Button>
        {/* The browser's own range check would replace the message below. */}
        <form
          noValidate
          onSubmit={handleJump}
          className="flex items-center gap-2"
        >
          <Input
            aria-label="Page number"
            type="number"
            min={1}
            max={totalPages}
            step={1}
            inputMode="numeric"
            className="w-24"
            value={jumpText}
            onChange={(event) => setJumpText(event.target.value)}
          />
          <Button type="submit" variant="outline" aria-label="Go to page">
            Go
          </Button>
        </form>
        {jumpError && (
          <p role="alert" className="text-sm text-destructive">
            {jumpError}
          </p>
        )}
        <PendingReviewNotice
          collectionId={collectionId}
          onJump={setRequestedPage}
        />
        <PendingReviewBanner
          key={reviewsVersion}
          collectionId={collectionId}
          page={binderPage.page}
          onResume={(review) => {
            setResuming(review)
            setScanning(true)
          }}
        />
        <ScanThisPageButton
          collectionId={collectionId}
          page={binderPage.page}
          onScan={(review) => {
            setResuming(review)
            setScanning(true)
          }}
        />
      </div>

      {moving && (
        <BinderMoveBanner
          source={moving.source}
          label={moving.label}
          error={moveError}
          onCancel={cancelMove}
        />
      )}

      <section
        ref={pageRef}
        aria-label="Binder page"
        className="flex flex-col gap-3"
        onTouchStart={handleTouchStart}
        onTouchEnd={handleTouchEnd}
      >
        <p className="text-sm">
          Leaf {leaf}, {side}
        </p>
        <BinderPageGrid
          page={binderPage}
          selected={openSlot ?? highlightedHere}
          picked={movingHere}
          onSelect={(slot, tile) => {
            const at: SlotRef = {
              page: binderPage.page,
              row: slot.row,
              column: slot.column,
            }
            if (moving) {
              moveTo(at)
              return
            }
            setHighlighted(null)
            openerRef.current = tile
            setSelectedSlot(at)
          }}
        />
        {openSlot && shownSlot && (
          <SlotPanel
            key={slotKey(openSlot)}
            collectionId={collectionId}
            at={openSlot}
            slot={shownSlot}
            onClose={closeSlot}
            onMove={() => startSlotMove(shownSlot, openSlot)}
          />
        )}
      </section>

      <div className="flex flex-col gap-2">
        <div className="flex flex-wrap gap-2">
          <Button
            variant="outline"
            disabled={leafBusy}
            onClick={() => addLeaf("before")}
          >
            Add Leaf before
          </Button>
          <Button
            variant="outline"
            disabled={leafBusy}
            onClick={() => addLeaf("after")}
          >
            Add Leaf after
          </Button>
          <Button variant="outline" disabled={leafBusy} onClick={removeLeaf}>
            Remove Leaf
          </Button>
        </div>
        {leafError && (
          <p role="alert" className="text-sm text-destructive">
            {leafError}
          </p>
        )}
      </div>

      <div className="flex flex-col gap-3 border-t pt-4">
        {!empty && (
          <p className="text-sm">
            The Layout and its rows and columns can change only while every Slot
            is empty.
          </p>
        )}
        {empty && !editingLayout && (
          <div className="flex flex-wrap gap-2">
            <Button variant="outline" onClick={() => setEditingLayout(true)}>
              Change layout
            </Button>
            {!confirmingRemove && (
              <Button
                variant="outline"
                onClick={() => setConfirmingRemove(true)}
              >
                Remove binder
              </Button>
            )}
            {confirmingRemove && (
              <>
                <Button
                  variant="destructive"
                  disabled={removeBinderMutation.isPending}
                  onClick={removeBinder}
                >
                  Confirm remove binder
                </Button>
                <Button
                  variant="outline"
                  onClick={() => setConfirmingRemove(false)}
                >
                  Keep binder
                </Button>
              </>
            )}
          </div>
        )}
        {empty && editingLayout && (
          <BinderLayoutForm
            initial={{ rows: layout.rows, columns: layout.columns }}
            withLeafCount={false}
            submitLabel="Save layout"
            pendingLabel="Saving..."
            isPending={updateGridMutation.isPending}
            errorMessage={layoutError}
            onCancel={() => setEditingLayout(false)}
            onSubmit={saveLayout}
          />
        )}
      </div>
    </div>
  )
}
