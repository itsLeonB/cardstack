import type { QueryClient } from "@tanstack/react-query"
import { getGetCollectionQueryKey } from "@/generated/endpoints/collections/collections"
import {
  invalidateCollectionCounts,
  invalidateCollectionEntries,
} from "@/lib/collections"
import { invalidateMasterInventory } from "@/lib/master-inventory"

export const BINDER_PRESETS = [
  { id: "2x2", label: "2 × 2", rows: 2, columns: 2 },
  { id: "3x3", label: "3 × 3", rows: 3, columns: 3 },
  { id: "3x4", label: "3 × 4", rows: 3, columns: 4 },
  { id: "4x3", label: "4 × 3", rows: 4, columns: 3 },
] as const

/** The preset whose rows and columns match, or "custom". */
export function presetIdFor(rows: number, columns: number): string {
  return (
    BINDER_PRESETS.find((p) => p.rows === rows && p.columns === columns)?.id ??
    "custom"
  )
}

// Page queries live under the Collection's URL, plus `/binder/`.
function binderPagesMatch(collectionId: string) {
  const [collectionUrl] = getGetCollectionQueryKey(collectionId)
  return ({ queryKey }: { queryKey: readonly unknown[] }) =>
    String(queryKey[0]).startsWith(`${collectionUrl}/binder/`)
}

/** Refreshes a binder after a write that changes its Pages or Layout. */
export function invalidateBinder(
  queryClient: QueryClient,
  collectionId: string
) {
  void queryClient.invalidateQueries({
    predicate: binderPagesMatch(collectionId),
  })
  invalidateCollectionCounts(queryClient, collectionId)
}

/**
 * After the Layout is removed, the Pages no longer exist: cancel them so they
 * do not refetch and 404 while the view is still mounted, then refresh the
 * Collection, which turns its page plain.
 */
export function refreshAfterRemoveBinder(
  queryClient: QueryClient,
  collectionId: string
) {
  void queryClient.cancelQueries({ predicate: binderPagesMatch(collectionId) })
  invalidateCollectionCounts(queryClient, collectionId)
}

/**
 * A Slot write changes quantities, so the Page, the Collection's entries and
 * Master Inventory all refresh. A Slot write is the only binder write that does.
 */
export function refreshAfterSlotWrite(
  queryClient: QueryClient,
  collectionId: string
) {
  invalidateBinder(queryClient, collectionId)
  invalidateCollectionEntries(queryClient, collectionId)
  invalidateMasterInventory(queryClient)
}
