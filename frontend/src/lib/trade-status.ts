const STATUS_LABELS = {
  open: "Open",
  negotiating: "Negotiating",
  locked: "Locked",
  executed: "Executed",
  cancelled: "Cancelled",
} satisfies Record<string, string>

/** The Trade's state as a label; a newer state the UI does not know reads as itself. */
export function tradeStatusLabel(status: string): string {
  // SAFETY: a status outside the map indexes to undefined, handled by ??.
  return STATUS_LABELS[status as keyof typeof STATUS_LABELS] ?? status
}

/**
 * The states a participant may still cancel; executed and cancelled are
 * history. Mirrors the backend's rule — the API still refuses the rest, so a
 * stale entry here only hides or shows a button, never allows a cancel.
 */
export const CANCELLABLE_STATUSES: ReadonlySet<string> = new Set([
  "open",
  "negotiating",
  "locked",
])
