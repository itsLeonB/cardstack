import { Button } from "@/components/ui/button"
import type { BinderSlotTarget } from "@/generated/models"

/**
 * Where a binder Card sits: one button per Slot holding it. Tapping one opens
 * the binder on that Page. Renders nothing for a plain Collection.
 */
export function CardPositions({
  positions,
  onFind,
}: {
  positions?: BinderSlotTarget[]
  onFind?: (position: BinderSlotTarget) => void
}) {
  if (!positions?.length) return null
  return (
    <div className="flex flex-wrap gap-1">
      {positions.map((position) => (
        <Button
          key={`${position.page},${position.row},${position.column}`}
          variant="outline"
          size="sm"
          // Above the card's stretched title link, which covers the tile.
          className="relative z-10"
          onClick={() => onFind?.(position)}
        >
          Page {position.page} · row {position.row} · col {position.column}
        </Button>
      ))}
    </div>
  )
}
