import { useQueryClient } from "@tanstack/react-query"
import {
  getGetAdminCardQueryKey,
  useGetAdminCard,
} from "@/generated/endpoints/admin/admin"
import type {
  AdminCard,
  ExpansionSetSummary,
  SeriesBrowseResult,
} from "@/generated/models"

// Same look as `Input`; a native select keeps the keyboard and screen-reader
// behaviour for free.
export const SELECT_CLASS =
  "h-9 w-full min-w-0 rounded-3xl border border-transparent bg-input/50 px-3 py-1 text-base outline-none focus-visible:border-ring focus-visible:ring-3 focus-visible:ring-ring/30 disabled:opacity-50 md:text-sm"

/** Every Expansion Set of the browse result, those under a Series first. */
export function flattenSets(
  browse: SeriesBrowseResult | null | undefined
): ExpansionSetSummary[] {
  return [
    ...(browse?.series ?? []).flatMap((s) => s.expansionSets ?? []),
    ...(browse?.ungroupedExpansionSets ?? []),
  ]
}

export interface CardLookup {
  /**
   * `idle` before both are entered, `checking` while the API is asked,
   * `found` when the Card exists, `missing` when there is none, `failed`
   * otherwise.
   */
  state: "idle" | "checking" | "found" | "missing" | "failed"
  /** The Card at the entry, kept while it is asked about again. */
  card: AdminCard | null
  /** Asks again for the current entry, after the Card has been written. */
  refresh: () => void
}

/**
 * Looks the Card up by its set and collector number through the admin API,
 * which also finds a Card with no image yet. Nothing is cached once the entry
 * is unused, so entering a number again always asks again.
 */
export function useCardLookup(setId: string, number: string): CardLookup {
  const queryClient = useQueryClient()
  const params = { expansionSetId: setId, localId: number }
  const enabled = setId !== "" && number !== ""
  const query = useGetAdminCard(params, {
    query: { enabled, gcTime: 0, retry: false },
  })

  const refresh = () =>
    void queryClient.invalidateQueries({
      queryKey: getGetAdminCardQueryKey(params),
    })

  if (!enabled) return { state: "idle", card: null, refresh }
  const card = query.data?.status === 200 ? query.data.data.data : null
  if (query.isFetching) return { state: "checking", card, refresh }
  if (card) return { state: "found", card, refresh }
  return {
    state: query.data?.status === 404 ? "missing" : "failed",
    card: null,
    refresh,
  }
}
