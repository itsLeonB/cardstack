import type { ReviewBinderCell, ReviewPoint } from "@/generated/models"
import { cellRegions, cornersProblem, cutCellPixels } from "./page-cut"
import type { Corners } from "./page-cut"
import type { Pixels } from "./photo-pixels"

/**
 * The cut the matcher would embed for one binder cell (ADR-0027): the cell's
 * region of the stored Page photo, cut with the grid and corners the user
 * recorded. Null when the item has no usable corners, so there is nothing to show.
 */
export function binderCellCut(
  photo: Pixels,
  binder: ReviewBinderCell
): Pixels | null {
  const corners = toCorners(binder.corners ?? [])
  if (!corners || cornersProblem(corners)) return null
  const grid = { rows: binder.gridRows, columns: binder.gridColumns }
  const cell = cellRegions(grid).find(
    (region) =>
      region.row === binder.slotRow && region.column === binder.slotColumn
  )
  if (!cell) return null
  return cutCellPixels(
    photo.pixels,
    photo.width,
    photo.height,
    corners,
    grid,
    cell
  )
}

function toCorners(points: ReviewPoint[]): Corners | null {
  const [a, b, c, d] = points
  return points.length === 4 && a && b && c && d ? [a, b, c, d] : null
}
