import { Button } from "@/components/ui/button"
import type { BinderSlotTarget } from "@/generated/models"
import { slotKey } from "@/lib/binder"

/**
 * Where a binder Card sits: one button per Slot holding it. Tapping one opens
 * the binder on that Page. Renders nothing for a plain Collection.
 */
export function CardPositions({
  positions,
  onFind,
}: {
  positions: BinderSlotTarget[]
  onFind: (position: BinderSlotTarget) => void
}) {
  return (
    <div className="flex flex-wrap gap-1">
      {positions.map((position) => (
        <Button
          key={slotKey(position)}
          variant="outline"
          size="sm"
          // Above the card's stretched title link, which covers the tile.
          className="relative z-10"
          // A fresh object per tap, so tapping the same position again jumps.
          onClick={() => onFind({ ...position })}
        >
          Page {position.page} · row {position.row} · col {position.column}
        </Button>
      ))}
    </div>
  )
}
