import { useRef, useState } from "react"
import { keepPreviousData, useQueryClient } from "@tanstack/react-query"
import { toast } from "sonner"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import {
  useAddBinderLeaf,
  useCreateBinder,
  useGetBinderPage,
  useRemoveBinder,
  useRemoveBinderLeaf,
  useUpdateBinderGrid,
} from "@/generated/endpoints/binder/binder"
import { BinderLayoutForm } from "@/components/collections/binder-layout-form"
import type { BinderLayoutValues } from "@/components/collections/binder-layout-form"
import { BinderPageGrid } from "@/components/collections/binder-page-grid"
import { SlotPanel } from "@/components/collections/binder-slot-panel"
import { errorDetail, NETWORK_ERROR } from "@/lib/collections"
import { invalidateBinder, refreshAfterRemoveBinder } from "@/lib/binder"

// A swipe shorter than this is a scroll or a tap, not a page turn.
const SWIPE_MIN_PX = 50

/** Turns through one binder Page at a time, with its Leaf actions. */
export function BinderView({ collectionId }: { collectionId: string }) {
  const queryClient = useQueryClient()
  const [requestedPage, setRequestedPage] = useState(1)
  const [jumpText, setJumpText] = useState("")
  const [jumpError, setJumpError] = useState<string | null>(null)
  const [leafError, setLeafError] = useState<string | null>(null)
  const [editingLayout, setEditingLayout] = useState(false)
  const [layoutError, setLayoutError] = useState<string | null>(null)
  const [confirmingRemove, setConfirmingRemove] = useState(false)
  // Set once the Layout is gone, so the Page queries stop before the
  // Collection refetch unmounts this view.
  const [detached, setDetached] = useState(false)
  const touchStart = useRef<{ x: number; y: number } | null>(null)
  // The Slot whose actions are open. Only the Page it was tapped on shows it.
  const [selectedSlot, setSelectedSlot] = useState<{
    page: number
    row: number
    column: number
  } | null>(null)

  // Keep the previous Page on screen while the next one loads, so a turn does
  // not flash the grid away.
  const query = useGetBinderPage(collectionId, requestedPage, {
    query: { placeholderData: keepPreviousData, enabled: !detached },
  })
  const binderPage =
    query.data?.status === 200 ? query.data.data.data : undefined
  const totalPages = binderPage ? binderPage.layout.leafCount * 2 : 0
  const loadError = errorDetail(query.data, NETWORK_ERROR)
  const currentPage = binderPage?.page ?? requestedPage

  const addLeafMutation = useAddBinderLeaf()
  const removeLeafMutation = useRemoveBinderLeaf()
  const updateGridMutation = useUpdateBinderGrid()
  const removeBinderMutation = useRemoveBinder()

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
          // Turn to the new Leaf's front Page, so the blank Leaf is on screen.
          const leaf =
            placement === "after" ? binderPage.leaf + 1 : binderPage.leaf
          setRequestedPage(2 * leaf - 1)
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
          // The Leaf after the removed one takes its place, or the last Leaf
          // when the removed one was the last.
          setRequestedPage(
            Math.min(2 * binderPage.leaf - 1, 2 * response.data.data.leafCount)
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

  const { layout, cardTotal, unidentifiedCount, leaf, side } = binderPage
  const empty = cardTotal === 0
  const shownSlot =
    selectedSlot?.page === binderPage.page
      ? binderPage.slots?.find(
          (slot) =>
            slot.row === selectedSlot.row && slot.column === selectedSlot.column
        )
      : undefined
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
      </div>

      <section
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
          selected={shownSlot}
          onSelect={(slot) =>
            setSelectedSlot({
              page: binderPage.page,
              row: slot.row,
              column: slot.column,
            })
          }
        />
        {shownSlot && (
          <SlotPanel
            key={`${binderPage.page},${shownSlot.row},${shownSlot.column}`}
            collectionId={collectionId}
            page={binderPage.page}
            slot={shownSlot}
            onClose={() => setSelectedSlot(null)}
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

/** Turns a plain, empty Collection into a binder. */
export function CreateBinderPanel({ collectionId }: { collectionId: string }) {
  const queryClient = useQueryClient()
  const [errorMessage, setErrorMessage] = useState<string | null>(null)
  const createMutation = useCreateBinder()

  function createBinder(values: BinderLayoutValues) {
    setErrorMessage(null)
    createMutation.mutate(
      {
        id: collectionId,
        data: {
          rows: values.rows,
          columns: values.columns,
          leafCount: values.leafCount,
        },
      },
      {
        onSuccess: (response) => {
          if (response.status !== 201) {
            setErrorMessage(
              errorDetail(response, "Could not create this binder.") ?? null
            )
            return
          }
          invalidateBinder(queryClient, collectionId)
        },
        onError: () => setErrorMessage(NETWORK_ERROR),
      }
    )
  }

  return (
    <section aria-label="Make a binder" className="flex flex-col gap-4">
      <h2 className="text-lg font-medium">Make this a binder</h2>
      <p className="text-sm">
        Choose the Layout of each Page and how many Leaves the binder starts
        with. Its Slots then hold the cards.
      </p>
      <div className="max-w-lg">
        <BinderLayoutForm
          withLeafCount
          submitLabel="Create binder"
          pendingLabel="Creating..."
          isPending={createMutation.isPending}
          errorMessage={errorMessage}
          onSubmit={createBinder}
        />
      </div>
    </section>
  )
}
