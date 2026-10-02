import { useCallback } from "react"
import { createFileRoute, useLocation, useNavigate } from "@tanstack/react-router"
import { keepPreviousData } from "@tanstack/react-query"
import {
  getListCatalogFacetsQueryOptions,
  getListCatalogSeriesQueryOptions,
  getSearchCatalogCardsQueryOptions,
  useListCatalogFacets,
  useListCatalogSeries,
  useSearchCatalogCards,
} from "@/generated/endpoints/catalog/catalog"
import { z } from "zod"
import { Breadcrumbs } from "@/components/layout/breadcrumbs"
import { PageContainer } from "@/components/layout/page-container"
import { PageHeader } from "@/components/layout/page-header"
import { CardResults } from "@/components/catalog/card-results"
import { CollectionCardResults } from "@/components/catalog/collection-card-results"
import { CollectionPicker, useCatalogCollection } from "@/components/catalog/collection-picker"
import { CatalogFilterPanel } from "@/components/catalog/filter-panel"
import { catalogSearchSchema, toFacetParams } from "@/lib/catalog-search"
import type { CatalogSearch } from "@/lib/catalog-search"

// `collectionId` is page state only; it is split off before any catalog query.
const searchSchema = catalogSearchSchema.extend({ collectionId: z.string().optional() })

export const Route = createFileRoute("/catalog/search")({
  validateSearch: searchSchema,
  loaderDeps: ({ search: { collectionId: _collectionId, ...filters } }) => filters,
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
  const { collectionId, ...search } = Route.useSearch()
  const navigate = useNavigate({ from: Route.fullPath })
  const selectCollection = useCallback(
    (id: string | undefined) =>
      void navigate({ search: (prev) => ({ ...prev, collectionId: id }), replace: true }),
    [navigate]
  )
  const href = useLocation({ select: (location) => location.href })
  const collection = useCatalogCollection(collectionId, selectCollection)

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

  const resultsProps = {
    cards: result?.data ?? [],
    total: result?.meta.total ?? 0,
    page: search.page,
    limit: result?.meta.limit ?? 24,
    isPending: cardsQuery.isPending,
    isError: cardsQuery.isError || Boolean(cardsErrorMessage),
    errorMessage: cardsErrorMessage,
    emptyMessage: "No cards match these filters.",
    onPageChange: handlePageChange,
  }

  return (
    <PageContainer variant="wide">
      <Breadcrumbs
        crumbs={[{ label: "Catalog", link: { to: "/catalog" } }, { label: "Search" }]}
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
        <CollectionCardResults key={collection.selected} collectionId={collection.selected} {...resultsProps} />
      ) : (
        <CardResults {...resultsProps} />
      )}
    </PageContainer>
  )
}
