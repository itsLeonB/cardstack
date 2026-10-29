import { cn } from "cn"
import type { BinderPage, BinderSlot } from "@/generated/models"
import { cardCode, slotKey } from "@/lib/binder"
import type { SlotRef } from "@/lib/binder"

/** One Page's Slots in the Layout's rows and columns; tapping a Slot selects it. */
export function BinderPageGrid({
  page,
  selected,
  picked,
  onSelect,
}: {
  page: BinderPage
  selected?: SlotRef
  /** The source Slot of a pending move, marked until the move lands or is cancelled. */
  picked?: SlotRef
  /** Passes the tapped tile too, so focus can return to it when the panel closes. Left out by a read-only view, whose Slots are not controls. */
  onSelect?: (slot: BinderSlot, tile: HTMLButtonElement) => void
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
          picked={picked?.row === slot.row && picked.column === slot.column}
          onSelect={onSelect && ((tile) => onSelect(slot, tile))}
        />
      ))}
    </div>
  )
}

function SlotTile({
  slot,
  selected,
  picked,
  onSelect,
}: {
  slot: BinderSlot
  selected: boolean
  picked: boolean
  onSelect?: (tile: HTMLButtonElement) => void
}) {
  const where = `Row ${slot.row}, column ${slot.column}`
  const tile = cn(
    "rounded-md border text-left focus-visible:ring-2 focus-visible:ring-ring focus-visible:outline-none",
    selected && "ring-2 ring-ring",
    picked && "outline-2 outline-offset-2 outline-primary outline-dashed"
  )

  if (slot.state === "card" && slot.card) {
    return (
      <SlotFrame
        label={`${where}: ${slot.card.name}, ${slot.card.expansionSet.name}`}
        selected={selected}
        onSelect={onSelect}
        className={cn(tile, "flex flex-col gap-1 bg-card p-1")}
      >
        <img
          src={slot.card.imageUrl}
          alt=""
          loading="lazy"
          className="aspect-[5/7] w-full rounded object-cover"
        />
        <span className="truncate text-xs font-medium">{slot.card.name}</span>
        <span className="truncate text-xs">{cardCode(slot.card)}</span>
      </SlotFrame>
    )
  }

  if (slot.state === "unidentified") {
    return (
      <SlotFrame
        label={`${where}: Unidentified`}
        selected={selected}
        onSelect={onSelect}
        className={cn(
          tile,
          "flex aspect-[5/7] items-center justify-center border-dashed bg-muted p-1 text-center text-xs font-medium"
        )}
      >
        Unidentified
      </SlotFrame>
    )
  }

  return (
    <SlotFrame
      label={`${where}: empty`}
      selected={selected}
      onSelect={onSelect}
      className={cn(tile, "aspect-[5/7] border-dashed")}
    />
  )
}

// A read-only Slot is a plain group, so there is no control to press.
function SlotFrame({
  label,
  selected,
  onSelect,
  className,
  children,
}: {
  label: string
  selected: boolean
  onSelect?: (tile: HTMLButtonElement) => void
  className: string
  children?: React.ReactNode
}) {
  if (!onSelect) {
    return (
      <div role="group" aria-label={label} className={className}>
        {children}
      </div>
    )
  }
  return (
    <button
      type="button"
      aria-label={label}
      aria-pressed={selected}
      onClick={(event) => onSelect(event.currentTarget)}
      className={className}
    >
      {children}
    </button>
  )
}
