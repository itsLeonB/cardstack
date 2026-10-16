import { QueryClient } from "@tanstack/react-query"
import { expect, it } from "vitest"
import { refreshAfterMove, refreshAfterSlotWrite } from "./binder"
import { getGetBinderPageQueryKey } from "@/generated/endpoints/binder/binder"
import { getGetCollectionQueryKey } from "@/generated/endpoints/collections/collections"
import {
  getListCollectionEntriesInfiniteQueryKey,
  getListMasterInventoryInfiniteQueryKey,
  getListMasterInventoryQueryKey,
} from "@/generated/endpoints/inventory/inventory"

// Real keys in a real cache: a prefix that misses fails here, which a mocked
// invalidateQueries would never notice.
it("refreshes the Pages, the entries and Master Inventory after a Slot write", () => {
  const client = new QueryClient()
  const page = getGetBinderPageQueryKey("col-1", 1)
  const entries = getListCollectionEntriesInfiniteQueryKey("col-1", {
    limit: 100,
  })
  const inventory = getListMasterInventoryInfiniteQueryKey({ limit: 100 })
  const plainInventory = getListMasterInventoryQueryKey({ limit: 1 })
  const collection = getGetCollectionQueryKey("col-1")
  const other = getGetBinderPageQueryKey("col-2", 1)
  const keys = [page, entries, inventory, plainInventory, collection, other]
  for (const key of keys) client.setQueryData(key, {})

  refreshAfterSlotWrite(client, "col-1")

  for (const key of [page, entries, inventory, plainInventory, collection])
    expect(client.getQueryState(key)?.isInvalidated).toBe(true)
  expect(client.getQueryState(other)?.isInvalidated).toBe(false)
})

it("after a move refreshes the Pages and the entries, but not Master Inventory or the Collection", () => {
  const client = new QueryClient()
  const page = getGetBinderPageQueryKey("col-1", 1)
  const entries = getListCollectionEntriesInfiniteQueryKey("col-1", {
    limit: 100,
  })
  const inventory = getListMasterInventoryInfiniteQueryKey({ limit: 100 })
  const collection = getGetCollectionQueryKey("col-1")
  for (const key of [page, entries, inventory, collection])
    client.setQueryData(key, {})

  refreshAfterMove(client, "col-1")

  for (const key of [page, entries])
    expect(client.getQueryState(key)?.isInvalidated).toBe(true)
  for (const key of [inventory, collection])
    expect(client.getQueryState(key)?.isInvalidated).toBe(false)
})
