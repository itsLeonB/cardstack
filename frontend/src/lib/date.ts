/**
 * Formats an RFC3339 date for display, e.g. "March 3, 2011".
 * Returns null when the date is absent or unparseable so callers can omit it.
 */
export function formatReleaseDate(releaseDate?: string): string | null {
  if (!releaseDate) return null
  const date = new Date(releaseDate)
  if (Number.isNaN(date.getTime())) return null
  return date.toLocaleDateString(undefined, {
    year: "numeric",
    month: "long",
    day: "numeric",
  })
}

const MINUTE_MS = 60_000
const HOUR_MS = 60 * MINUTE_MS
const DAY_MS = 24 * HOUR_MS

/**
 * Formats an RFC3339 timestamp as a relative age, e.g. "3 hours ago".
 * Falls back to a calendar date past a month and returns null when the
 * timestamp is absent, unparseable or in the future.
 */
export function formatAge(iso?: string): string | null {
  if (!iso) return null
  const then = new Date(iso).getTime()
  if (Number.isNaN(then)) return null
  const diff = Date.now() - then
  if (diff < 0) return null
  if (diff < MINUTE_MS) return "just now"
  if (diff < HOUR_MS)
    return `${Math.floor(diff / MINUTE_MS)} minute${diff < 2 * MINUTE_MS ? "" : "s"} ago`
  if (diff < DAY_MS)
    return `${Math.floor(diff / HOUR_MS)} hour${diff < 2 * HOUR_MS ? "" : "s"} ago`
  if (diff < 30 * DAY_MS)
    return `${Math.floor(diff / DAY_MS)} day${diff < 2 * DAY_MS ? "" : "s"} ago`
  return formatReleaseDate(iso)
}
