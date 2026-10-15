import type { BinderPage, BinderSlot } from "@/generated/models"

/** One Page's Slots in the Layout's rows and columns. */
export function BinderPageGrid({ page }: { page: BinderPage }) {
  return (
    <div
      className="grid gap-2"
      style={{
        gridTemplateColumns: `repeat(${page.layout.columns}, minmax(0, 1fr))`,
      }}
    >
      {page.slots?.map((slot) => (
        <SlotTile key={`${slot.row},${slot.column}`} slot={slot} />
      ))}
    </div>
  )
}

function SlotTile({ slot }: { slot: BinderSlot }) {
  const where = `Row ${slot.row}, column ${slot.column}`

  if (slot.state === "card" && slot.card) {
    return (
      <div
        role="group"
        aria-label={`${where}: ${slot.card.name}`}
        className="flex flex-col gap-1 rounded-md border bg-card p-1"
      >
        <img
          src={slot.card.imageUrl}
          alt=""
          loading="lazy"
          className="aspect-[5/7] w-full rounded object-cover"
        />
        <p className="truncate text-xs font-medium">{slot.card.name}</p>
        <p className="text-xs">#{slot.card.localId}</p>
      </div>
    )
  }

  if (slot.state === "unidentified") {
    return (
      <div
        role="group"
        aria-label={`${where}: Unidentified`}
        className="flex aspect-[5/7] items-center justify-center rounded-md border border-dashed bg-muted p-1 text-center text-xs font-medium"
      >
        Unidentified
      </div>
    )
  }

  return (
    <div
      role="group"
      aria-label={`${where}: empty`}
      className="aspect-[5/7] rounded-md border border-dashed"
    />
  )
}
