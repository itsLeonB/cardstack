import { Button } from "@/components/ui/button"
import type { SlotRef } from "@/lib/binder"

/**
 * Shown while a Slot is picked for a move. The source stays picked while the
 * user turns to another Page, so the target can be on any Page of the binder.
 */
export function BinderMoveBanner({
  source,
  label,
  error,
  onCancel,
}: {
  source: SlotRef
  /** What the source Slot holds: a Card's name, or Unidentified. */
  label: string
  error: string | null
  onCancel: () => void
}) {
  return (
    <section
      aria-label="Move a card"
      className="flex flex-col gap-2 rounded-md border p-3"
    >
      <p className="text-sm">
        Moving {label} from Page {source.page}, row {source.row}, column{" "}
        {source.column}. Turn to the target Page and tap its Slot. Tapping the
        source Slot again cancels.
      </p>
      {error && (
        <p role="alert" className="text-sm text-destructive">
          {error}
        </p>
      )}
      <div>
        <Button variant="ghost" onClick={onCancel}>
          Cancel move
        </Button>
      </div>
    </section>
  )
}
