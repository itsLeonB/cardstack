import type { QueryClient } from "@tanstack/react-query"
import { getGetCollectionQueryKey } from "@/generated/endpoints/collections/collections"

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

/**
 * Refreshes a binder after a write: its Pages (keys under the Collection's
 * `/binder/` path) and the Collection itself, which carries the Layout.
 */
export function invalidateBinder(
  queryClient: QueryClient,
  collectionId: string
) {
  const [collectionUrl] = getGetCollectionQueryKey(collectionId)
  void queryClient.invalidateQueries({
    predicate: ({ queryKey }) =>
      String(queryKey[0]).startsWith(`${collectionUrl}/binder/`),
  })
  void queryClient.invalidateQueries({
    queryKey: getGetCollectionQueryKey(collectionId),
  })
}
