import { useRef, useState } from "react"
import { useQueries, useQueryClient } from "@tanstack/react-query"
import { Link } from "@tanstack/react-router"
import { Button } from "@/components/ui/button"
import { CardThumb } from "./card-thumb"
import {
  MAX_CARD_IDS_PER_REQUEST,
  MISSING_NAME,
  chunked,
  useDraftCards,
} from "./use-draft-cards"
import { useGetCollection } from "@/generated/endpoints/collections/collections"
import {
  bulkUpdateCollectionEntries,
  getListCollectionEntriesQueryOptions,
} from "@/generated/endpoints/inventory/inventory"
import {
  placeBinderCards,
  useGetBinderPage,
} from "@/generated/endpoints/binder/binder"
import {
  BinderPlacementResultReason,
  InventoryChangeResultStatus,
} from "@/generated/models"
import type { CardSummary } from "@/generated/models"
import {
  NETWORK_ERROR,
  invalidateCollectionCounts,
  invalidateCollectionEntries,
} from "@/lib/collections"
import { refreshAfterSlotWrite } from "@/lib/binder"
import { invalidateMasterInventory } from "@/lib/master-inventory"

class RequestError extends Error {}

export interface ReviewRowData {
  cardId: string
  /** The quantities of this Card's Scans, added together. */
  quantity: number
}

/** Why the Collection declined a Card's copies: the message to show, and the reason code, which a binder gives. */
export interface Decline {
  message: string
  reason?: string
}

/** What one commit did: the copies the Collection took, by card id, the reasons it declined the rest, and whether a request failed. */
export interface CommitResult {
  placed: Record<string, number>
  declined: Record<string, Decline>
  failed: boolean
}

/** What a commit has done so far, filled as each request answers. */
interface CommitProgress {
  placed: Record<string, number>
  declined: Record<string, Decline>
  /** Whether a request left, so the data must refresh even if its answer was lost. */
  sent: boolean
}

/**
 * The review step of a Draft Addition: one row per Card, what each adds, what
 * the Collection already holds, and the commit. On a plain Collection a commit
 * sends absolute targets (held plus added) through the bulk update. Targets are
 * fixed per row once computed and reused by a retry, so a write whose answer was
 * lost is not added twice. The capacity total only warns: the server decides.
 * On a binder a commit places each copy in an empty Slot instead, so there are
 * no targets. A placement is not idempotent, so a lost answer there is reported
 * as unconfirmed, and the step shows the binder's empty Slots.
 */
export function ReviewStep({
  collectionId,
  rows,
  seen,
  declined,
  onCommitted,
  onBack,
}: {
  collectionId: string
  rows: ReviewRowData[]
  seen: Map<string, CardSummary>
  /** Reasons from the last commit, by card id, shown on the rows they belong to. */
  declined: Record<string, Decline>
  onCommitted: (result: CommitResult) => void
  onBack: () => void
}) {
  const queryClient = useQueryClient()
  const {
    cards,
    missing: lookup,
    retry,
  } = useDraftCards(
    rows.map((r) => r.cardId),
    seen
  )
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const targets = useRef(new Map<string, { added: number; target: number }>())

  const ids = rows.map((r) => r.cardId).sort()
  const heldResults = useQueries({
    queries: chunked(ids).map((cardId) =>
      getListCollectionEntriesQueryOptions(collectionId, {
        cardId,
        limit: MAX_CARD_IDS_PER_REQUEST,
      })
    ),
  })
  const held = new Map<string, number>()
  for (const result of heldResults) {
    if (result.data?.status === 200)
      for (const item of result.data.data.data ?? [])
        held.set(item.card.id, item.quantity)
  }
  const heldLoaded = heldResults.every((r) => r.data?.status === 200)

  const collectionQuery = useGetCollection(collectionId)
  const collection =
    collectionQuery.data?.status === 200
      ? collectionQuery.data.data.data
      : undefined
  const binder = collection?.binder
  // A binder's Page read counts its Slots across the whole binder.
  const binderPage = useGetBinderPage(collectionId, 1, {
    query: { enabled: !!binder },
  })
  const emptySlots =
    binder && binderPage.data?.status === 200
      ? binder.capacity - binderPage.data.data.data.cardTotal
      : undefined
  const total = rows.reduce((sum, r) => sum + r.quantity, 0)
  const afterAdding = collection ? collection.cardCount + total : undefined
  const overLimit =
    collection !== undefined &&
    collection.maxCardCount > 0 &&
    afterAdding! > collection.maxCardCount

  // A placement is not idempotent: each copy takes an empty Slot, so a retry
  // after a lost answer would take more. Chunks fill `progress` as they answer.
  async function placeInBinder(progress: CommitProgress) {
    // Rows are already one per Card, as the endpoint requires.
    for (let i = 0; i < rows.length; i += MAX_CARD_IDS_PER_REQUEST) {
      progress.sent = true
      const response = await placeBinderCards(collectionId, {
        items: rows.slice(i, i + MAX_CARD_IDS_PER_REQUEST),
      })
      if (response.status !== 200)
        throw new RequestError(
          response.data.detail ?? "Could not add these cards."
        )
      // Placed per chunk, so a later chunk's failure cannot undo what this one did.
      for (const result of response.data.data ?? []) {
        if (result.placed > 0) progress.placed[result.cardId] = result.placed
        if (result.declined > 0)
          progress.declined[result.cardId] = {
            message: result.message ?? result.reason ?? "Declined.",
            reason: result.reason,
          }
      }
    }
  }

  async function addToPlainCollection(progress: CommitProgress) {
    const missing = rows.filter(
      (r) => targets.current.get(r.cardId)?.added !== r.quantity
    )
    // Fresh, not cached: the targets must come from what the Collection holds now.
    for (const cardId of chunked(missing.map((r) => r.cardId))) {
      const response = await queryClient.fetchQuery({
        ...getListCollectionEntriesQueryOptions(collectionId, {
          cardId,
          limit: MAX_CARD_IDS_PER_REQUEST,
        }),
        staleTime: 0,
      })
      if (response.status !== 200)
        throw new RequestError("Could not read the Collection's quantities.")
      const current = new Map(
        (response.data.data ?? []).map((i) => [i.card.id, i.quantity])
      )
      for (const r of missing.filter((m) => cardId.includes(m.cardId)))
        targets.current.set(r.cardId, {
          added: r.quantity,
          target: (current.get(r.cardId) ?? 0) + r.quantity,
        })
    }
    const items = rows.map((r) => ({
      cardId: r.cardId,
      quantity: targets.current.get(r.cardId)!.target,
    }))
    for (let i = 0; i < items.length; i += MAX_CARD_IDS_PER_REQUEST) {
      progress.sent = true
      const response = await bulkUpdateCollectionEntries(collectionId, {
        items: items.slice(i, i + MAX_CARD_IDS_PER_REQUEST),
      })
      if (response.status !== 200)
        throw new RequestError(
          response.data.detail ?? "Could not add these cards."
        )
      // Applied per chunk, so a later chunk's failure cannot undo what this one did.
      for (const result of response.data.data ?? []) {
        const { added } = targets.current.get(result.cardId)!
        targets.current.delete(result.cardId)
        if (result.status === InventoryChangeResultStatus.declined)
          progress.declined[result.cardId] = {
            message: result.message ?? result.reason ?? "Declined.",
            reason: result.reason,
          }
        else progress.placed[result.cardId] = added
      }
    }
  }

  async function commit() {
    setBusy(true)
    setError(null)
    const progress: CommitProgress = { placed: {}, declined: {}, sent: false }
    let failed = false
    try {
      if (binder) await placeInBinder(progress)
      else await addToPlainCollection(progress)
    } catch (e) {
      failed = true
      const count = Object.keys(progress.placed).length
      if (binder && !(e instanceof RequestError)) {
        // The answer was lost, so the copies may already be in the binder: the
        // draft stays as it is and the user checks the binder before retrying.
        const done =
          count > 0
            ? `${count} of ${rows.length} cards were already added and left your draft. `
            : ""
        setError(
          `${done}The add could not be confirmed. Check the binder before adding again.`
        )
      } else {
        const reason = e instanceof RequestError ? e.message : NETWORK_ERROR
        setError(
          count > 0
            ? `${reason} ${count} of ${rows.length} cards were already added and left your draft; the rest are still here.`
            : `${reason} Your draft is unchanged.`
        )
      }
    } finally {
      setBusy(false)
      onCommitted({
        placed: progress.placed,
        declined: progress.declined,
        failed,
      })
      if (progress.sent) {
        if (binder) refreshAfterSlotWrite(queryClient, collectionId)
        else {
          invalidateMasterInventory(queryClient)
          invalidateCollectionEntries(queryClient, collectionId)
          invalidateCollectionCounts(queryClient, collectionId)
        }
      }
    }
  }

  return (
    <section aria-labelledby="review-heading" className="flex flex-col gap-3">
      <h2 id="review-heading" className="font-heading text-lg font-medium">
        Review and add
      </h2>
      {emptySlots !== undefined && (
        <p className="text-sm">
          Empty Slots: {emptySlots} of {binder?.capacity}
        </p>
      )}
      {!binder && afterAdding !== undefined && (
        <p className="text-sm">
          Collection total after adding: {afterAdding}
          {collection!.maxCardCount > 0
            ? ` of ${collection!.maxCardCount} cards`
            : " cards"}
          {overLimit && (
            <strong className="block text-destructive">
              Over the limit: the Collection may decline some cards.
            </strong>
          )}
        </p>
      )}
      {error && (
        <p role="alert" className="text-sm text-destructive">
          {error}
        </p>
      )}
      {rows.length === 0 ? (
        <p className="text-sm">Nothing left to add.</p>
      ) : (
        <ul className="flex flex-col gap-2">
          {rows.map((row) => {
            const card = cards.get(row.cardId)
            return (
              <li
                key={row.cardId}
                className="flex flex-col gap-1 rounded-xl border p-2"
              >
                <div className="flex items-center gap-2">
                  <CardThumb card={card} />
                  <span className="flex min-w-0 flex-1 flex-col">
                    <span className="truncate text-sm font-medium">
                      {card?.name ?? MISSING_NAME[lookup]}
                    </span>
                    <span className="text-xs">
                      {heldLoaded
                        ? `Holds ${held.get(row.cardId) ?? 0}`
                        : "Holds …"}
                    </span>
                    {!card && lookup === "failed" && (
                      <Button
                        variant="outline"
                        size="xs"
                        className="mt-1 self-start"
                        onClick={retry}
                      >
                        Retry
                      </Button>
                    )}
                  </span>
                  <span className="tabular-nums">+{row.quantity}</span>
                </div>
                {declined[row.cardId] && (
                  <>
                    <p className="text-sm text-destructive">
                      Not added: {declined[row.cardId].message}
                    </p>
                    {declined[row.cardId].reason ===
                      BinderPlacementResultReason.not_enough_empty_slots && (
                      <p className="text-sm">
                        Add a Leaf in the{" "}
                        <Link
                          to="/collections/$collectionId"
                          params={{ collectionId }}
                          className="text-foreground underline"
                        >
                          binder view
                        </Link>{" "}
                        to make room.
                      </p>
                    )}
                  </>
                )}
              </li>
            )
          })}
        </ul>
      )}
      <div className="flex flex-wrap gap-2">
        <Button
          disabled={busy || rows.length === 0}
          onClick={() => void commit()}
        >
          {busy ? "Adding…" : "Add to Collection"}
        </Button>
        <Button variant="outline" disabled={busy} onClick={onBack}>
          Back to scanning
        </Button>
      </div>
    </section>
  )
}
