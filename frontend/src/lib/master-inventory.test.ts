import { QueryClient } from "@tanstack/react-query"
import { expect, it } from "vitest"
import { invalidateMasterInventory } from "./master-inventory"
import {
  getListMasterInventoryFacetsQueryKey,
  getListMasterInventoryInfiniteQueryKey,
  getListMasterInventoryQueryKey,
} from "@/generated/endpoints/inventory/inventory"

// Real keys in a real cache: a prefix that doesn't match fails here, which a
// mocked invalidateQueries would never notice.
it("invalidates the infinite list, the plain list (dashboard) and the facets", () => {
  const client = new QueryClient()
  const keys = [
    getListMasterInventoryInfiniteQueryKey({ name: "pika", limit: 100 }),
    getListMasterInventoryQueryKey({ limit: 1 }),
    getListMasterInventoryFacetsQueryKey({ name: "pika" }),
  ]
  for (const key of keys) client.setQueryData(key, {})

  invalidateMasterInventory(client)

  for (const key of keys)
    expect(client.getQueryState(key)?.isInvalidated).toBe(true)
})
