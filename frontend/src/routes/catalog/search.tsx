import { useState } from "react"
import { createFileRoute, useNavigate } from "@tanstack/react-router"
import { keepPreviousData } from "@tanstack/react-query"
import { RiSearchLine } from "@remixicon/react"
import {
  getListCatalogFacetsQueryOptions,
  getListCatalogSeriesQueryOptions,
  getSearchCatalogCardsQueryOptions,
  useListCatalogFacets,
  useListCatalogSeries,
  useSearchCatalogCards,
} from "@/generated/endpoints/catalog/catalog"
import { CardResults } from "@/components/catalog/card-results"
import { FacetFilters } from "@/components/catalog/facet-filters"
import type { FacetKey } from "@/components/catalog/facet-filters"
import { Button } from "@/components/ui/button"
import { Field, FieldDescription, FieldGroup, FieldLabel } from "@/components/ui/field"
import { Input } from "@/components/ui/input"
import { catalogSearchSchema } from "@/lib/catalog-search"
import type { CatalogSearch } from "@/lib/catalog-search"

// Facets take the same filters as the card search, minus pagination.
function toFacetParams({ page: _page, ...filters }: CatalogSearch) {
  return filters
}

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

  const [nameInput, setNameInput] = useState(search.name ?? "")
  const [localIdInput, setLocalIdInput] = useState(search.localId ?? "")

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

  function handleTextSearchSubmit(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault()
    updateSearch({
      name: nameInput.trim() || undefined,
      localId: localIdInput.trim() || undefined,
    })
  }

  function handleClearFilters() {
    setNameInput("")
    setLocalIdInput("")
    void navigate({ search: {} })
  }

  const hasActiveFilters = Boolean(
    search.name ||
      search.localId ||
      search.expansionSetId?.length ||
      search.rarityId?.length ||
      search.category?.length ||
      search.tag?.length
  )

  function handleFacetChange(key: FacetKey, values: string[]) {
    updateSearch({ [key]: values.length > 0 ? values : undefined })
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

      <form onSubmit={handleTextSearchSubmit}>
        <FieldGroup>
          <div className="grid gap-4 sm:grid-cols-[1fr_auto]">
            <Field>
              <FieldLabel htmlFor="catalog-search-name">Card name</FieldLabel>
              <Input
                id="catalog-search-name"
                type="search"
                placeholder="e.g. Pikachu"
                value={nameInput}
                onChange={(event) => setNameInput(event.target.value)}
              />
            </Field>
            <Field className="sm:w-40">
              <FieldLabel htmlFor="catalog-search-local-id">Card number</FieldLabel>
              <Input
                id="catalog-search-local-id"
                placeholder="e.g. 048"
                value={localIdInput}
                onChange={(event) => setLocalIdInput(event.target.value)}
              />
              <FieldDescription>Pairs with an Expansion Set below.</FieldDescription>
            </Field>
          </div>

          <FacetFilters
            facets={facets}
            series={series}
            selected={{
              expansionSetId: search.expansionSetId ?? [],
              rarityId: search.rarityId ?? [],
              category: search.category ?? [],
              tag: search.tag ?? [],
            }}
            onChange={handleFacetChange}
          />

          <div className="flex flex-wrap items-center gap-3">
            <Button type="submit">
              <RiSearchLine data-icon="inline-start" />
              Search
            </Button>
            {hasActiveFilters && (
              <Button type="button" variant="ghost" onClick={handleClearFilters}>
                Clear filters
              </Button>
            )}
          </div>
        </FieldGroup>
      </form>

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
