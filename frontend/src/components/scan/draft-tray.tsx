import { RiAddLine, RiDeleteBinLine, RiSubtractLine } from "@remixicon/react"
import { Button } from "@/components/ui/button"
import { CardThumb } from "./card-thumb"
import { useDraftCards } from "./use-draft-cards"
import type { CardSummary } from "@/generated/models"
import type { DraftRow } from "@/lib/draft-addition"

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
