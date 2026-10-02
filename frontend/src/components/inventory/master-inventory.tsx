import { keepPreviousData } from "@tanstack/react-query"
import { useListCatalogSeries } from "@/generated/endpoints/catalog/catalog"
import {
  useListMasterInventory,
  useListMasterInventoryFacets,
} from "@/generated/endpoints/inventory/inventory"
import { CardResults } from "@/components/catalog/card-results"
import { CatalogFilterPanel } from "@/components/catalog/filter-panel"
import type { CatalogSearch } from "@/lib/catalog-search"
import { hasActiveFilters, toFacetParams } from "@/lib/catalog-search"
import { errorDetail } from "@/lib/collections"

interface MasterInventoryProps {
  search: CatalogSearch
  /** Receives the next search (filters or page); the route writes it to the URL. */
  onSearchChange: (next: CatalogSearch) => void
}

/** The user's Cards with the total quantity owned across all their Collections (read-only). */
export function MasterInventory({
  search,
  onSearchChange,
}: MasterInventoryProps) {
  // gcTime 0 + refetchOnMount: no cached page can show totals from before a Collection change.
  const query = useListMasterInventory(search, {
    query: {
      gcTime: 0,
      refetchOnMount: "always",
      placeholderData: keepPreviousData,
    },
  })
  const facetsQuery = useListMasterInventoryFacets(toFacetParams(search), {
    query: { placeholderData: keepPreviousData },
  })
  const seriesQuery = useListCatalogSeries()

  const result = query.data?.status === 200 ? query.data.data : undefined
  const items = result?.data ?? []
  const quantities = new Map(items.map((item) => [item.card.id, item.quantity]))
  const facets =
    facetsQuery.data?.status === 200 ? facetsQuery.data.data.data : undefined
  const series =
    seriesQuery.data?.status === 200
      ? (seriesQuery.data.data.data?.series ?? [])
      : []
  const loadError = errorDetail(query.data, "Could not load your inventory.")

  return (
    <section
      className="flex flex-col gap-6"
      aria-label="Master Inventory contents"
    >
      <CatalogFilterPanel
        search={search}
        facets={facets}
        series={series}
        onChange={(patch) => onSearchChange({ ...search, ...patch, page: 1 })}
        onClear={() => onSearchChange({ page: 1 })}
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
            ? "No Cards in your inventory match these filters."
            : "You don't own any Cards yet. Add some to a Collection from the catalog."
        }
        onPageChange={(page) => onSearchChange({ ...search, page })}
        renderControl={(card) => (
          <p className="text-sm font-medium text-muted-foreground">
            ×{quantities.get(card.id) ?? 0}
          </p>
        )}
      />
    </section>
  )
}
