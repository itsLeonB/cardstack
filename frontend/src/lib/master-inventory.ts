import type { QueryClient } from "@tanstack/react-query"
import {
  getListMasterInventoryFacetsQueryKey,
  getListMasterInventoryQueryKey,
} from "@/generated/endpoints/inventory/inventory"

/** Refreshes the Master Inventory list and its facets (different key roots, so both are needed). */
export function invalidateMasterInventory(queryClient: QueryClient) {
  void queryClient.invalidateQueries({ queryKey: getListMasterInventoryQueryKey() })
  void queryClient.invalidateQueries({ queryKey: getListMasterInventoryFacetsQueryKey() })
}
