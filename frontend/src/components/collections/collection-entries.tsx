import { useEffect, useRef, useState } from "react"
import { keepPreviousData, useQueryClient } from "@tanstack/react-query"
import { useListCatalogSeries } from "@/generated/endpoints/catalog/catalog"
import {
  useListCollectionEntries,
  useListCollectionFacets,
} from "@/generated/endpoints/inventory/inventory"
import { QuantityControl } from "@/components/collections/quantity-control"
import { CardResults } from "@/components/catalog/card-results"
import { CatalogFilterPanel } from "@/components/catalog/filter-panel"
import type { CatalogSearch } from "@/lib/catalog-search"
import { hasActiveFilters, toFacetParams } from "@/lib/catalog-search"
import { invalidateMasterInventory } from "@/lib/master-inventory"
import { useQuantityBatch } from "@/lib/use-quantity-batch"

interface CollectionEntriesProps {
  collectionId: string
  search: CatalogSearch
  /** Receives the next search (filters or page); the route writes it to the URL. */
  onSearchChange: (next: CatalogSearch) => void
}

export function CollectionEntries({ collectionId, search, onSearchChange }: CollectionEntriesProps) {
  const queryClient = useQueryClient()
  const batch = useQuantityBatch(collectionId, () => {
    invalidateMasterInventory(queryClient)
  })

  // Always refetch and never keep the entry list around after leaving: a card
  // taken to 0 stays on screen only until the user comes back. Refocusing the
  // tab counts as coming back, but not while edits are unsent or saving.
  const query = useListCollectionEntries(collectionId, search, {
    query: {
      gcTime: 0,
      refetchOnMount: "always",
      refetchOnWindowFocus: () => !batch.isBusy(),
      placeholderData: keepPreviousData,
    },
  })
  const facetsQuery = useListCollectionFacets(collectionId, toFacetParams(search), {
    query: { placeholderData: keepPreviousData },
  })
  const seriesQuery = useListCatalogSeries()

  const result = query.data?.status === 200 ? query.data.data : undefined
  const items = result?.data ?? []
  const serverQuantities = new Map(items.map((item) => [item.card.id, item.quantity]))
  const facets = facetsQuery.data?.status === 200 ? facetsQuery.data.data.data : undefined
  const series = seriesQuery.data?.status === 200 ? (seriesQuery.data.data.data?.series ?? []) : []
  const loadError =
    query.data && query.data.status !== 200
      ? (query.data.data.detail ?? "Could not load this Collection's Cards.")
      : undefined

  // Fresh server data replaces optimistic values (not after a save, which leaves data untouched).
  useEffect(() => batch.prune(), [query.data])

  // Rapid filter/page clicks build on each other (pendingSearch is what the
  // panel shows meanwhile); the drain flushes edits until none are left, then
  // writes the final search to the URL once.
  const [pendingSearch, setPendingSearch] = useState<CatalogSearch | null>(null)
  const target = useRef<CatalogSearch | null>(null)

  async function drain() {
    do {
      await batch.flush()
    } while (batch.hasPending())
    const next = target.current
    target.current = null
    setPendingSearch(null)
    if (next) onSearchChange(next)
  }

  function changeSearch(update: (current: CatalogSearch) => CatalogSearch) {
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
        onChange={(patch) => changeSearch((current) => ({ ...current, ...patch, page: 1 }))}
        onClear={() => changeSearch(() => ({ page: 1 }))}
      />

      <CardResults
        cards={items.map((item) => item.card)}
        total={result?.meta.total ?? 0}
        page={search.page}
        limit={result?.meta.limit ?? 24}
        isPending={query.isPending}
        isError={query.isError || Boolean(loadError)}
        errorMessage={loadError}
        emptyMessage={
          hasActiveFilters(search)
            ? "No Cards in this Collection match these filters."
            : "This Collection has no Cards yet. Add some from the catalog."
        }
        onPageChange={(page) => changeSearch((current) => ({ ...current, page }))}
        renderControl={(card) => {
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
    </section>
  )
}
