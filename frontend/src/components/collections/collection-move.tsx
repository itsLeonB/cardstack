import { useRef, useState } from "react"
import { useQueryClient } from "@tanstack/react-query"
import {
  useGetCollection,
  useListCollections,
} from "@/generated/endpoints/collections/collections"
import { useMoveCollectionEntries } from "@/generated/endpoints/inventory/inventory"
import type { CollectionSummary } from "@/generated/models"
import { Button } from "@/components/ui/button"
import {
  AlertDialog,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import {
  NETWORK_ERROR,
  errorDetail,
  invalidateCollectionCounts,
  invalidateCollectionEntries,
} from "@/lib/collections"
import { invalidateMasterInventory } from "@/lib/master-inventory"

const MOVE_FAILED = "Could not move these cards."

function count(n: number, one: string, many: string) {
  return `${n} ${n === 1 ? one : many}`
}

/** Why a Collection can't take the Move, or undefined when it can. */
function refusal(collection: CollectionSummary, sourceId: string) {
  if (collection.id === sourceId)
    return "This is the Collection you are moving from."
  // A binder's Cards sit in Slots, and a Move chooses no Slot (GLOSSARY: Move).
  if (collection.binder)
    return "A binder needs a Slot for each card, and a Move chooses none."
  return undefined
}

/**
 * Selection mode's action bar, the destination sheet and the confirm dialog.
 * The Move is all or nothing, so a failure leaves the selection in place.
 */
export function CollectionMoveBar({
  collectionId,
  cardIds,
  copies,
  flushEdits,
  onCancel,
  onMoved,
}: {
  collectionId: string
  cardIds: string[]
  /** Copies across the selected Cards, shown in the confirm dialog. */
  copies: number
  /**
   * Sends unsaved quantity edits first, so the Move never runs beside them.
   * Resolves false when an edit to a selected Card could not be saved.
   */
  flushEdits: () => Promise<boolean>
  onCancel: () => void
  /** Receives the result to announce; the caller leaves selection mode. */
  onMoved: (announcement: string) => void
}) {
  const queryClient = useQueryClient()
  const sourceQuery = useGetCollection(collectionId)
  const collectionsQuery = useListCollections()
  const move = useMoveCollectionEntries()
  // Dialogs return focus here when they close without moving.
  const moveButton = useRef<HTMLButtonElement>(null)
  const [sheetOpen, setSheetOpen] = useState(false)
  const [destination, setDestination] = useState<CollectionSummary | null>(null)
  const [moving, setMoving] = useState(false)
  const [error, setError] = useState<string | null>(null)
  // Shown in the bar when edits could not be saved before opening the sheet.
  const [notice, setNotice] = useState<string | null>(null)

  const sourceTitle =
    sourceQuery.data?.status === 200
      ? sourceQuery.data.data.data.title
      : "this Collection"
  const listLoading = collectionsQuery.isPending
  const listFailed =
    collectionsQuery.isError ||
    (collectionsQuery.data !== undefined &&
      collectionsQuery.data.status !== 200)
  const collections =
    collectionsQuery.data?.status === 200
      ? (collectionsQuery.data.data.data ?? [])
      : []

  async function openSheet() {
    setNotice(null)
    if (!(await flushEdits())) {
      setNotice(
        "Some quantity edits could not be saved, so nothing was moved. Try again."
      )
      return
    }
    setSheetOpen(true)
  }

  function closeConfirm() {
    setDestination(null)
    setError(null)
  }

  async function confirm(target: CollectionSummary) {
    setMoving(true)
    setError(null)
    try {
      const response = await move.mutateAsync({
        id: collectionId,
        data: { cardIds, destinationId: target.id },
      })
      if (response.status === 200) {
        const { cards, copies: moved } = response.data.data
        // Only the two Collections touched by the Move, and the master list, go stale.
        invalidateCollectionEntries(queryClient, collectionId)
        invalidateCollectionCounts(queryClient, collectionId)
        invalidateCollectionEntries(queryClient, target.id)
        invalidateCollectionCounts(queryClient, target.id)
        invalidateMasterInventory(queryClient)
        onMoved(
          `Moved ${count(cards, "card", "cards")} (${count(moved, "copy", "copies")}) to ${target.title}.`
        )
        return
      }
      setError(errorDetail(response, MOVE_FAILED) ?? MOVE_FAILED)
    } catch {
      setError(NETWORK_ERROR)
    } finally {
      setMoving(false)
    }
  }

  return (
    <>
      <div className="sticky bottom-0 z-20 flex flex-wrap items-center justify-between gap-3 border-t bg-background/95 px-4 pt-3 pb-[calc(0.75rem+env(safe-area-inset-bottom))] backdrop-blur">
        <p role="status" className="text-sm font-medium">
          {`${cardIds.length} selected`}
        </p>
        {notice && (
          <p role="alert" className="w-full text-sm text-destructive">
            {notice}
          </p>
        )}
        <div className="flex gap-2">
          <Button
            ref={moveButton}
            disabled={cardIds.length === 0}
            onClick={() => void openSheet()}
          >
            Move to…
          </Button>
          <Button variant="outline" onClick={onCancel}>
            Cancel
          </Button>
        </div>
      </div>

      <Dialog open={sheetOpen} onOpenChange={setSheetOpen}>
        <DialogContent
          finalFocus={moveButton}
          className="top-auto right-0 bottom-0 left-0 max-w-none translate-x-0 translate-y-0 rounded-b-none"
        >
          <DialogHeader>
            <DialogTitle>Move to…</DialogTitle>
            <DialogDescription>
              Choose the Collection for the selected cards.
            </DialogDescription>
          </DialogHeader>
          {listLoading && (
            <p role="status" className="text-sm">
              Loading your Collections…
            </p>
          )}
          {listFailed && (
            <p role="alert" className="text-sm text-destructive">
              Could not load your Collections.
            </p>
          )}
          <div className="flex flex-col gap-2">
            {collections.map((collection) => {
              const reason = refusal(collection, collectionId)
              return (
                <Button
                  key={collection.id}
                  variant="outline"
                  className="h-auto flex-col items-start gap-1 py-2"
                  // aria-disabled, not disabled: a keyboard user can still focus it and hear the reason.
                  aria-disabled={Boolean(reason)}
                  aria-describedby={
                    reason ? `move-refusal-${collection.id}` : undefined
                  }
                  onClick={() => {
                    if (reason) return
                    setSheetOpen(false)
                    setDestination(collection)
                  }}
                >
                  <span>{collection.title}</span>
                  {reason && (
                    <span
                      id={`move-refusal-${collection.id}`}
                      className="text-xs font-normal"
                    >
                      {reason}
                    </span>
                  )}
                </Button>
              )
            })}
          </div>
        </DialogContent>
      </Dialog>

      <AlertDialog
        open={destination !== null}
        onOpenChange={(open) => {
          if (!open && !moving) closeConfirm()
        }}
      >
        <AlertDialogContent finalFocus={moveButton}>
          <AlertDialogHeader>
            <AlertDialogTitle>
              {`Move ${count(cardIds.length, "card", "cards")} to ${destination?.title ?? ""}?`}
            </AlertDialogTitle>
            <AlertDialogDescription>
              {`This moves ${count(cardIds.length, "card", "cards")} and ${count(copies, "copy", "copies")} from ${sourceTitle} to ${destination?.title ?? ""}. There is no undo.`}
            </AlertDialogDescription>
          </AlertDialogHeader>
          {error && (
            <p role="alert" className="text-sm text-destructive">
              {error}
            </p>
          )}
          <AlertDialogFooter>
            <AlertDialogCancel disabled={moving}>Cancel</AlertDialogCancel>
            <Button
              disabled={moving}
              onClick={() => {
                if (destination) void confirm(destination)
              }}
            >
              {moving ? "Moving…" : "Move"}
            </Button>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </>
  )
}
