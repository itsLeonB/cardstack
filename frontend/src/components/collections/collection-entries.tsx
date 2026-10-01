import { useState } from "react"
import { keepPreviousData } from "@tanstack/react-query"
import { useListCatalogSeries } from "@/generated/endpoints/catalog/catalog"
import {
  useListCollectionEntries,
  useListCollectionFacets,
} from "@/generated/endpoints/inventory/inventory"
import { AddEntry } from "@/components/collections/add-entry"
import { QuantityControl } from "@/components/collections/quantity-control"
import { CardResults } from "@/components/catalog/card-results"
import { CatalogFilterPanel } from "@/components/catalog/filter-panel"
import type { CatalogSearch } from "@/lib/catalog-search"
import { hasActiveFilters, toFacetParams } from "@/lib/catalog-search"
import { useQuantityBatch } from "@/lib/use-quantity-batch"

interface CollectionEntriesProps {
  collectionId: string
  search: CatalogSearch
  /** Receives the next search (filters or page); the route writes it to the URL. */
  onSearchChange: (next: CatalogSearch) => void
}

export function CollectionEntries({ collectionId, search, onSearchChange }: CollectionEntriesProps) {
  const [addError, setAddError] = useState<string | null>(null)
  const batch = useQuantityBatch(collectionId)

  // Always refetch and never keep the entry list around after leaving: a card
  // taken to 0 stays on screen only until the user comes back.
  const query = useListCollectionEntries(collectionId, search, {
    query: { gcTime: 0, refetchOnMount: "always", placeholderData: keepPreviousData },
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

  // Pending edits go out before the filters/page change so they aren't lost.
  function changeSearch(next: CatalogSearch) {
    void batch.flush().then(() => onSearchChange(next))
  }

  return (
    <section className="flex flex-col gap-6" aria-label="Collection contents">
      <AddEntry collectionId={collectionId} onError={setAddError} />

      {addError && (
        <p role="alert" className="text-sm text-destructive">
          {addError}
        </p>
      )}

      <CatalogFilterPanel
        search={search}
        facets={facets}
        series={series}
        onChange={(patch) => changeSearch({ ...search, ...patch, page: 1 })}
        onClear={() => changeSearch({ page: 1 })}
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
            : "This Collection has no Cards yet. Search above to add one."
        }
        onPageChange={(page) => changeSearch({ ...search, page })}
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
