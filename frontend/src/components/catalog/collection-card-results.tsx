import { useEffect } from "react"
import { useQueries, useQueryClient } from "@tanstack/react-query"
import {
  getListCollectionEntriesQueryOptions,
  getListCollectionFacetsQueryKey,
} from "@/generated/endpoints/inventory/inventory"
import type { CardSummary } from "@/generated/models"
import { InfiniteCardResults } from "@/components/catalog/infinite-card-results"
import type { InfiniteCardResultsProps } from "@/components/catalog/infinite-card-results"
import { QuantityControl } from "@/components/collections/quantity-control"
import {
  invalidateCollectionCounts,
  invalidateCollectionEntries,
} from "@/lib/collections"
import { CATALOG_PAGE_SIZE } from "@/lib/infinite-catalog-cards"
import { invalidateMasterInventory } from "@/lib/master-inventory"
import { useQuantityBatch } from "@/lib/use-quantity-batch"

// Ids per lookup: the catalog's page size, so a slice is one loaded page, and
// under the endpoint's cap of 100 per request.
const LOOKUP_SLICE = CATALOG_PAGE_SIZE

function sliceCardIds(cards: CardSummary[]) {
  const slices: string[][] = []
  for (let start = 0; start < cards.length; start += LOOKUP_SLICE) {
    slices.push(cards.slice(start, start + LOOKUP_SLICE).map((card) => card.id))
  }
  return slices
}

/**
 * Catalog results with a quantity control per Card for one Collection. Mount
 * with `key={collectionId}` so switching Collections resets the edit batch.
 */
export function CollectionCardResults({
  collectionId,
  ...results
}: Omit<InfiniteCardResultsProps, "renderControl"> & { collectionId: string }) {
  const queryClient = useQueryClient()
  const batch = useQuantityBatch(collectionId, (saved) => {
    // Refreshes the lookups that hold the saved cards (plain keys) and the
    // Collection page's infinite list (`'infinite'` keys) if it is cached; the
    // latter refetches on mount anyway. A lost response (no results) may have
    // changed any card, so then every lookup refreshes. Facets are cached, so
    // refresh them too.
    invalidateCollectionEntries(
      queryClient,
      collectionId,
      saved.length > 0 ? saved.map((result) => result.cardId) : undefined
    )
    void queryClient.invalidateQueries({
      queryKey: getListCollectionFacetsQueryKey(collectionId),
    })
    invalidateMasterInventory(queryClient)
    invalidateCollectionCounts(queryClient, collectionId)
  })

  // One lookup per slice of loaded cards, so a page that appends adds a query
  // instead of changing the key (and dropping the data) of the earlier ones.
  // The endpoint caps `limit` at 100, and an empty cardId list means "no
  // restriction" to the API, so there is never a slice of 0 ids.
  const slices = sliceCardIds(results.cards)
  const lookups = useQueries({
    queries: slices.map((cardIds) =>
      getListCollectionEntriesQueryOptions(
        collectionId,
        { cardId: cardIds, limit: cardIds.length },
        // gcTime 0 + refetchOnMount: no cached page can show (or seed a revert with) a quantity from before an edit.
        { query: { gcTime: 0, refetchOnMount: "always" } }
      )
    ),
    combine: (queries) => {
      const quantities: Record<string, number> = {}
      const loaded: Record<string, true> = {}
      queries.forEach((query, index) => {
        if (query.data?.status !== 200) return
        for (const item of query.data.data.data ?? []) {
          quantities[item.card.id] = item.quantity
        }
        for (const cardId of slices[index] ?? []) loaded[cardId] = true
      })
      return {
        quantities,
        loaded,
        failed: queries.some(
          (query) => query.isError || (query.data && query.data.status !== 200)
        ),
      }
    },
  })

  // Drops overrides the looked-up rows now contradict (changed elsewhere); agreeing ones stay, so a declined card keeps its error.
  useEffect(
    () =>
      batch.prune((cardId) =>
        lookups.loaded[cardId] ? (lookups.quantities[cardId] ?? 0) : undefined
      ),
    [lookups]
  )

  return (
    <>
      {lookups.failed && (
        <p role="alert" className="text-sm text-destructive">
          Could not load this Collection&apos;s quantities.
        </p>
      )}
      <InfiniteCardResults
        {...results}
        renderControl={(card) => {
          if (!lookups.loaded[card.id]) return null
          const server = lookups.quantities[card.id] ?? 0
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
