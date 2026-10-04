import { useCallback } from "react"
import type { QueryClient } from "@tanstack/react-query"
import {
  getSearchCatalogCardsInfiniteQueryOptions,
  searchCatalogCards,
  useSearchCatalogCardsInfinite,
} from "@/generated/endpoints/catalog/catalog"
import type { CardSummary, SearchCatalogCardsParams } from "@/generated/models"
import {
  LoginRequiredError,
  infinitePages,
  mergePages,
} from "@/lib/infinite-pages"
import type { CardList } from "@/lib/infinite-pages"

export const CATALOG_PAGE_SIZE = 60

export type CatalogFilterParams = Omit<
  SearchCatalogCardsParams,
  "page" | "limit"
>

function infiniteArgs(filters: CatalogFilterParams) {
  const params = { ...filters, limit: CATALOG_PAGE_SIZE }
  const query = {
    ...infinitePages(
      (page, signal) => searchCatalogCards({ ...params, page }, { signal }),
      "Could not search the catalog."
    ),
    // Catalog data only changes on ingestion; every loaded page refetches
    // sequentially when stale, so don't refetch on each tab focus.
    staleTime: 60_000,
  }
  return { params, query }
}

/** Loader options: the key is the filters alone, so a filter change is a new list. */
export function catalogInfiniteQueryOptions(filters: CatalogFilterParams) {
  const { params, query } = infiniteArgs(filters)
  return getSearchCatalogCardsInfiniteQueryOptions(params, { query })
}

export function useInfiniteCatalogCards(filters: CatalogFilterParams) {
  const { params, query } = infiniteArgs(filters)
  return useSearchCatalogCardsInfinite(params, {
    query: {
      ...query,
      select: (data): CardList => {
        const { rows, total } = mergePages(
          data.pages,
          (card: CardSummary) => card.id
        )
        return { cards: rows, total }
      },
    },
  })
}

/**
 * Route-loader prefetch of page 1; later pages load on scroll. A failed request
 * resolves to undefined so the page shows its inline error instead of crashing
 * the route.
 */
export function prefetchInfiniteCatalogCards(
  queryClient: QueryClient,
  filters: CatalogFilterParams
) {
  return queryClient
    .infiniteQuery({
      ...catalogInfiniteQueryOptions(filters),
      staleTime: "static",
    })
    .catch(() => undefined)
}

/** Maps an infinite catalog query onto the props `InfiniteCardResults` takes, bar `emptyMessage`. */
export function useInfiniteCardResultsProps(query: {
  data: CardList | undefined
  fetchNextPage: (options: { cancelRefetch: boolean }) => void
  hasNextPage: boolean
  isError: boolean
  isFetching: boolean
  isPending: boolean
  error: unknown
}) {
  const { fetchNextPage } = query
  // cancelRefetch: false, or a call during a background refetch cancels it.
  const onLoadMore = useCallback(
    () => void fetchNextPage({ cancelRefetch: false }),
    [fetchNextPage]
  )
  return {
    cards: query.data?.cards ?? [],
    total: query.data?.total ?? 0,
    isPending: query.isPending,
    isError: query.isError,
    loginRequired: query.error instanceof LoginRequiredError,
    errorMessage:
      query.error instanceof Error ? query.error.message : undefined,
    hasNextPage: query.hasNextPage,
    isFetching: query.isFetching,
    onLoadMore,
  }
}
