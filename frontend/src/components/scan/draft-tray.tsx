import { Button } from "@/components/ui/button"
import { DraftRowItem } from "./draft-row"
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
            return (
              <DraftRowItem
                key={row.cardId}
                card={card}
                name={card?.name ?? (settled ? "Card unavailable" : "Loading…")}
                quantity={row.quantity}
                quantityLabel={String(row.quantity)}
                onRaise={() => onRaise(row.cardId)}
                onLower={() => onLower(row.cardId)}
                onRemove={() => onRemove(row.cardId)}
              />
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
