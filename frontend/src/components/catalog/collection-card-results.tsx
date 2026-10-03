import { useEffect } from "react"
import { useQueryClient } from "@tanstack/react-query"
import {
  getListCollectionFacetsQueryKey,
  useListCollectionEntries,
} from "@/generated/endpoints/inventory/inventory"
import { InfiniteCardResults } from "@/components/catalog/infinite-card-results"
import type { InfiniteCardResultsProps } from "@/components/catalog/infinite-card-results"
import { QuantityControl } from "@/components/collections/quantity-control"
import {
  invalidateCollectionCounts,
  invalidateCollectionEntries,
} from "@/lib/collections"
import { invalidateMasterInventory } from "@/lib/master-inventory"
import { useQuantityBatch } from "@/lib/use-quantity-batch"

const MAX_LOOKUP = 100

/**
 * Catalog results with a quantity control per Card for one Collection. Mount
 * with `key={collectionId}` so switching Collections resets the edit batch.
 */
export function CollectionCardResults({
  collectionId,
  ...results
}: Omit<InfiniteCardResultsProps, "renderControl"> & { collectionId: string }) {
  const queryClient = useQueryClient()
  const batch = useQuantityBatch(collectionId, () => {
    // Refreshes this page's quantity lookup (plain keys) and the Collection
    // page's infinite list (`'infinite'` keys) if it is cached; the latter
    // refetches on mount anyway. Facets are cached, so refresh them too.
    invalidateCollectionEntries(queryClient, collectionId)
    void queryClient.invalidateQueries({
      queryKey: getListCollectionFacetsQueryKey(collectionId),
    })
    invalidateMasterInventory(queryClient)
    invalidateCollectionCounts(queryClient, collectionId)
  })

  // An empty cardId list means "no restriction" to the API, so never ask with one.
  // The endpoint caps `limit` at 100, so cards past the first 100 get no
  // control until the lookup is paged (ticket 37).
  const cardIds = results.cards.slice(0, MAX_LOOKUP).map((card) => card.id)
  const query = useListCollectionEntries(
    collectionId,
    { cardId: cardIds, limit: cardIds.length },
    // gcTime 0 + refetchOnMount: no cached page can show (or seed a revert with) a quantity from before an edit.
    {
      query: {
        enabled: cardIds.length > 0,
        gcTime: 0,
        refetchOnMount: "always",
      },
    }
  )
  const response = query.data?.status === 200 ? query.data.data : undefined
  const lookedUpIds = new Set(cardIds)
  const serverQuantities = new Map(
    (response?.data ?? []).map((item) => [item.card.id, item.quantity])
  )

  // Drops overrides the looked-up rows now contradict (changed elsewhere); agreeing ones stay, so a declined card keeps its error.
  useEffect(
    () =>
      batch.prune((cardId) =>
        lookedUpIds.has(cardId)
          ? (serverQuantities.get(cardId) ?? 0)
          : undefined
      ),
    [query.data]
  )

  return (
    <>
      {(query.isError || (query.data && !response)) && (
        <p role="alert" className="text-sm text-destructive">
          Could not load this Collection&apos;s quantities.
        </p>
      )}
      <InfiniteCardResults
        {...results}
        renderControl={(card) => {
          if (!response || !lookedUpIds.has(card.id)) return null
          const server = serverQuantities.get(card.id) ?? 0
          return (
            <QuantityControl
              cardName={card.name}
              value={batch.quantities[card.id] ?? server}
              error={batch.errors[card.id]}
              onChange={(quantity) =>
                batch.setQuantity(card.id, quantity, server)
              }
            />
          )
        }}
      />
    </>
  )
}
