import { keepPreviousData } from "@tanstack/react-query"
import { useListCatalogSeries } from "@/generated/endpoints/catalog/catalog"
import { useListMasterInventoryFacets } from "@/generated/endpoints/inventory/inventory"
import { InfiniteCardResults } from "@/components/catalog/infinite-card-results"
import { CatalogFilterPanel } from "@/components/catalog/filter-panel"
import type { CatalogFilters } from "@/lib/catalog-search"
import { hasActiveFilters } from "@/lib/catalog-search"
import { useInfiniteCardResultsProps } from "@/lib/infinite-catalog-cards"
import { useInfiniteMasterInventory } from "@/lib/infinite-inventory"

interface MasterInventoryProps {
  search: CatalogFilters
  /** Receives the next filters; the route writes them to the URL. */
  onSearchChange: (next: CatalogFilters) => void
}

/** The user's Cards with the total quantity owned across all their Collections (read-only). */
export function MasterInventory({
  search,
  onSearchChange,
}: MasterInventoryProps) {
  const query = useInfiniteMasterInventory(search)
  const facetsQuery = useListMasterInventoryFacets(search, {
    query: { placeholderData: keepPreviousData },
  })
  const seriesQuery = useListCatalogSeries()

  const results = useInfiniteCardResultsProps(query)
  const quantities = query.data?.quantities
  const facets =
    facetsQuery.data?.status === 200 ? facetsQuery.data.data.data : undefined
  const series =
    seriesQuery.data?.status === 200
      ? (seriesQuery.data.data.data?.series ?? [])
      : []

  return (
    <section
      className="flex flex-col gap-6"
      aria-label="Master Inventory contents"
    >
      <CatalogFilterPanel
        search={search}
        facets={facets}
        series={series}
        onChange={(patch) => onSearchChange({ ...search, ...patch })}
        onClear={() => onSearchChange({})}
      />
      <InfiniteCardResults
        {...results}
        emptyMessage={
          hasActiveFilters(search)
            ? "No Cards in your inventory match these filters."
            : "You don't own any Cards yet. Add some to a Collection from the catalog."
        }
        renderControl={(card) => (
          <p className="text-sm font-medium text-muted-foreground">
            ×{quantities?.[card.id] ?? 0}
          </p>
        )}
      />
    </section>
  )
}
