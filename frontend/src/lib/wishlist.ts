import { keepPreviousData, useQueries } from "@tanstack/react-query"
import type { QueryClient } from "@tanstack/react-query"
import { z } from "zod"
import {
  listSharedWishlistEntries,
  useListSharedWishlistEntriesInfinite,
  getListSharedWishlistEntriesInfiniteQueryKey,
  getListSharedWishlistEntriesQueryKey,
  getListSharedWishlistFacetsQueryKey,
} from "@/generated/endpoints/shared/shared"
import {
  getListWishlistFacetsQueryKey,
  getListWishlistInfiniteQueryKey,
  getListWishlistQueryKey,
  getListWishlistQueryOptions,
  listWishlist,
  useListWishlistInfinite,
} from "@/generated/endpoints/wishlist/wishlist"
import { ListWishlistStatus } from "@/generated/models"
import type {
  ListSharedWishlistEntriesViewerFilter,
  WishlistItem,
} from "@/generated/models"
import { catalogFilterSchema, hasActiveFilters } from "@/lib/catalog-search"
import type { CatalogFilters } from "@/lib/catalog-search"
import { CATALOG_PAGE_SIZE } from "@/lib/infinite-catalog-cards"
import { infinitePages, mergePages } from "@/lib/infinite-pages"
import type { CardList, ListPage } from "@/lib/infinite-pages"
import { INVENTORY_PAGE_SIZE } from "@/lib/infinite-inventory"

/**
 * The Wishlist's URL filters: the shared card filters plus the derived status.
 * `status` defaults to `all`, so the API always gets one of its three values.
 */
export const wishlistSearchSchema = catalogFilterSchema.extend({
  status: z.enum(ListWishlistStatus).default(ListWishlistStatus.all),
  // The signed-in viewer's own "have"/"dont-have" filter; absent for a Guest
  // and for the owner, who get no viewer overlay.
  viewerFilter: z.enum(["have", "dont-have"]).optional(),
})

export type WishlistFilters = CatalogFilters & {
  status?: ListWishlistStatus
  /** The signed-in viewer's own "have"/"dont-have" filter on their owned quantity. */
  viewerFilter?: ListSharedWishlistEntriesViewerFilter
}

/** The All / Still needed / Fulfilled filter, shared by the owner's list and the Shared page. */
export const WISHLIST_STATUS_OPTIONS = [
  { value: "all", label: "All" },
  { value: "needed", label: "Still needed" },
  { value: "fulfilled", label: "Fulfilled" },
]

/** The status filter's value, with `all` for the shared filter panel's clear. */
export function wishlistStatus(search: WishlistFilters): ListWishlistStatus {
  return search.status ?? "all"
}

/** Narrows a value to one of the three statuses the API accepts. */
export function isWishlistStatus(value: string): value is ListWishlistStatus {
  return Object.values(ListWishlistStatus).some((status) => status === value)
}

/** Whether anything narrows the list, the status included. */
export function hasActiveWishlistFilters(search: WishlistFilters) {
  return (
    wishlistStatus(search) !== "all" ||
    hasActiveFilters(search) ||
    search.viewerFilter !== undefined
  )
}

/**
 * A Wishlist page's items plus the server's row for each loaded Card. A plain
 * record, not a Map: `select` results keep their identity only as plain values.
 */
export interface WishlistList extends CardList {
  items: Record<string, WishlistItem>
}

export function mergeWishlistPages(
  pages: ListPage<WishlistItem>[]
): WishlistList {
  const { rows, total } = mergePages(pages, (item) => item.card.id)
  return {
    cards: rows.map((item) => item.card),
    total,
    items: Object.fromEntries(rows.map((item) => [item.card.id, item])),
  }
}

/**
 * The Wishlist, page after page. Edits invalidate it (`invalidateWishlist`)
 * rather than patching, because the status on each item is derived on read: a
 * patched wanted quantity would have to re-derive Fulfilled in the browser.
 */
export function useInfiniteWishlist(filters: WishlistFilters) {
  const params = { ...filters, limit: INVENTORY_PAGE_SIZE }
  return useListWishlistInfinite(params, {
    query: {
      ...infinitePages(
        (page, signal) => listWishlist({ ...params, page }, { signal }),
        "Could not load your Wishlist."
      ),
      refetchOnMount: "always",
      placeholderData: keepPreviousData,
      select: (data) => mergeWishlistPages(data.pages),
    },
  })
}

/**
 * Refreshes the Wishlist list and its facets. Two key roots, so both are
 * needed: the infinite list (keys start with `'infinite'`) and the plain list.
 */
export function invalidateWishlist(queryClient: QueryClient) {
  for (const queryKey of [
    getListWishlistInfiniteQueryKey(),
    getListWishlistQueryKey(),
    getListWishlistFacetsQueryKey(),
  ]) {
    void queryClient.invalidateQueries({ queryKey })
  }
}

/**
 * A Shared Wishlist's items, page after page. Read-only, so it is never
 * patched: the status on each item is derived on read.
 */
export function useInfiniteSharedWishlist(
  profileSlug: string,
  filters: WishlistFilters
) {
  const params = { ...filters, limit: INVENTORY_PAGE_SIZE }
  return useListSharedWishlistEntriesInfinite(profileSlug, params, {
    query: {
      ...infinitePages(
        (page, signal) =>
          listSharedWishlistEntries(
            profileSlug,
            { ...params, page },
            { signal }
          ),
        "Could not load this Wishlist's Cards."
      ),
      placeholderData: keepPreviousData,
      select: (data) => mergeWishlistPages(data.pages),
    },
  })
}

/**
 * Invalidates the shared Wishlist's queries (entries, facets) and its header.
 * Called when sharing changes so an already-open shared page refreshes.
 */
export function invalidateSharedWishlist(
  queryClient: QueryClient,
  profileSlug: string
) {
  for (const queryKey of [
    getListSharedWishlistEntriesInfiniteQueryKey(profileSlug),
    getListSharedWishlistEntriesQueryKey(profileSlug),
    getListSharedWishlistFacetsQueryKey(profileSlug),
  ]) {
    void queryClient.invalidateQueries({ queryKey })
  }
}

// Ids per lookup: the catalog's page size, so a slice is one loaded page: an
// appended page adds a lookup instead of changing the key (and dropping the
// data) of the earlier ones, and the request stays under the endpoint's cap of
// 100 `cardId` values.
function sliceCardIds(cardIds: string[]) {
  const slices: string[][] = []
  for (let start = 0; start < cardIds.length; start += CATALOG_PAGE_SIZE) {
    slices.push(cardIds.slice(start, start + CATALOG_PAGE_SIZE))
  }
  return slices
}

/**
 * The wanted quantity the Wishlist already holds for each of these Cards, so a
 * catalog result can show its state instead of offering a blind add.
 * `loaded` tells "not wanted" apart from "no answer yet".
 */
export function useWishlistWanted(cardIds: string[], enabled: boolean) {
  const slices = sliceCardIds(cardIds)
  return useQueries({
    queries: slices.map((ids) =>
      getListWishlistQueryOptions(
        { cardId: ids, limit: ids.length },
        // gcTime 0 + refetchOnMount: no cached answer can show a wanted quantity from before an edit.
        { query: { enabled, gcTime: 0, refetchOnMount: "always" } }
      )
    ),
    combine: (queries) => {
      const wanted: Record<string, number> = {}
      const loaded: Record<string, true> = {}
      queries.forEach((query, index) => {
        if (query.data?.status !== 200) return
        for (const item of query.data.data.data ?? []) {
          wanted[item.card.id] = item.wantedQuantity
        }
        for (const cardId of slices[index] ?? []) loaded[cardId] = true
      })
      return {
        wanted,
        loaded,
        failed: queries.some(
          (query) => query.isError || (query.data && query.data.status !== 200)
        ),
      }
    },
  })
}
