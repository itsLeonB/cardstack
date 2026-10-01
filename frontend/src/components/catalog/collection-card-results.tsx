import { useEffect } from "react"
import { useQueryClient } from "@tanstack/react-query"
import {
  getListCollectionEntriesQueryKey,
  getListCollectionFacetsQueryKey,
  useListCollectionEntries,
} from "@/generated/endpoints/inventory/inventory"
import { CardResults } from "@/components/catalog/card-results"
import type { CardResultsProps } from "@/components/catalog/card-results"
import { QuantityControl } from "@/components/collections/quantity-control"
import { useQuantityBatch } from "@/lib/use-quantity-batch"

/**
 * Catalog results with a quantity control per Card for one Collection. Mount
 * with `key={collectionId}` so switching Collections resets the edit batch.
 */
export function CollectionCardResults({
  collectionId,
  ...results
}: Omit<CardResultsProps, "renderControl"> & { collectionId: string }) {
  const queryClient = useQueryClient()
  const batch = useQuantityBatch(collectionId, () => {
    // Collection detail refetches entries on mount; facets are cached, so refresh them.
    void queryClient.invalidateQueries({ queryKey: getListCollectionEntriesQueryKey(collectionId) })
    void queryClient.invalidateQueries({ queryKey: getListCollectionFacetsQueryKey(collectionId) })
  })

  // An empty cardId list means "no restriction" to the API, so never ask with one.
  const cardIds = results.cards.map((card) => card.id)
  const query = useListCollectionEntries(
    collectionId,
    { cardId: cardIds, limit: cardIds.length },
    { query: { enabled: cardIds.length > 0 } }
  )
  const response = query.data?.status === 200 ? query.data.data : undefined
  const serverQuantities = new Map((response?.data ?? []).map((item) => [item.card.id, item.quantity]))

  // Fresh server data replaces optimistic values.
  useEffect(() => batch.prune(), [query.data])

  return (
    <>
      {(query.isError || (query.data && !response)) && (
        <p role="alert" className="text-sm text-destructive">
          Could not load this Collection&apos;s quantities.
        </p>
      )}
      <CardResults
        {...results}
        renderControl={(card) => {
          if (!response) return null
          const server = serverQuantities.get(card.id) ?? 0
          return (
            <QuantityControl
              cardName={card.name}
              value={batch.quantities[card.id] ?? server}
              error={batch.errors[card.id]}
              onChange={(quantity) => batch.setQuantity(card.id, quantity, server)}
            />
          )
        }}
      />
    </>
  )
}
