/**
 * Build-time switch for card scanning: only `VITE_SCAN_ENABLED=true` turns it
 * on, so production stays off until the real matcher lands. Read per call so
 * tests can stub it.
 */
export function scanEnabled(): boolean {
  return import.meta.env.VITE_SCAN_ENABLED === "true"
}
