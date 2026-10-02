import { keepPreviousData } from "@tanstack/react-query"
import {
  listCollectionEntries,
  listMasterInventory,
  useListCollectionEntriesInfinite,
  useListMasterInventoryInfinite,
} from "@/generated/endpoints/inventory/inventory"
import type { InventoryItem } from "@/generated/models"
import type { CatalogFilters } from "@/lib/catalog-search"
import type { CardList } from "@/lib/infinite-catalog-cards"
import { infinitePages, mergePages } from "@/lib/infinite-pages"
import type { ListPage } from "@/lib/infinite-pages"

/** The endpoints allow 100 per page at most. */
export const INVENTORY_PAGE_SIZE = 100

/**
 * A card list plus the server's quantity for each loaded card. A plain record,
 * not a Map: `select` results keep their identity through structural sharing
 * only when they are plain objects, and effects key on that identity.
 */
export interface InventoryList extends CardList {
  quantities: Record<string, number>
}

export function mergeInventoryPages(
  pages: ListPage<InventoryItem>[]
): InventoryList {
  const { rows, total } = mergePages(pages, (item) => item.card.id)
  return {
    cards: rows.map((item) => item.card),
    total,
    quantities: Object.fromEntries(
      rows.map((item) => [item.card.id, item.quantity])
    ),
  }
}

/**
 * A Collection's entries, page after page. `gcTime: 0` + `refetchOnMount`: a
 * card taken to 0 stays on screen only until the user comes back, so no cached
 * list may be reused on return (back navigation therefore lands at the top).
 * Refocusing the tab counts as coming back, but not while edits are unsent or
 * saving (`canRefetchOnFocus`).
 */
export function useInfiniteCollectionEntries(
  collectionId: string,
  filters: CatalogFilters,
  canRefetchOnFocus: () => boolean
) {
  const params = { ...filters, limit: INVENTORY_PAGE_SIZE }
  return useListCollectionEntriesInfinite(collectionId, params, {
    query: {
      ...infinitePages(
        (page, signal) =>
          listCollectionEntries(collectionId, { ...params, page }, { signal }),
        "Could not load this Collection's Cards."
      ),
      gcTime: 0,
      refetchOnMount: "always",
      refetchOnWindowFocus: canRefetchOnFocus,
      placeholderData: keepPreviousData,
      select: (data) => mergeInventoryPages(data.pages),
    },
  })
}

/**
 * The Master Inventory, page after page. It is read-only, so no zeroed tile
 * can reappear and the default `gcTime` stays: back navigation restores the
 * scroll position from the cached pages while `refetchOnMount` refreshes them.
 * Edits elsewhere invalidate it (`invalidateMasterInventory`), so a longer
 * `staleTime` spares the N sequential requests a tab refocus would cost.
 */
export function useInfiniteMasterInventory(filters: CatalogFilters) {
  const params = { ...filters, limit: INVENTORY_PAGE_SIZE }
  return useListMasterInventoryInfinite(params, {
    query: {
      ...infinitePages(
        (page, signal) => listMasterInventory({ ...params, page }, { signal }),
        "Could not load your inventory."
      ),
      refetchOnMount: "always",
      staleTime: 60_000,
      placeholderData: keepPreviousData,
      select: (data) => mergeInventoryPages(data.pages),
    },
  })
}
