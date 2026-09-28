/** Default page size used across catalog browse/search views (matches the API default). */
export const CATALOG_PAGE_SIZE = 24

/**
 * Formats an RFC3339 release date for display, e.g. "March 3, 2011".
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
