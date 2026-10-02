import type { QueryClient } from "@tanstack/react-query"
import {
  getListMasterInventoryFacetsQueryKey,
  getListMasterInventoryInfiniteQueryKey,
  getListMasterInventoryQueryKey,
} from "@/generated/endpoints/inventory/inventory"

/**
 * Refreshes the Master Inventory list and its facets. Three key roots, so all
 * are needed: the infinite list (its keys start with `'infinite'`), the plain
 * list (the dashboard's count) and the facets.
 */
export function invalidateMasterInventory(queryClient: QueryClient) {
  for (const queryKey of [
    getListMasterInventoryInfiniteQueryKey(),
    getListMasterInventoryQueryKey(),
    getListMasterInventoryFacetsQueryKey(),
  ]) {
    void queryClient.invalidateQueries({ queryKey })
  }
}
