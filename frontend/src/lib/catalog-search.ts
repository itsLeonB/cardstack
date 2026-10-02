import { z } from "zod"
import { SearchCatalogCardsQueryParams } from "@/generated/endpoints/catalog/catalog.zod"

// TanStack parses `?tag=2` to the number 2 and a legacy single-value URL
// `?rarityId=x` to a bare string; both normalize to string[] so old links keep working.
const multiValue = z.preprocess(
  (value) => (value == null ? undefined : [value].flat().map(String)),
  z.array(z.string()).optional()
)

// Derived from the generated params; `limit` is omitted (no page-size control).
export const catalogSearchSchema = SearchCatalogCardsQueryParams.omit({
  limit: true,
}).extend({
  expansionSetId: multiValue,
  rarityId: multiValue,
  category: multiValue,
  tag: multiValue,
})

export type CatalogSearch = z.infer<typeof catalogSearchSchema>

// The infinite catalog search carries filters only: a stale `?page=` in an old
// link is stripped by the object schema, so it opens the first page.
export const catalogFilterSchema = catalogSearchSchema.omit({ page: true })

export type CatalogFilters = z.infer<typeof catalogFilterSchema>

// Facets take the same filters as the card search, minus pagination.
export function toFacetParams({ page: _page, ...filters }: CatalogSearch) {
  return filters
}

export function hasActiveFilters(search: CatalogFilters) {
  return Boolean(
    search.name ||
    search.localId ||
    search.expansionSetId?.length ||
    search.rarityId?.length ||
    search.category?.length ||
    search.tag?.length
  )
}
