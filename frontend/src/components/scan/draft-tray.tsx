import { useQueries } from "@tanstack/react-query"
import { RiAddLine, RiDeleteBinLine, RiSubtractLine } from "@remixicon/react"
import { Button } from "@/components/ui/button"
import { getSearchCatalogCardsQueryOptions } from "@/generated/endpoints/catalog/catalog"
import type { CardSummary } from "@/generated/models"
import type { DraftRow } from "@/lib/draft-addition"
import { imageSources } from "@/lib/image"

// The catalog's `cardId` filter takes at most 100 ids per request.
const CHUNK = 100

function chunked(ids: string[]) {
  const chunks: string[][] = []
  for (let i = 0; i < ids.length; i += CHUNK)
    chunks.push(ids.slice(i, i + CHUNK))
  return chunks
}

/** Resolves the draft's card ids to cards. `known` fills in cards just matched, so a new row never flashes empty while the lookup runs. */
function useDraftCards(rows: DraftRow[], known: Map<string, CardSummary>) {
  // Sorted, so a quantity edit or a re-order keeps the same cache entry.
  const ids = rows.map((r) => r.cardId).sort()
  const results = useQueries({
    queries: chunked(ids).map((cardId) =>
      getSearchCatalogCardsQueryOptions({ cardId, limit: CHUNK })
    ),
  })
  const cards = new Map(known)
  for (const result of results) {
    if (result.data?.status === 200) {
      for (const card of result.data.data.data ?? []) cards.set(card.id, card)
    }
  }
  return cards
}

export function DraftTray({
  rows,
  known,
  onRaise,
  onLower,
  onRemove,
  onDiscard,
  onReview,
}: {
  rows: DraftRow[]
  known: Map<string, CardSummary>
  onRaise: (cardId: string) => void
  onLower: (cardId: string) => void
  onRemove: (cardId: string) => void
  onDiscard: () => void
  onReview: () => void
}) {
  const cards = useDraftCards(rows, known)
  const total = rows.reduce((sum, r) => sum + r.quantity, 0)

  return (
    <section aria-labelledby="draft-heading" className="flex flex-col gap-3">
      <h2 id="draft-heading" className="font-heading text-lg font-medium">
        Draft ({total} {total === 1 ? "card" : "cards"})
      </h2>
      {rows.length === 0 ? (
        <p className="text-sm text-muted-foreground">
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
                aria-label={card?.name}
                className="flex items-center gap-2 rounded-xl border p-2"
              >
                {card?.imageUrl ? (
                  <img
                    src={imageSources(card.imageUrl, "cardTile").src}
                    alt=""
                    width={48}
                    height={67}
                    className="aspect-[5/7] w-12 rounded object-cover"
                  />
                ) : (
                  <div className="aspect-[5/7] w-12 rounded bg-muted" />
                )}
                <span className="min-w-0 flex-1 truncate text-sm font-medium">
                  {card?.name ?? "Loading…"}
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
