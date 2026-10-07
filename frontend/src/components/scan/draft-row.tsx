import type { ReactNode } from "react"
import { RiAddLine, RiDeleteBinLine, RiSubtractLine } from "@remixicon/react"
import { Button } from "@/components/ui/button"
import { CardThumb } from "./card-thumb"
import type { CardSummary } from "@/generated/models"

/** One draft row: thumb, name, and the Lower / Raise / Remove cluster, shared by the tray and the review step. */
export function DraftRowItem({
  card,
  name,
  detail,
  quantityLabel,
  quantity,
  disabled = false,
  footer,
  onRetry,
  onRaise,
  onLower,
  onRemove,
}: {
  card?: CardSummary
  /** What to show when the card is unknown (loading or unavailable). */
  name: string
  /** A second line under the name. */
  detail?: ReactNode
  quantityLabel: string
  quantity: number
  disabled?: boolean
  footer?: ReactNode
  /** Set when the card's lookup failed: shows a Retry button. */
  onRetry?: () => void
  onRaise: () => void
  onLower: () => void
  onRemove: () => void
}) {
  const label = card?.name ?? "card"
  return (
    <li className="flex flex-col gap-1 rounded-xl border p-2">
      <div className="flex items-center gap-2">
        <CardThumb card={card} />
        <span className="flex min-w-0 flex-1 flex-col">
          <span className="truncate text-sm font-medium">{name}</span>
          {detail}
          {onRetry && (
            <Button
              variant="outline"
              size="xs"
              className="mt-1 self-start"
              onClick={onRetry}
            >
              Retry
            </Button>
          )}
        </span>
        <Button
          variant="outline"
          size="icon-sm"
          aria-label={`Lower quantity of ${label}`}
          disabled={disabled || quantity <= 1}
          onClick={onLower}
        >
          <RiSubtractLine />
        </Button>
        <span className="w-8 text-center tabular-nums">{quantityLabel}</span>
        <Button
          variant="outline"
          size="icon-sm"
          aria-label={`Raise quantity of ${label}`}
          disabled={disabled}
          onClick={onRaise}
        >
          <RiAddLine />
        </Button>
        <Button
          variant="ghost"
          size="icon-sm"
          aria-label={`Remove ${label}`}
          disabled={disabled}
          onClick={onRemove}
        >
          <RiDeleteBinLine />
        </Button>
      </div>
      {footer}
    </li>
  )
}
