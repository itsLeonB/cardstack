import { createFileRoute, useNavigate } from "@tanstack/react-router"
import { keepPreviousData } from "@tanstack/react-query"
import {
  getListCatalogFacetsQueryOptions,
  getListCatalogSeriesQueryOptions,
  getSearchCatalogCardsQueryOptions,
  useListCatalogFacets,
  useListCatalogSeries,
  useSearchCatalogCards,
} from "@/generated/endpoints/catalog/catalog"
import { CardResults } from "@/components/catalog/card-results"
import { CatalogFilterPanel } from "@/components/catalog/filter-panel"
import { catalogSearchSchema, toFacetParams } from "@/lib/catalog-search"
import type { CatalogSearch } from "@/lib/catalog-search"

export const Route = createFileRoute("/catalog/search")({
  validateSearch: catalogSearchSchema,
  loaderDeps: ({ search }) => search,
  // Only the card search itself is required for this route to render:
  // series names and facets just populate the filters, so a transport-level
  // failure on either degrades the filters instead of blocking the results.
  loader: ({ context: { queryClient }, deps }) =>
    Promise.all([
      queryClient.ensureQueryData(getSearchCatalogCardsQueryOptions(deps)),
      queryClient.ensureQueryData(getListCatalogSeriesQueryOptions()).catch(() => undefined),
      queryClient
        .ensureQueryData(getListCatalogFacetsQueryOptions(toFacetParams(deps)))
        .catch(() => undefined),
    ]),
  component: CatalogSearchPage,
})

function CatalogSearchPage() {
  const search = Route.useSearch()
  const navigate = useNavigate({ from: Route.fullPath })

  const seriesQuery = useListCatalogSeries()
  const facetsQuery = useListCatalogFacets(toFacetParams(search), {
    query: { placeholderData: keepPreviousData },
  })
  const cardsQuery = useSearchCatalogCards(search)

  const series = seriesQuery.data?.status === 200 ? (seriesQuery.data.data.data?.series ?? []) : []
  const facets = facetsQuery.data?.status === 200 ? facetsQuery.data.data.data : undefined

  const result = cardsQuery.data?.status === 200 ? cardsQuery.data.data : undefined
  const cardsErrorMessage =
    cardsQuery.data && cardsQuery.data.status !== 200
      ? (cardsQuery.data.data.detail ?? "Could not search the catalog.")
      : undefined

  function updateSearch(patch: Partial<CatalogSearch>) {
    void navigate({
      search: (prev) => ({ ...prev, ...patch, page: 1 }),
    })
  }

  function handlePageChange(nextPage: number) {
    void navigate({ search: (prev) => ({ ...prev, page: nextPage }) })
  }

  return (
    <main className="mx-auto flex max-w-5xl flex-col gap-8 p-6">
      <header className="flex flex-col gap-2">
        <h1 className="font-heading text-2xl font-medium">Search the catalog</h1>
        <p className="text-sm text-muted-foreground">
          Search by name, or filter by Expansion Set and card number, rarity,
          category, and tag.
        </p>
      </header>

      <CatalogFilterPanel
        search={search}
        facets={facets}
        series={series}
        onChange={updateSearch}
        onClear={() => void navigate({ search: {} })}
      />

      <CardResults
        cards={result?.data ?? []}
        total={result?.meta.total ?? 0}
        page={search.page}
        limit={result?.meta.limit ?? 24}
        isPending={cardsQuery.isPending}
        isError={cardsQuery.isError || Boolean(cardsErrorMessage)}
        errorMessage={cardsErrorMessage}
        emptyMessage="No cards match these filters."
        onPageChange={handlePageChange}
      />
    </main>
  )
}
