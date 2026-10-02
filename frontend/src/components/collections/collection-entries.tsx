import { useEffect, useRef, useState } from "react"
import { keepPreviousData, useQueryClient } from "@tanstack/react-query"
import { useListCatalogSeries } from "@/generated/endpoints/catalog/catalog"
import { useListCollectionFacets } from "@/generated/endpoints/inventory/inventory"
import { QuantityControl } from "@/components/collections/quantity-control"
import { InfiniteCardResults } from "@/components/catalog/infinite-card-results"
import { CatalogFilterPanel } from "@/components/catalog/filter-panel"
import type { CatalogFilters } from "@/lib/catalog-search"
import { hasActiveFilters } from "@/lib/catalog-search"
import {
  invalidateCollectionCounts,
  patchCollectionEntryQuantities,
} from "@/lib/collections"
import { useInfiniteCardResultsProps } from "@/lib/infinite-catalog-cards"
import { useInfiniteCollectionEntries } from "@/lib/infinite-inventory"
import { invalidateMasterInventory } from "@/lib/master-inventory"
import { useQuantityBatch } from "@/lib/use-quantity-batch"

interface CollectionEntriesProps {
  collectionId: string
  search: CatalogFilters
  /** Receives the next filters; the route writes them to the URL. */
  onSearchChange: (next: CatalogFilters) => void
}

export function CollectionEntries({
  collectionId,
  search,
  onSearchChange,
}: CollectionEntriesProps) {
  const queryClient = useQueryClient()
  // A save writes the confirmed quantities into the cached pages instead of
  // refetching them: a refetch costs a request per loaded page and would drop
  // a card saved at 0, whose tile stays until the next real refetch.
  const batch = useQuantityBatch(collectionId, (results) => {
    void patchCollectionEntryQuantities(queryClient, collectionId, results)
    invalidateMasterInventory(queryClient)
    invalidateCollectionCounts(queryClient, collectionId)
  })

  const query = useInfiniteCollectionEntries(
    collectionId,
    search,
    () => !batch.isBusy()
  )
  const facetsQuery = useListCollectionFacets(collectionId, search, {
    query: { placeholderData: keepPreviousData },
  })
  const seriesQuery = useListCatalogSeries()

  const results = useInfiniteCardResultsProps(query)
  const serverQuantities = query.data?.quantities
  const facets =
    facetsQuery.data?.status === 200 ? facetsQuery.data.data.data : undefined
  const series =
    seriesQuery.data?.status === 200
      ? (seriesQuery.data.data.data?.series ?? [])
      : []

  // Fresh server data replaces optimistic values. Appending a page or patching
  // a save changes `data` too, but prune keeps what the cache still agrees with.
  useEffect(
    () => batch.prune((cardId) => serverQuantities?.[cardId]),
    [query.data]
  )

  // Rapid filter clicks build on each other (pendingSearch is what the panel
  // shows meanwhile); the drain flushes edits until none are left, then writes
  // the final filters to the URL once.
  const [pendingSearch, setPendingSearch] = useState<CatalogFilters | null>(
    null
  )
  const target = useRef<CatalogFilters | null>(null)

  async function drain() {
    do {
      await batch.flush()
    } while (batch.hasPending())
    const next = target.current
    target.current = null
    setPendingSearch(null)
    if (next) onSearchChange(next)
  }

  function changeSearch(update: (current: CatalogFilters) => CatalogFilters) {
    const next = update(pendingSearch ?? search)
    const draining = target.current !== null
    target.current = next
    setPendingSearch(next)
    if (!draining) void drain()
  }

  return (
    <section className="flex flex-col gap-6" aria-label="Collection contents">
      <CatalogFilterPanel
        search={pendingSearch ?? search}
        facets={facets}
        series={series}
        onChange={(patch) =>
          changeSearch((current) => ({ ...current, ...patch }))
        }
        onClear={() => changeSearch(() => ({}))}
      />

      <InfiniteCardResults
        {...results}
        emptyMessage={
          hasActiveFilters(search)
            ? "No Cards in this Collection match these filters."
            : "This Collection has no Cards yet. Add some from the catalog."
        }
        renderControl={(card) => {
          const server = serverQuantities?.[card.id] ?? 0
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
    </section>
  )
}
