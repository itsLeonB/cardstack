import { useDebouncer } from "@tanstack/react-pacer"
import { useRef, useState } from "react"
import { bulkUpdateCollectionEntries } from "@/generated/endpoints/inventory/inventory"
import { InventoryChangeResultStatus } from "@/generated/models"
import { NETWORK_ERROR } from "@/lib/collections"

/** Quiet period after the last quantity change before the batch is sent. */
export const QUANTITY_DEBOUNCE_MS = 500

type Quantities = Record<string, number>
type Errors = Record<string, string>

/**
 * Optimistic, debounced quantity edits for one Collection. Every changed card
 * goes out in a single bulk call, ordered by when it was last changed (oldest
 * first) so that, if capacity runs short, the most recently pressed card is the
 * one declined. `quantities` holds the optimistic value per touched card.
 */
export function useQuantityBatch(collectionId: string, onSaved?: () => void) {
  const [quantities, setQuantities] = useState<Quantities>({})
  const [errors, setErrors] = useState<Errors>({})
  // Map iteration order is insertion order; re-inserting on change keeps it by last change.
  const pending = useRef(new Map<string, number>())
  // Last quantity the server is known to hold, the revert target.
  const confirmed = useRef<Quantities>({})
  const inFlight = useRef(new Set<string>())
  // Bumped on every edit; a batch only acts on a card while its revision is still the latest.
  const revisions = useRef(new Map<string, number>())
  const isLatest = (cardId: string, revision: number) => revisions.current.get(cardId) === revision
  // The debouncer may hold an older closure; always run the latest flush.
  const latestFlush = useRef<() => Promise<void>>(() => Promise.resolve())
  // Leaving the page must not drop edits still waiting on the debounce.
  const debouncer = useDebouncer(() => void latestFlush.current(), {
    wait: QUANTITY_DEBOUNCE_MS,
    onUnmount: (d) => d.flush(),
  })
  // Serializes requests so a later batch can't overtake an earlier one.
  const queue = useRef<Promise<void>>(Promise.resolve())

  function revert(cardId: string, revision: number, quantity: number, message: string) {
    // A newer edit for the same card (pending or already sent) supersedes this outcome, error included.
    if (!isLatest(cardId, revision)) return
    setQuantities((prev) => ({ ...prev, [cardId]: quantity }))
    setErrors((prev) => ({ ...prev, [cardId]: message }))
  }

  async function send(items: [string, number, number][]) {
    const revisionOf = new Map(items.map(([cardId, , revision]) => [cardId, revision]))
    try {
      const response = await bulkUpdateCollectionEntries(collectionId, {
        items: items.map(([cardId, quantity]) => ({ cardId, quantity })),
      })
      if (response.status !== 200) {
        const message = response.data.detail ?? "Could not update these quantities."
        for (const [cardId, , revision] of items) revert(cardId, revision, confirmed.current[cardId], message)
        return
      }
      onSaved?.()
      for (const result of response.data.data ?? []) {
        confirmed.current[result.cardId] = result.quantity
        if (result.status === InventoryChangeResultStatus.declined) {
          revert(
            result.cardId,
            revisionOf.get(result.cardId) ?? -1,
            result.quantity,
            result.message ?? "Could not update this quantity."
          )
        }
      }
    } catch {
      for (const [cardId, , revision] of items) revert(cardId, revision, confirmed.current[cardId], NETWORK_ERROR)
      // The write may have committed with only the response lost, so cached server data can be stale.
      onSaved?.()
    } finally {
      // A newer batch for the card keeps its protection until that batch settles.
      for (const [cardId, , revision] of items) if (isLatest(cardId, revision)) inFlight.current.delete(cardId)
    }
  }

  function flush() {
    debouncer.cancel()
    const items = [...pending.current].map(([cardId, quantity]): [string, number, number] => [
      cardId,
      quantity,
      revisions.current.get(cardId) ?? 0,
    ])
    pending.current.clear()
    for (const [cardId] of items) inFlight.current.add(cardId)
    if (items.length > 0) queue.current = queue.current.then(() => send(items))
    return queue.current
  }

  latestFlush.current = flush

  /** Whether edits are waiting for the next batch (e.g. made while a flush was in flight). */
  function hasPending() {
    return pending.current.size > 0
  }

  /** Whether any edit is unsent or being saved. */
  function isBusy() {
    return pending.current.size > 0 || inFlight.current.size > 0
  }

  /**
   * Drop optimistic values for cards with nothing outstanding. Call when fresh
   * server data arrives so stale overrides can't mask it. Not called after a
   * successful batch, which is what keeps a card at 0 on screen until reload.
   */
  function prune() {
    const settled = (cardId: string) => !pending.current.has(cardId) && !inFlight.current.has(cardId)
    for (const cardId of Object.keys(confirmed.current)) {
      if (settled(cardId)) delete confirmed.current[cardId]
    }
    const keep = <T,>(record: Record<string, T>) =>
      Object.fromEntries(Object.entries(record).filter(([cardId]) => !settled(cardId)))
    setQuantities(keep)
    setErrors(keep)
  }

  function setQuantity(cardId: string, quantity: number, serverQuantity: number) {
    confirmed.current[cardId] ??= serverQuantity
    revisions.current.set(cardId, (revisions.current.get(cardId) ?? 0) + 1)
    pending.current.delete(cardId)
    pending.current.set(cardId, quantity)
    setQuantities((prev) => ({ ...prev, [cardId]: quantity }))
    setErrors(({ [cardId]: _cleared, ...rest }) => rest)
    debouncer.maybeExecute()
  }

  return { quantities, errors, setQuantity, flush, hasPending, isBusy, prune }
}
