import { useRef, useState } from "react"
import { useQueryClient } from "@tanstack/react-query"
import {
  useGetCollection,
  useListCollections,
} from "@/generated/endpoints/collections/collections"
import {
  getCountCollectionEntriesQueryKey,
  getListCollectionFacetsQueryKey,
  useCountCollectionEntries,
  useMoveCollectionEntries,
} from "@/generated/endpoints/inventory/inventory"
import type { CardFilterParams, CollectionSummary } from "@/generated/models"
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
import type { CatalogFilters } from "@/lib/catalog-search"
import { invalidateBinder } from "@/lib/binder"
import {
  NETWORK_ERROR,
  errorDetail,
  invalidateCollectionCounts,
  invalidateCollectionEntries,
} from "@/lib/collections"
import { invalidateMasterInventory } from "@/lib/master-inventory"

const MOVE_FAILED = "Could not move these cards."
const COUNT_FAILED = "Could not count the matching cards."

function count(n: number, one: string, many: string) {
  return `${n} ${n === 1 ? one : many}`
}

/** The shared card filters, in the Move's request shape. Unset fields are dropped from the JSON. */
function cardFilter(search: CatalogFilters): CardFilterParams {
  return {
    name: search.name || undefined,
    expansionSetIds: search.expansionSetId,
    localId: search.localId || undefined,
    rarityIds: search.rarityId,
    categories: search.category,
    tags: search.tag,
  }
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
  binder,
  cardIds,
  copies,
  matching,
  flushEdits,
  onCancel,
  onClearMatching,
  onMoved,
}: {
  collectionId: string
  /** A binder source empties the Slots holding the moved Cards. */
  binder: boolean
  cardIds: string[]
  /** Copies across the selected Cards, shown in the confirm dialog. */
  copies: number
  /**
   * Every Card the filter matches, instead of `cardIds`. `total` is the list's
   * count, shown in the bar before the server counts them.
   */
  matching?: { filter: CatalogFilters; total: number }
  /**
   * Sends unsaved quantity edits first, so the Move never runs beside them.
   * Resolves false when an edit to one of these Cards could not be saved.
   */
  flushEdits: (cardIds: string[]) => Promise<boolean>
  onCancel: () => void
  /** Returns to the ticked-card form with nothing ticked. */
  onClearMatching: () => void
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
  // The Cards the sheet and the confirm dialog act on, fixed when "Move to…" is pressed.
  const [picked, setPicked] = useState({ cardIds, copies, matching })
  // Fetched only once the confirm dialog opens, for a filter Move.
  const countQuery = useCountCollectionEntries(
    collectionId,
    picked.matching?.filter,
    {
      query: { enabled: destination !== null && picked.matching !== undefined },
    }
  )

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
  // A filter Move states the server's count; an ids Move states its own.
  // A cached count from an earlier open is not this one: it counts only once
  // the refetch for this open has settled.
  const countSettled = !countQuery.isFetching
  const counted =
    countSettled && countQuery.data?.status === 200
      ? countQuery.data.data.data
      : undefined
  const countError = !countSettled
    ? undefined
    : countQuery.isError
      ? NETWORK_ERROR
      : errorDetail(countQuery.data, COUNT_FAILED)
  const shown = picked.matching
    ? counted
    : { cards: picked.cardIds.length, copies: picked.copies }

  async function openSheet() {
    // The selection can change while the edits save; check and move this one.
    const chosen = { cardIds, copies, matching }
    setNotice(null)
    if (!(await flushEdits(chosen.cardIds))) {
      setNotice(
        "Some quantity edits could not be saved, so nothing was moved. Try again."
      )
      return
    }
    setPicked(chosen)
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
        data: picked.matching
          ? {
              destinationId: target.id,
              filter: cardFilter(picked.matching.filter),
            }
          : { destinationId: target.id, cardIds: picked.cardIds },
      })
      if (response.status === 200) {
        const { cards, copies: moved } = response.data.data
        // Only the two Collections touched by the Move, and the master list, go stale.
        // A binder source also refetches its pages, where the emptied Slots show.
        if (binder) invalidateBinder(queryClient, collectionId)
        else {
          invalidateCollectionEntries(queryClient, collectionId)
          invalidateCollectionCounts(queryClient, collectionId)
        }
        invalidateCollectionEntries(queryClient, target.id)
        invalidateCollectionCounts(queryClient, target.id)
        for (const id of [collectionId, target.id]) {
          void queryClient.invalidateQueries({
            queryKey: getListCollectionFacetsQueryKey(id),
          })
          void queryClient.invalidateQueries({
            queryKey: getCountCollectionEntriesQueryKey(id),
          })
        }
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
          {matching
            ? `All ${matching.total} matching this filter selected`
            : `${cardIds.length} selected`}
        </p>
        {notice && (
          <p role="alert" className="w-full text-sm text-destructive">
            {notice}
          </p>
        )}
        <div className="flex gap-2">
          <Button
            ref={moveButton}
            disabled={!matching && cardIds.length === 0}
            onClick={() => void openSheet()}
          >
            Move to…
          </Button>
          {matching && (
            <Button variant="outline" onClick={onClearMatching}>
              Clear selection
            </Button>
          )}
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
              {shown
                ? `Move ${count(shown.cards, "card", "cards")} to ${destination?.title ?? ""}?`
                : `Move matching cards to ${destination?.title ?? ""}?`}
            </AlertDialogTitle>
            <AlertDialogDescription>
              {shown
                ? `This moves ${count(shown.cards, "card", "cards")} and ${count(shown.copies, "copy", "copies")} from ${sourceTitle} to ${destination?.title ?? ""}. ${binder ? "The Slots holding them will be emptied. " : ""}There is no undo.`
                : countError
                  ? ""
                  : "Counting the matching cards…"}
            </AlertDialogDescription>
          </AlertDialogHeader>
          {(error ?? countError) && (
            <p role="alert" className="text-sm text-destructive">
              {error ?? countError}
            </p>
          )}
          <AlertDialogFooter>
            <AlertDialogCancel disabled={moving}>Cancel</AlertDialogCancel>
            <Button
              disabled={moving || !shown}
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
