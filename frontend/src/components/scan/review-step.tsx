import { useRef, useState } from "react"
import { useQueries, useQueryClient } from "@tanstack/react-query"
import { RiAddLine, RiDeleteBinLine, RiSubtractLine } from "@remixicon/react"
import { Button } from "@/components/ui/button"
import { CardThumb } from "./card-thumb"
import {
  MAX_CARD_IDS_PER_REQUEST,
  chunked,
  useDraftCards,
} from "./use-draft-cards"
import { useGetCollection } from "@/generated/endpoints/collections/collections"
import {
  bulkUpdateCollectionEntries,
  getListCollectionEntriesQueryOptions,
  listCollectionEntries,
} from "@/generated/endpoints/inventory/inventory"
import { InventoryChangeResultStatus } from "@/generated/models"
import type { CardSummary } from "@/generated/models"
import type { DraftRow } from "@/lib/draft-addition"
import {
  NETWORK_ERROR,
  invalidateCollectionCounts,
  invalidateCollectionEntries,
} from "@/lib/collections"
import { invalidateMasterInventory } from "@/lib/master-inventory"

class RequestError extends Error {}

async function heldQuantities(collectionId: string, cardIds: string[]) {
  const held = new Map<string, number>()
  for (const cardId of chunked(cardIds)) {
    const response = await listCollectionEntries(collectionId, {
      cardId,
      limit: MAX_CARD_IDS_PER_REQUEST,
    })
    if (response.status !== 200)
      throw new RequestError("Could not read the Collection's quantities.")
    for (const item of response.data.data ?? [])
      held.set(item.card.id, item.quantity)
  }
  return held
}

/**
 * The review step of a Draft Addition: what each row adds, what the Collection
 * already holds, and the commit. A commit sends absolute targets (held plus
 * added) through the bulk update. Targets are fixed per row once computed and
 * reused by a retry, so a write whose answer was lost is not added twice. The
 * capacity total only warns: the server decides.
 */
export function ReviewStep({
  collectionId,
  rows,
  matchedThisVisit,
  onRaise,
  onLower,
  onRemove,
  onBack,
  onAdded,
}: {
  collectionId: string
  rows: DraftRow[]
  matchedThisVisit: Map<string, CardSummary>
  onRaise: (cardId: string) => void
  onLower: (cardId: string) => void
  onRemove: (cardId: string) => void
  onBack: () => void
  /** Every row was applied: the parent clears the draft and leaves. */
  onAdded: () => void
}) {
  const queryClient = useQueryClient()
  const { cards, settled } = useDraftCards(rows, matchedThisVisit)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [declined, setDeclined] = useState<Record<string, string>>({})
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
    let sent = false
    try {
      const missing = rows.filter(
        (r) => targets.current.get(r.cardId)?.added !== r.quantity
      )
      if (missing.length > 0) {
        const current = await heldQuantities(
          collectionId,
          missing.map((r) => r.cardId)
        )
        for (const r of missing)
          targets.current.set(r.cardId, {
            added: r.quantity,
            target: (current.get(r.cardId) ?? 0) + r.quantity,
          })
      }
      const items = rows.map((r) => ({
        cardId: r.cardId,
        quantity: targets.current.get(r.cardId)!.target,
      }))
      const stillDeclined: Record<string, string> = {}
      for (let i = 0; i < items.length; i += MAX_CARD_IDS_PER_REQUEST) {
        sent = true
        const response = await bulkUpdateCollectionEntries(collectionId, {
          items: items.slice(i, i + MAX_CARD_IDS_PER_REQUEST),
        })
        if (response.status !== 200)
          throw new RequestError(
            response.data.detail ?? "Could not add these cards."
          )
        for (const result of response.data.data ?? []) {
          targets.current.delete(result.cardId)
          if (result.status === InventoryChangeResultStatus.declined)
            stillDeclined[result.cardId] =
              result.message ?? result.reason ?? "Declined."
          else onRemove(result.cardId)
        }
      }
      setDeclined(stillDeclined)
      if (Object.keys(stillDeclined).length === 0) onAdded()
    } catch (e) {
      setError(
        `${e instanceof RequestError ? e.message : NETWORK_ERROR} Your draft is unchanged.`
      )
    } finally {
      setBusy(false)
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
            const name = card?.name ?? "card"
            return (
              <li
                key={row.cardId}
                className="flex flex-col gap-1 rounded-xl border p-2"
              >
                <div className="flex items-center gap-2">
                  <CardThumb card={card} />
                  <span className="flex min-w-0 flex-1 flex-col">
                    <span className="truncate text-sm font-medium">
                      {card?.name ??
                        (settled ? "Card unavailable" : "Loading…")}
                    </span>
                    <span className="text-xs">
                      {heldLoaded
                        ? `Holds ${held.get(row.cardId) ?? 0}`
                        : "Holds …"}
                    </span>
                  </span>
                  <Button
                    variant="outline"
                    size="icon-sm"
                    aria-label={`Lower quantity of ${name}`}
                    disabled={busy || row.quantity <= 1}
                    onClick={() => onLower(row.cardId)}
                  >
                    <RiSubtractLine />
                  </Button>
                  <span className="w-8 text-center tabular-nums">
                    +{row.quantity}
                  </span>
                  <Button
                    variant="outline"
                    size="icon-sm"
                    aria-label={`Raise quantity of ${name}`}
                    disabled={busy}
                    onClick={() => onRaise(row.cardId)}
                  >
                    <RiAddLine />
                  </Button>
                  <Button
                    variant="ghost"
                    size="icon-sm"
                    aria-label={`Remove ${name}`}
                    disabled={busy}
                    onClick={() => onRemove(row.cardId)}
                  >
                    <RiDeleteBinLine />
                  </Button>
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
