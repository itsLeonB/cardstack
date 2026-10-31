/**
 * Says that photos of corrected scans may be used to improve matching. It is
 * shown only where a scan can end in a Correction, and has no setting to turn
 * it off (ADR-0027).
 */
export function PhotoUseNotice() {
  return (
    <p className="text-sm">
      Photos of corrected scans may be used to improve matching.
    </p>
  )
}
