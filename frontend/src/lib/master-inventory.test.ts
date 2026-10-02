import { expect, it, vi } from "vitest"
import { invalidateMasterInventory } from "./master-inventory"
import {
  getListMasterInventoryFacetsQueryKey,
  getListMasterInventoryQueryKey,
} from "@/generated/endpoints/inventory/inventory"

it("invalidates both the Master Inventory list and its facets", () => {
  const invalidateQueries = vi.fn()
  // SAFETY: only invalidateQueries is used.
  invalidateMasterInventory({ invalidateQueries } as any)
  const keys = invalidateQueries.mock.calls.map(([arg]) => arg.queryKey)
  expect(keys).toEqual([
    getListMasterInventoryQueryKey(),
    getListMasterInventoryFacetsQueryKey(),
  ])
})
