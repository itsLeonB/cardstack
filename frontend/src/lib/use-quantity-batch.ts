import { useCallback, useEffect, useRef, useState } from "react"
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
export function useQuantityBatch(collectionId: string) {
  const [quantities, setQuantities] = useState<Quantities>({})
  const [errors, setErrors] = useState<Errors>({})
  // Map iteration order is insertion order; re-inserting on change keeps it by last change.
  const pending = useRef(new Map<string, number>())
  // Last quantity the server is known to hold, the revert target.
  const confirmed = useRef<Quantities>({})
  const timer = useRef<ReturnType<typeof setTimeout> | undefined>(undefined)
  // Serializes requests so a later batch can't overtake an earlier one.
  const queue = useRef<Promise<void>>(Promise.resolve())

  function revert(cardId: string, quantity: number, message: string) {
    // A newer pending change for the same card wins over the revert.
    if (!pending.current.has(cardId)) {
      setQuantities((prev) => ({ ...prev, [cardId]: quantity }))
    }
    setErrors((prev) => ({ ...prev, [cardId]: message }))
  }

  async function send(items: [string, number][]) {
    try {
      const response = await bulkUpdateCollectionEntries(collectionId, {
        items: items.map(([cardId, quantity]) => ({ cardId, quantity })),
      })
      if (response.status !== 200) {
        const message = response.data.detail ?? "Could not update these quantities."
        for (const [cardId] of items) revert(cardId, confirmed.current[cardId] ?? 0, message)
        return
      }
      for (const result of response.data.data ?? []) {
        confirmed.current[result.cardId] = result.quantity
        if (result.status === InventoryChangeResultStatus.declined) {
          revert(result.cardId, result.quantity, result.message ?? "Could not update this quantity.")
        }
      }
    } catch {
      for (const [cardId] of items) revert(cardId, confirmed.current[cardId] ?? 0, NETWORK_ERROR)
    }
  }

  // ponytail: reads only refs, so a stale closure is harmless; stable for effect cleanup.
  const flush = useCallback(() => {
    clearTimeout(timer.current)
    const items = [...pending.current]
    pending.current.clear()
    if (items.length > 0) queue.current = queue.current.then(() => send(items))
    return queue.current
    // oxlint-disable-next-line react-hooks/exhaustive-deps -- send only closes over collectionId and refs
  }, [collectionId])

  function setQuantity(cardId: string, quantity: number, serverQuantity: number) {
    confirmed.current[cardId] ??= serverQuantity
    pending.current.delete(cardId)
    pending.current.set(cardId, quantity)
    setQuantities((prev) => ({ ...prev, [cardId]: quantity }))
    setErrors(({ [cardId]: _cleared, ...rest }) => rest)
    clearTimeout(timer.current)
    timer.current = setTimeout(() => void flush(), QUANTITY_DEBOUNCE_MS)
  }

  // Leaving the page must not drop edits still waiting on the debounce.
  useEffect(() => () => void flush(), [flush])

  return { quantities, errors, setQuantity, flush }
}
