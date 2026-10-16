import type { QueryClient } from "@tanstack/react-query"
import { getGetCollectionQueryKey } from "@/generated/endpoints/collections/collections"
import type { BinderPageSide, BinderSlotTarget } from "@/generated/models"
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

// The Layout limits a binder accepts. They mirror the backend (its grid side
// limit and the leaf count on the create route), so change them together.
export const MAX_GRID = 6
export const MAX_LEAVES = 100

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

/**
 * Refreshes a binder after a write that changes its Pages or Layout. The
 * entries follow too: a Leaf change renumbers the Pages, and the entries carry
 * each Card's positions.
 */
export function invalidateBinder(
  queryClient: QueryClient,
  collectionId: string
) {
  invalidatePages(queryClient, collectionId)
  invalidateCollectionCounts(queryClient, collectionId)
  invalidateCollectionEntries(queryClient, collectionId)
}

function invalidatePages(queryClient: QueryClient, collectionId: string) {
  void queryClient.invalidateQueries({
    predicate: binderPagesMatch(collectionId),
  })
}

/**
 * A move exchanges Slot rows and changes no Card's count, so only the Pages and
 * the entries (whose positions it changes) refresh. Master Inventory and the
 * counts keep their data.
 */
export function refreshAfterMove(
  queryClient: QueryClient,
  collectionId: string
) {
  invalidatePages(queryClient, collectionId)
  invalidateCollectionEntries(queryClient, collectionId)
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
 * A write that changes which Cards the Slots hold changes quantities, so the
 * Page, the Collection's entries and Master Inventory all refresh. Slot edits
 * and the Phase 1 placement call this.
 */
export function refreshAfterSlotWrite(
  queryClient: QueryClient,
  collectionId: string
) {
  invalidateBinder(queryClient, collectionId)
  invalidateMasterInventory(queryClient)
}

/** A Slot on a Page, addressed as the API addresses it. */
export type SlotRef = BinderSlotTarget

/** The Page number of a Leaf's front or back. Pages turn from 1, so Leaf 1 is Pages 1 and 2. */
export function pageOf(leaf: number, side: BinderPageSide) {
  return side === "front" ? 2 * leaf - 1 : 2 * leaf
}

/** The key that names one Slot in a React list or a selection. */
export function slotKey({ page, row, column }: SlotRef) {
  return `${page},${row},${column}`
}
