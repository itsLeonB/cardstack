import { useRef } from "react"
import { useQueries } from "@tanstack/react-query"
import { RiAddLine, RiDeleteBinLine, RiSubtractLine } from "@remixicon/react"
import { Button } from "@/components/ui/button"
import { CardThumb } from "./card-thumb"
import { getSearchCatalogCardsQueryOptions } from "@/generated/endpoints/catalog/catalog"
import type { CardSummary } from "@/generated/models"
import type { DraftRow } from "@/lib/draft-addition"

// The catalog's `cardId` filter takes at most 100 ids per request.
const MAX_CARD_IDS_PER_REQUEST = 100

function chunked(ids: string[]) {
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
function useDraftCards(
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

export function DraftTray({
  rows,
  matchedThisVisit,
  onRaise,
  onLower,
  onRemove,
  onDiscard,
  onReview,
}: {
  rows: DraftRow[]
  matchedThisVisit: Map<string, CardSummary>
  onRaise: (cardId: string) => void
  onLower: (cardId: string) => void
  onRemove: (cardId: string) => void
  onDiscard: () => void
  onReview: () => void
}) {
  const { cards, settled } = useDraftCards(rows, matchedThisVisit)
  const total = rows.reduce((sum, r) => sum + r.quantity, 0)

  return (
    <section aria-labelledby="draft-heading" className="flex flex-col gap-3">
      <h2 id="draft-heading" className="font-heading text-lg font-medium">
        Draft ({total} {total === 1 ? "card" : "cards"})
      </h2>
      {rows.length === 0 ? (
        <p className="text-sm text-foreground">
          Scanned cards will appear here.
        </p>
      ) : (
        <ul className="flex flex-col gap-2">
          {rows.map((row) => {
            const card = cards.get(row.cardId)
            const name = card?.name ?? "card"
            return (
              <li
                key={row.cardId}
                className="flex items-center gap-2 rounded-xl border p-2"
              >
                <CardThumb card={card} />
                <span className="min-w-0 flex-1 truncate text-sm font-medium">
                  {card?.name ?? (settled ? "Card unavailable" : "Loading…")}
                </span>
                <Button
                  variant="outline"
                  size="icon-sm"
                  aria-label={`Lower quantity of ${name}`}
                  disabled={row.quantity <= 1}
                  onClick={() => onLower(row.cardId)}
                >
                  <RiSubtractLine />
                </Button>
                <span className="w-6 text-center tabular-nums">
                  {row.quantity}
                </span>
                <Button
                  variant="outline"
                  size="icon-sm"
                  aria-label={`Raise quantity of ${name}`}
                  onClick={() => onRaise(row.cardId)}
                >
                  <RiAddLine />
                </Button>
                <Button
                  variant="ghost"
                  size="icon-sm"
                  aria-label={`Remove ${name}`}
                  onClick={() => onRemove(row.cardId)}
                >
                  <RiDeleteBinLine />
                </Button>
              </li>
            )
          })}
        </ul>
      )}
      <div className="flex flex-wrap gap-2">
        <Button disabled={rows.length === 0} onClick={onReview}>
          Review and add
        </Button>
        <Button
          variant="outline"
          disabled={rows.length === 0}
          onClick={onDiscard}
        >
          Discard draft
        </Button>
      </div>
    </section>
  )
}
