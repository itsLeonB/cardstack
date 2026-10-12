import { useRef, useState } from "react"
import { useQueries, useQueryClient } from "@tanstack/react-query"
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
import { InventoryChangeResultStatus } from "@/generated/models"
import type { CardSummary } from "@/generated/models"
import {
  NETWORK_ERROR,
  invalidateCollectionCounts,
  invalidateCollectionEntries,
} from "@/lib/collections"
import { invalidateMasterInventory } from "@/lib/master-inventory"

class RequestError extends Error {}

export interface ReviewRowData {
  cardId: string
  /** The quantities of this Card's Scans, added together. */
  quantity: number
}

/** What one commit did: the Cards the Collection took, the reasons it declined the rest, and whether a request failed. */
export interface CommitResult {
  applied: string[]
  declined: Record<string, string>
  failed: boolean
}

/**
 * The review step of a Draft Addition: one row per Card, what each adds, what
 * the Collection already holds, and the commit. A commit sends absolute targets
 * (held plus added) through the bulk update. Targets are fixed per row once
 * computed and reused by a retry, so a write whose answer was lost is not added
 * twice. The capacity total only warns: the server decides.
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
  declined: Record<string, string>
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
  const total = rows.reduce((sum, r) => sum + r.quantity, 0)
  const afterAdding = collection ? collection.cardCount + total : undefined
  const overLimit =
    collection !== undefined &&
    collection.maxCardCount > 0 &&
    afterAdding! > collection.maxCardCount

  async function commit() {
    setBusy(true)
    setError(null)
    const applied: string[] = []
    const declinedNow: Record<string, string> = {}
    let sent = false
    let failed = false
    try {
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
        sent = true
        const response = await bulkUpdateCollectionEntries(collectionId, {
          items: items.slice(i, i + MAX_CARD_IDS_PER_REQUEST),
        })
        if (response.status !== 200)
          throw new RequestError(
            response.data.detail ?? "Could not add these cards."
          )
        // Applied per chunk, so a later chunk's failure cannot undo what this one did.
        for (const result of response.data.data ?? []) {
          targets.current.delete(result.cardId)
          if (result.status === InventoryChangeResultStatus.declined)
            declinedNow[result.cardId] =
              result.message ?? result.reason ?? "Declined."
          else applied.push(result.cardId)
        }
      }
    } catch (e) {
      failed = true
      const reason = e instanceof RequestError ? e.message : NETWORK_ERROR
      setError(
        applied.length > 0
          ? `${reason} ${applied.length} of ${rows.length} cards were already added and left your draft; the rest are still here.`
          : `${reason} Your draft is unchanged.`
      )
    } finally {
      setBusy(false)
      onCommitted({ applied, declined: declinedNow, failed })
      // A lost answer may hide a committed write, so refresh after any send.
      if (sent) {
        invalidateMasterInventory(queryClient)
        invalidateCollectionCounts(queryClient, collectionId)
        invalidateCollectionEntries(queryClient, collectionId)
      }
    }
  }

  return (
    <section aria-labelledby="review-heading" className="flex flex-col gap-3">
      <h2 id="review-heading" className="font-heading text-lg font-medium">
        Review and add
      </h2>
      {afterAdding !== undefined && (
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
                  <p className="text-sm text-destructive">
                    Not added: {declined[row.cardId]}
                  </p>
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
