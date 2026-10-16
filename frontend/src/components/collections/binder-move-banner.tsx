import { useEffect, useRef } from "react"
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
  const ref = useRef<HTMLDivElement>(null)
  // Focus moves here when a move is picked, so the next step is read out.
  useEffect(() => {
    ref.current?.focus()
  }, [])
  useEffect(() => {
    function cancelOnEscape(event: KeyboardEvent) {
      if (event.key === "Escape") onCancel()
    }
    document.addEventListener("keydown", cancelOnEscape)
    return () => document.removeEventListener("keydown", cancelOnEscape)
  }, [onCancel])

  return (
    <div
      ref={ref}
      role="status"
      tabIndex={-1}
      aria-label="Move a card"
      className="flex flex-col gap-2 rounded-md border p-3 focus:outline-none"
    >
      <p className="text-sm">
        Moving {label} from Page {source.page}, row {source.row}, column{" "}
        {source.column}. Turn to the target Page and tap its Slot. Tapping the
        source Slot again or pressing Escape cancels.
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
    </div>
  )
}
