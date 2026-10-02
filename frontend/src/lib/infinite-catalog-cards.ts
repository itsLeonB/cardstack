import {
  getSearchCatalogCardsInfiniteQueryOptions,
  searchCatalogCards,
  useSearchCatalogCardsInfinite,
} from "@/generated/endpoints/catalog/catalog"
import type { CardSummary, SearchCatalogCardsParams } from "@/generated/models"

export const CATALOG_PAGE_SIZE = 60

export type CatalogFilterParams = Omit<
  SearchCatalogCardsParams,
  "page" | "limit"
>
type Page = Awaited<ReturnType<typeof searchCatalogCards>>

/**
 * Next 1-indexed page, or undefined at the end. A short page also ends the
 * list: `total` and the rows come from separate statements, so it can be stale.
 */
export function nextCatalogPageParam(last: Page) {
  if (last.status !== 200) return undefined
  const { page, limit, total } = last.data.meta
  const loaded = last.data.data?.length ?? 0
  return loaded === limit && page * limit < total ? page + 1 : undefined
}

/**
 * Flatten pages into one list, dropping repeats by card id: a row inserted
 * ahead of the window between two fetches repeats the boundary card.
 */
export function mergeCatalogPages(pages: Page[]) {
  const byId = new Map<string, CardSummary>()
  let total = 0
  for (const page of pages) {
    if (page.status !== 200) continue
    total = page.data.meta.total
    for (const card of page.data.data ?? []) {
      if (!byId.has(card.id)) byId.set(card.id, card)
    }
  }
  return { cards: [...byId.values()], total }
}

function infiniteOptions(filters: CatalogFilterParams) {
  const params = { ...filters, limit: CATALOG_PAGE_SIZE }
  return {
    initialPageParam: 1,
    getNextPageParam: nextCatalogPageParam,
    // The generated fetcher resolves for every HTTP status; throw so an error
    // body is never stored as a page (getNextPageParam would read `meta` off it).
    queryFn: async ({
      pageParam,
      signal,
    }: {
      pageParam: number | undefined
      signal: AbortSignal
    }) => {
      const response = await searchCatalogCards(
        { ...params, page: pageParam },
        { signal }
      )
      if (response.status !== 200) {
        throw new Error(response.data.detail ?? "Could not search the catalog.")
      }
      return response
    },
    // Catalog data only changes on ingestion; every loaded page refetches
    // sequentially when stale, so don't refetch on each tab focus.
    staleTime: 60_000,
  }
}

/** Loader options: the key is the filters alone, so a filter change is a new list. */
export function catalogInfiniteQueryOptions(filters: CatalogFilterParams) {
  return getSearchCatalogCardsInfiniteQueryOptions(
    { ...filters, limit: CATALOG_PAGE_SIZE },
    { query: infiniteOptions(filters) }
  )
}

export function useInfiniteCatalogCards(filters: CatalogFilterParams) {
  return useSearchCatalogCardsInfinite(
    { ...filters, limit: CATALOG_PAGE_SIZE },
    {
      query: {
        ...infiniteOptions(filters),
        select: (data) => mergeCatalogPages(data.pages),
      },
    }
  )
}
