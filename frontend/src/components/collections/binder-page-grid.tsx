import { cn } from "cn"
import type { BinderPage, BinderSlot } from "@/generated/models"
import { slotKey } from "@/lib/binder"
import type { SlotRef } from "@/lib/binder"

/** One Page's Slots in the Layout's rows and columns; tapping a Slot selects it. */
export function BinderPageGrid({
  page,
  selected,
  onSelect,
}: {
  page: BinderPage
  /** The Slot whose actions are open. */
  selected?: SlotRef
  /** Passes the tapped tile too, so focus can return to it when the panel closes. */
  onSelect: (slot: BinderSlot, tile: HTMLButtonElement) => void
}) {
  return (
    <div
      className="grid gap-2"
      style={{
        gridTemplateColumns: `repeat(${page.layout.columns}, minmax(0, 1fr))`,
      }}
    >
      {page.slots?.map((slot) => (
        <SlotTile
          key={slotKey({ page: page.page, row: slot.row, column: slot.column })}
          slot={slot}
          selected={
            selected?.row === slot.row && selected.column === slot.column
          }
          onSelect={(tile) => onSelect(slot, tile)}
        />
      ))}
    </div>
  )
}

function SlotTile({
  slot,
  selected,
  onSelect,
}: {
  slot: BinderSlot
  selected: boolean
  onSelect: (tile: HTMLButtonElement) => void
}) {
  const where = `Row ${slot.row}, column ${slot.column}`
  const tile = cn(
    "rounded-md border text-left focus-visible:ring-2 focus-visible:ring-ring focus-visible:outline-none",
    selected && "ring-2 ring-ring"
  )

  if (slot.state === "card" && slot.card) {
    return (
      <button
        type="button"
        aria-label={`${where}: ${slot.card.name}`}
        aria-pressed={selected}
        onClick={(event) => onSelect(event.currentTarget)}
        className={cn(tile, "flex flex-col gap-1 bg-card p-1")}
      >
        <img
          src={slot.card.imageUrl}
          alt=""
          loading="lazy"
          className="aspect-[5/7] w-full rounded object-cover"
        />
        <span className="truncate text-xs font-medium">{slot.card.name}</span>
        <span className="text-xs">#{slot.card.localId}</span>
      </button>
    )
  }

  if (slot.state === "unidentified") {
    return (
      <button
        type="button"
        aria-label={`${where}: Unidentified`}
        aria-pressed={selected}
        onClick={(event) => onSelect(event.currentTarget)}
        className={cn(
          tile,
          "flex aspect-[5/7] items-center justify-center border-dashed bg-muted p-1 text-center text-xs font-medium"
        )}
      >
        Unidentified
      </button>
    )
  }

  return (
    <button
      type="button"
      aria-label={`${where}: empty`}
      aria-pressed={selected}
      onClick={(event) => onSelect(event.currentTarget)}
      className={cn(tile, "aspect-[5/7] border-dashed")}
    />
  )
}
