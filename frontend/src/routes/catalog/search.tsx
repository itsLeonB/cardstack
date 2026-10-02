import { useCallback } from "react"
import {
  createFileRoute,
  useLocation,
  useNavigate,
} from "@tanstack/react-router"
import { pageHead } from "@/lib/site"
import { keepPreviousData } from "@tanstack/react-query"
import {
  getListCatalogFacetsQueryOptions,
  getListCatalogSeriesQueryOptions,
  useListCatalogFacets,
  useListCatalogSeries,
} from "@/generated/endpoints/catalog/catalog"
import { z } from "zod"
import { Breadcrumbs } from "@/components/layout/breadcrumbs"
import { PageContainer } from "@/components/layout/page-container"
import { PageHeader } from "@/components/layout/page-header"
import { InfiniteCardResults } from "@/components/catalog/infinite-card-results"
import { CollectionCardResults } from "@/components/catalog/collection-card-results"
import {
  CollectionPicker,
  useCatalogCollection,
} from "@/components/catalog/collection-picker"
import { CatalogFilterPanel } from "@/components/catalog/filter-panel"
import { catalogFilterSchema } from "@/lib/catalog-search"
import type { CatalogFilters } from "@/lib/catalog-search"
import {
  catalogInfiniteQueryOptions,
  useInfiniteCatalogCards,
} from "@/lib/infinite-catalog-cards"

// `collectionId` is page state only; it is split off before any catalog query.
const searchSchema = catalogFilterSchema.extend({
  collectionId: z.string().optional(),
})

export const Route = createFileRoute("/catalog/search")({
  head: () => pageHead("Search the catalog"),
  validateSearch: searchSchema,
  loaderDeps: ({ search: { collectionId: _collectionId, ...filters } }) =>
    filters,
  // Page 1 of the card search is prefetched; later pages load on scroll.
  // Series names and facets just populate the filters, so a transport-level
  // failure on any of the three degrades to the page's inline error or an
  // empty filter instead of crashing the route.
  loader: ({ context: { queryClient }, deps }) =>
    Promise.all([
      queryClient
        .infiniteQuery({
          ...catalogInfiniteQueryOptions(deps),
          staleTime: "static",
        })
        .catch(() => undefined),
      queryClient
        .ensureQueryData(getListCatalogSeriesQueryOptions())
        .catch(() => undefined),
      queryClient
        .ensureQueryData(getListCatalogFacetsQueryOptions(deps))
        .catch(() => undefined),
    ]),
  component: CatalogSearchPage,
})

function CatalogSearchPage() {
  const { collectionId, ...search } = Route.useSearch()
  const navigate = useNavigate({ from: Route.fullPath })
  const selectCollection = useCallback(
    (id: string | undefined) =>
      void navigate({
        search: (prev) => ({ ...prev, collectionId: id }),
        replace: true,
      }),
    [navigate]
  )
  const href = useLocation({ select: (location) => location.href })
  const collection = useCatalogCollection(collectionId, selectCollection)

  const seriesQuery = useListCatalogSeries()
  const facetsQuery = useListCatalogFacets(search, {
    query: { placeholderData: keepPreviousData },
  })
  const cardsQuery = useInfiniteCatalogCards(search)
  const { fetchNextPage } = cardsQuery
  // cancelRefetch: false, or a call during a background refetch cancels it.
  const loadMore = useCallback(
    () => void fetchNextPage({ cancelRefetch: false }),
    [fetchNextPage]
  )

  const series =
    seriesQuery.data?.status === 200
      ? (seriesQuery.data.data.data?.series ?? [])
      : []
  const facets =
    facetsQuery.data?.status === 200 ? facetsQuery.data.data.data : undefined

  function updateSearch(patch: Partial<CatalogFilters>) {
    void navigate({ search: (prev) => ({ ...prev, ...patch }) })
  }

  const resultsProps = {
    cards: cardsQuery.data?.cards ?? [],
    total: cardsQuery.data?.total ?? 0,
    isPending: cardsQuery.isPending,
    isError: cardsQuery.isError,
    errorMessage:
      cardsQuery.error instanceof Error ? cardsQuery.error.message : undefined,
    emptyMessage: "No cards match these filters.",
    hasNextPage: cardsQuery.hasNextPage,
    isFetching: cardsQuery.isFetching,
    onLoadMore: loadMore,
  }

  return (
    <PageContainer variant="wide">
      <Breadcrumbs
        crumbs={[
          { label: "Catalog", link: { to: "/catalog" } },
          { label: "Search" },
        ]}
      />
      <PageHeader
        title="Search the catalog"
        description="Search by name, or filter by Expansion Set and card number, rarity, category, and tag."
      />

      <CatalogFilterPanel
        search={search}
        facets={facets}
        series={series}
        onChange={updateSearch}
        onClear={() => void navigate({ search: { collectionId } })}
      />

      <CollectionPicker
        isAuthenticated={collection.isAuthenticated}
        isLoading={collection.isLoading}
        loginRedirect={href}
        collections={collection.collections}
        value={collection.selected}
        onChange={selectCollection}
      />

      {collection.selected ? (
        <CollectionCardResults
          key={collection.selected}
          collectionId={collection.selected}
          {...resultsProps}
        />
      ) : (
        <InfiniteCardResults {...resultsProps} />
      )}
    </PageContainer>
  )
}
