import { useState } from "react"
import { createFileRoute, useNavigate } from "@tanstack/react-router"
import type { z } from "zod"
import { RiSearchLine } from "@remixicon/react"
import {
  getListCatalogCategoriesQueryOptions,
  getListCatalogRaritiesQueryOptions,
  getListCatalogSeriesQueryOptions,
  getListCatalogTagsQueryOptions,
  getSearchCatalogCardsQueryOptions,
  useListCatalogCategories,
  useListCatalogRarities,
  useListCatalogSeries,
  useListCatalogTags,
  useSearchCatalogCards,
} from "@/generated/endpoints/catalog/catalog"
import { SearchCatalogCardsQueryParams } from "@/generated/endpoints/catalog/catalog.zod"
import { CardResults } from "@/components/catalog/card-results"
import { Button } from "@/components/ui/button"
import { Field, FieldDescription, FieldGroup, FieldLabel } from "@/components/ui/field"
import { Input } from "@/components/ui/input"
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectLabel,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"

// Derived from orval's generated SearchCatalogCardsQueryParams rather than
// hand-duplicated: same fields/bounds as the backend actually enforces, one
// definition to keep in sync. `limit` is omitted since this search UI has
// no page-size control — CardResults gets `limit` from the query response.
// The `.min(1)` refinements the previous hand-written schema had aren't
// reinstated: every UI control here (Select/Input handlers below) already
// converts an empty value to `undefined` before it reaches `navigate`, so
// an empty-string search param is not something this page's own UI can
// produce; a hand-crafted URL with `?name=` is an edge case the generated
// schema and the backend are both fine accepting as a no-op filter.
const catalogSearchSchema = SearchCatalogCardsQueryParams.omit({ limit: true })

type CatalogSearch = z.infer<typeof catalogSearchSchema>

export const Route = createFileRoute("/catalog/search")({
  validateSearch: catalogSearchSchema,
  loaderDeps: ({ search }) => search,
  loader: ({ context: { queryClient }, deps }) =>
    Promise.all([
      queryClient.ensureQueryData(getSearchCatalogCardsQueryOptions(deps)),
      queryClient.ensureQueryData(getListCatalogSeriesQueryOptions()),
      queryClient.ensureQueryData(getListCatalogRaritiesQueryOptions()),
      queryClient.ensureQueryData(getListCatalogCategoriesQueryOptions()),
      queryClient.ensureQueryData(getListCatalogTagsQueryOptions()),
    ]),
  component: CatalogSearchPage,
})

function CatalogSearchPage() {
  const search = Route.useSearch()
  const navigate = useNavigate({ from: Route.fullPath })

  const [nameInput, setNameInput] = useState(search.name ?? "")
  const [localIdInput, setLocalIdInput] = useState(search.localId ?? "")

  const seriesQuery = useListCatalogSeries()
  const raritiesQuery = useListCatalogRarities()
  const categoriesQuery = useListCatalogCategories()
  const tagsQuery = useListCatalogTags()
  const cardsQuery = useSearchCatalogCards(search)

  const seriesBrowseResult =
    seriesQuery.data?.status === 200 ? seriesQuery.data.data.data : undefined
  const series = seriesBrowseResult?.series ?? []
  const ungroupedExpansionSets = seriesBrowseResult?.ungroupedExpansionSets ?? []
  const rarities = raritiesQuery.data?.status === 200 ? (raritiesQuery.data.data.data ?? []) : []
  const categories =
    categoriesQuery.data?.status === 200 ? (categoriesQuery.data.data.data ?? []) : []
  const tags = tagsQuery.data?.status === 200 ? (tagsQuery.data.data.data ?? []) : []

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
      search.expansionSetId ||
      search.localId ||
      search.rarityId ||
      search.category ||
      search.tag
  )

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

          <div className="grid gap-4 sm:grid-cols-3">
            <Field>
              <FieldLabel htmlFor="catalog-search-set">Expansion Set</FieldLabel>
              <Select
                items={[
                  { label: "Any Expansion Set", value: null },
                  ...series.flatMap((oneSeries) =>
                    (oneSeries.expansionSets ?? []).map((set) => ({
                      label: `${set.name} (${set.code})`,
                      value: set.id,
                    }))
                  ),
                  ...ungroupedExpansionSets.map((set) => ({
                    label: `${set.name} (${set.code}) · Ungrouped`,
                    value: set.id,
                  })),
                ]}
                value={search.expansionSetId ?? null}
                onValueChange={(value) =>
                  updateSearch({ expansionSetId: value ?? undefined })
                }
              >
                <SelectTrigger id="catalog-search-set" className="w-full">
                  <SelectValue placeholder="Any Expansion Set" />
                </SelectTrigger>
                <SelectContent>
                  <SelectGroup>
                    <SelectItem value={null}>Any Expansion Set</SelectItem>
                  </SelectGroup>
                  {series.map((oneSeries) => (
                    <SelectGroup key={oneSeries.id}>
                      <SelectLabel>{oneSeries.name}</SelectLabel>
                      {(oneSeries.expansionSets ?? []).map((set) => (
                        <SelectItem key={set.id} value={set.id}>
                          {set.name} ({set.code})
                        </SelectItem>
                      ))}
                    </SelectGroup>
                  ))}
                  {ungroupedExpansionSets.length > 0 && (
                    <SelectGroup>
                      <SelectLabel>Ungrouped</SelectLabel>
                      {ungroupedExpansionSets.map((set) => (
                        <SelectItem key={set.id} value={set.id}>
                          {set.name} ({set.code})
                        </SelectItem>
                      ))}
                    </SelectGroup>
                  )}
                </SelectContent>
              </Select>
            </Field>

            <Field>
              <FieldLabel htmlFor="catalog-search-rarity">Rarity</FieldLabel>
              <Select
                items={[
                  { label: "Any rarity", value: null },
                  ...rarities.map((rarity) => ({ label: rarity.name, value: rarity.id })),
                ]}
                value={search.rarityId ?? null}
                onValueChange={(value) => updateSearch({ rarityId: value ?? undefined })}
              >
                <SelectTrigger id="catalog-search-rarity" className="w-full">
                  <SelectValue placeholder="Any rarity" />
                </SelectTrigger>
                <SelectContent>
                  <SelectGroup>
                    <SelectItem value={null}>Any rarity</SelectItem>
                    {rarities.map((rarity) => (
                      <SelectItem key={rarity.id} value={rarity.id}>
                        {rarity.name}
                      </SelectItem>
                    ))}
                  </SelectGroup>
                </SelectContent>
              </Select>
            </Field>

            <Field>
              <FieldLabel htmlFor="catalog-search-category">Category</FieldLabel>
              <Select
                items={[
                  { label: "Any category", value: null },
                  ...categories.map((category) => ({ label: category, value: category })),
                ]}
                value={search.category ?? null}
                onValueChange={(value) => updateSearch({ category: value ?? undefined })}
              >
                <SelectTrigger id="catalog-search-category" className="w-full">
                  <SelectValue placeholder="Any category" />
                </SelectTrigger>
                <SelectContent>
                  <SelectGroup>
                    <SelectItem value={null}>Any category</SelectItem>
                    {categories.map((category) => (
                      <SelectItem key={category} value={category}>
                        {category}
                      </SelectItem>
                    ))}
                  </SelectGroup>
                </SelectContent>
              </Select>
            </Field>
          </div>

          <div className="grid gap-4 sm:grid-cols-3">
            <Field>
              <FieldLabel htmlFor="catalog-search-tag">Tag</FieldLabel>
              <Select
                items={[
                  { label: "Any tag", value: null },
                  ...tags.map((tag) => ({ label: tag, value: tag })),
                ]}
                value={search.tag ?? null}
                onValueChange={(value) => updateSearch({ tag: value ?? undefined })}
              >
                <SelectTrigger id="catalog-search-tag" className="w-full">
                  <SelectValue placeholder="Any tag" />
                </SelectTrigger>
                <SelectContent>
                  <SelectGroup>
                    <SelectItem value={null}>Any tag</SelectItem>
                    {tags.map((tag) => (
                      <SelectItem key={tag} value={tag}>
                        {tag}
                      </SelectItem>
                    ))}
                  </SelectGroup>
                </SelectContent>
              </Select>
            </Field>
          </div>

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
