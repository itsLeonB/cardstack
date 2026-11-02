import { keepPreviousData } from "@tanstack/react-query"
import type { QueryClient } from "@tanstack/react-query"
import { z } from "zod"
import {
  getListWishlistFacetsQueryKey,
  getListWishlistInfiniteQueryKey,
  getListWishlistQueryKey,
  listWishlist,
  useListWishlistInfinite,
} from "@/generated/endpoints/wishlist/wishlist"
import { ListWishlistStatus } from "@/generated/models"
import type { WishlistItem } from "@/generated/models"
import { catalogFilterSchema, hasActiveFilters } from "@/lib/catalog-search"
import type { CatalogFilters } from "@/lib/catalog-search"
import { infinitePages, mergePages } from "@/lib/infinite-pages"
import type { CardList, ListPage } from "@/lib/infinite-pages"
import { INVENTORY_PAGE_SIZE } from "@/lib/infinite-inventory"

/**
 * The Wishlist's URL filters: the shared card filters plus the derived status.
 * `status` defaults to `all`, so the API always gets one of its three values.
 */
export const wishlistSearchSchema = catalogFilterSchema.extend({
  status: z.enum(ListWishlistStatus).default(ListWishlistStatus.all),
})

export type WishlistFilters = CatalogFilters & {
  status?: ListWishlistStatus
}

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
  return wishlistStatus(search) !== "all" || hasActiveFilters(search)
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
