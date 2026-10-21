import { fitLongSide, UPLOAD_LONG_SIDE } from "./frame-crop"

// Flattens a photographed Page and cuts it into one JPEG per pocket. The
// geometry is pure, so a test or an offline script can check a cut without a
// canvas; `cutPage` is the only part that draws.

export interface Point {
  x: number
  y: number
}

/** Top-left, top-right, bottom-right, bottom-left: the outer corners of the pocket grid, each axis 0 to 1 across the photo. */
export type Corners = [Point, Point, Point, Point]

export interface Grid {
  rows: number
  columns: number
}

/** Share of a cell's width and height cropped off each side, so the neighbouring pockets and the binder's edge stay out. */
export const CELL_MARGIN = 0.03

/** One pocket, 1-based like the API's Slots. Edges are in grid units: a cell is 1 by 1. */
export interface CellRegion {
  row: number
  column: number
  x0: number
  y0: number
  x1: number
  y1: number
}

/** The 1-based row and column of a cell, from its index in grid order. */
export function cellPosition(index: number, columns: number) {
  return {
    row: Math.floor(index / columns) + 1,
    column: (index % columns) + 1,
  }
}

export function cellRegions(grid: Grid, margin = CELL_MARGIN): CellRegion[] {
  const cells: CellRegion[] = []
  for (let row = 0; row < grid.rows; row++) {
    for (let column = 0; column < grid.columns; column++) {
      cells.push({
        row: row + 1,
        column: column + 1,
        x0: column + margin,
        y0: row + margin,
        x1: column + 1 - margin,
        y1: row + 1 - margin,
      })
    }
  }
  return cells
}

/** Maps a point of the grid (0 to columns across, 0 to rows down) to the photo (0 to 1 on each axis). */
export function gridToPhoto(
  corners: Corners,
  grid: Grid
): (x: number, y: number) => Point {
  const quad = unitSquareToQuad(corners)
  return (x, y) => quad(x / grid.columns, y / grid.rows)
}

/** The output size in pixels of one cell, its long side at most `limit`. */
export function cellSize(
  corners: Corners,
  grid: Grid,
  cell: CellRegion,
  width: number,
  height: number,
  limit = UPLOAD_LONG_SIDE
) {
  const map = gridToPhoto(corners, grid)
  const px = (x: number, y: number) => {
    const p = map(x, y)
    return { x: p.x * width, y: p.y * height }
  }
  const topLeft = px(cell.x0, cell.y0)
  const topRight = px(cell.x1, cell.y0)
  const bottomRight = px(cell.x1, cell.y1)
  const bottomLeft = px(cell.x0, cell.y1)
  const w = Math.max(
    distance(topLeft, topRight),
    distance(bottomLeft, bottomRight)
  )
  const h = Math.max(
    distance(topLeft, bottomLeft),
    distance(topRight, bottomRight)
  )
  const out = fitLongSide(Math.round(w), Math.round(h), limit)
  return { w: Math.max(1, out.w), h: Math.max(1, out.h) }
}

/** One cell cut from an RGBA photo, the pure half of `cutPage`: an offline script calls it too, so its numbers match the app. */
export function cutCellPixels(
  pixels: Uint8ClampedArray,
  width: number,
  height: number,
  corners: Corners,
  grid: Grid,
  cell: CellRegion
) {
  const size = cellSize(corners, grid, cell, width, height)
  const map = gridToPhoto(corners, grid)
  const out = new Uint8ClampedArray(size.w * size.h * 4)
  for (let j = 0; j < size.h; j++) {
    for (let i = 0; i < size.w; i++) {
      const u = cell.x0 + ((cell.x1 - cell.x0) * (i + 0.5)) / size.w
      const v = cell.y0 + ((cell.y1 - cell.y0) * (j + 0.5)) / size.h
      const p = map(u, v)
      sampleInto(
        pixels,
        width,
        height,
        p.x * width - 0.5,
        p.y * height - 0.5,
        out,
        (j * size.w + i) * 4
      )
    }
  }
  return { width: size.w, height: size.h, pixels: out }
}

/**
 * The longest side the whole photo needs, so that the largest cell of the
 * Layout can still reach the upload size after the margin is cropped.
 */
export function frameLongSide(grid: Grid) {
  return Math.ceil(
    (Math.max(grid.rows, grid.columns) * UPLOAD_LONG_SIDE) /
      (1 - 2 * CELL_MARGIN)
  )
}

// Below this, a corner's turn is too close to flat for the mapping to be usable.
const MIN_TURN = 1e-4

/**
 * Why the corners cannot be cut, or null when they form a convex quadrilateral
 * on the photo. Walking the corners in order, every turn must go the same way:
 * a crossed or folded shape flips sign, and a flat one has no turn at all.
 */
export function cornersProblem(corners: Corners): string | null {
  if (corners.some(({ x, y }) => x < 0 || x > 1 || y < 0 || y > 1))
    return "The corners must stay on the photo."
  const turns = corners.map((p, i) => {
    const a = corners[(i + 1) % 4]
    const b = corners[(i + 2) % 4]
    return (a.x - p.x) * (b.y - a.y) - (a.y - p.y) * (b.x - a.x)
  })
  const sign = Math.sign(turns[0])
  if (turns.some((t) => Math.abs(t) < MIN_TURN || Math.sign(t) !== sign))
    return "Move the corners so they make a four-sided shape with no crossed or flat sides."
  return null
}

/** Flattens the photo's quadrilateral and cuts it into one JPEG per cell, row by row. The photo is drawn to a canvas here, so this needs a browser. */
export async function cutPage(
  frame: Blob,
  corners: Corners,
  grid: Grid
): Promise<Blob[]> {
  const bitmap = await createImageBitmap(frame)
  const { width, height } = bitmap
  const source = canvasOf(width, height)
  source.context.drawImage(bitmap, 0, 0)
  bitmap.close()
  const pixels = source.context.getImageData(0, 0, width, height).data

  const jpegs: Blob[] = []
  for (const cell of cellRegions(grid)) {
    const cut = cutCellPixels(pixels, width, height, corners, grid, cell)
    const out = canvasOf(cut.width, cut.height)
    const image = out.context.createImageData(cut.width, cut.height)
    image.data.set(cut.pixels)
    out.context.putImageData(image, 0, 0)
    jpegs.push(await encodeJpeg(out.canvas))
  }
  return jpegs
}

function distance(a: Point, b: Point) {
  return Math.hypot(a.x - b.x, a.y - b.y)
}

/**
 * The projective map from the unit square to a quadrilateral (Heckbert's
 * square-to-quad), so a page photographed at an angle comes out square.
 */
function unitSquareToQuad([p0, p1, p2, p3]: Corners) {
  const dx1 = p1.x - p2.x
  const dx2 = p3.x - p2.x
  const dx3 = p0.x - p1.x + p2.x - p3.x
  const dy1 = p1.y - p2.y
  const dy2 = p3.y - p2.y
  const dy3 = p0.y - p1.y + p2.y - p3.y
  let g = 0
  let h = 0
  if (dx3 !== 0 || dy3 !== 0) {
    const den = dx1 * dy2 - dx2 * dy1
    g = (dx3 * dy2 - dx2 * dy3) / den
    h = (dx1 * dy3 - dx3 * dy1) / den
  }
  const a = p1.x - p0.x + g * p1.x
  const b = p3.x - p0.x + h * p3.x
  const d = p1.y - p0.y + g * p1.y
  const e = p3.y - p0.y + h * p3.y
  return (u: number, v: number): Point => {
    const w = g * u + h * v + 1
    return {
      x: (a * u + b * v + p0.x) / w,
      y: (d * u + e * v + p0.y) / w,
    }
  }
}

/** Bilinear sample of an RGBA buffer, clamped at the edges, written as opaque RGBA at `at`. */
function sampleInto(
  pixels: Uint8ClampedArray,
  width: number,
  height: number,
  x: number,
  y: number,
  out: Uint8ClampedArray,
  at: number
) {
  const fx = Math.floor(x)
  const fy = Math.floor(y)
  const tx = x - fx
  const ty = y - fy
  const xa = Math.min(Math.max(fx, 0), width - 1)
  const xb = Math.min(Math.max(fx + 1, 0), width - 1)
  const ya = Math.min(Math.max(fy, 0), height - 1)
  const yb = Math.min(Math.max(fy + 1, 0), height - 1)
  for (let c = 0; c < 3; c++) {
    const top =
      pixels[(ya * width + xa) * 4 + c] * (1 - tx) +
      pixels[(ya * width + xb) * 4 + c] * tx
    const bottom =
      pixels[(yb * width + xa) * 4 + c] * (1 - tx) +
      pixels[(yb * width + xb) * 4 + c] * tx
    out[at + c] = top * (1 - ty) + bottom * ty
  }
  out[at + 3] = 255
}

function canvasOf(width: number, height: number) {
  const canvas = document.createElement("canvas")
  canvas.width = width
  canvas.height = height
  const context = canvas.getContext("2d")
  if (!context) throw new Error("Canvas is not available")
  return { canvas, context }
}

function encodeJpeg(canvas: HTMLCanvasElement): Promise<Blob> {
  return new Promise((resolve, reject) =>
    canvas.toBlob(
      (blob) => (blob ? resolve(blob) : reject(new Error("Encoding failed"))),
      "image/jpeg",
      0.9
    )
  )
}
