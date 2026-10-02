export const GRID_GAP = 16

// Container-width equivalents of the old viewport breakpoints (sm/md/lg: 640,
// 768, 1024), less the page container's 2 x 24px side padding.
const COLUMN_STEPS = [
  { minWidth: 960, columns: 5 },
  { minWidth: 704, columns: 4 },
  { minWidth: 576, columns: 3 },
]

export function columnsForWidth(width: number) {
  return COLUMN_STEPS.find((step) => width >= step.minWidth)?.columns ?? 2
}

/** Split a flat list into rows of `columns` items; the last row may be short. */
export function chunkRows<T>(items: T[], columns: number) {
  const rows: T[][] = []
  for (let start = 0; start < items.length; start += columns) {
    rows.push(items.slice(start, start + columns))
  }
  return rows
}

/**
 * True once the last rendered row reaches the end of the loaded rows. The
 * caller folds "has a next page, nothing in flight, no error" into `canLoad`,
 * so a failed request is never retried by scrolling alone.
 */
export function shouldLoadMore({
  lastRenderedRow,
  rowCount,
  canLoad,
}: {
  lastRenderedRow: number | undefined
  rowCount: number
  canLoad: boolean
}) {
  return (
    canLoad && lastRenderedRow !== undefined && lastRenderedRow >= rowCount - 1
  )
}

// Rough tile height before it is measured: art at 5:7 plus ~150px of text and badges.
export function estimateRowHeight(width: number, columns: number) {
  const tileWidth = (width - GRID_GAP * (columns - 1)) / columns
  return Math.round((tileWidth * 7) / 5) + 150
}
