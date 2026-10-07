import { useRef } from "react"
import { useQueries } from "@tanstack/react-query"
import { getSearchCatalogCardsQueryOptions } from "@/generated/endpoints/catalog/catalog"
import type { CardSummary } from "@/generated/models"
import type { DraftRow } from "@/lib/draft-addition"

// The catalog's `cardId` filter takes at most 100 ids per request.
export const MAX_CARD_IDS_PER_REQUEST = 100

export function chunked(ids: string[]) {
  const chunks: string[][] = []
  for (let i = 0; i < ids.length; i += MAX_CARD_IDS_PER_REQUEST)
    chunks.push(ids.slice(i, i + MAX_CARD_IDS_PER_REQUEST))
  return chunks
}

/**
 * Resolves the draft's card ids to cards. `matchedThisVisit` fills in cards
 * just matched, so a new row never flashes empty while the lookup runs.
 * `settled`
 * says every lookup has answered, so a card still missing is unavailable.
 */
export function useDraftCards(
  rows: DraftRow[],
  matchedThisVisit: Map<string, CardSummary>
) {
  // Sorted, so a quantity edit or a re-order keeps the same cache entry.
  // Every card answered so far. A changed id set is a new query with no data
  // yet (even a kept-previous-data option does not carry across a key change
  // in `useQueries`), so rows already read stay readable from here. Only ever
  // grows, so filling it during render is safe to repeat.
  const loaded = useRef(new Map<string, CardSummary>())
  const ids = rows.map((r) => r.cardId).sort()
  const results = useQueries({
    queries: chunked(ids).map((cardId) =>
      getSearchCatalogCardsQueryOptions({
        cardId,
        limit: MAX_CARD_IDS_PER_REQUEST,
      })
    ),
  })
  for (const result of results) {
    if (result.data?.status === 200) {
      for (const card of result.data.data.data ?? [])
        loaded.current.set(card.id, card)
    }
  }
  const cards = new Map([...matchedThisVisit, ...loaded.current])
  const settled = results.every((r) => !r.isPending)
  return { cards, settled }
}
