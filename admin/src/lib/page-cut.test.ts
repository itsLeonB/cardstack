import { describe, expect, it } from "vitest"
import {
  CELL_MARGIN,
  cellRegions,
  cellSize,
  cutCellPixels,
  gridToPhoto,
} from "./page-cut"
import type { Corners } from "./page-cut"

const whole: Corners = [
  { x: 0, y: 0 },
  { x: 1, y: 0 },
  { x: 1, y: 1 },
  { x: 0, y: 1 },
]

// A page photographed at an angle: the top edge is narrower than the bottom.
const trapezoid: Corners = [
  { x: 0.2, y: 0.1 },
  { x: 0.8, y: 0.15 },
  { x: 0.9, y: 0.9 },
  { x: 0.1, y: 0.85 },
]

describe("cellRegions", () => {
  it("cuts the grid into 1-based cells, each inset by the margin", () => {
    const cells = cellRegions({ rows: 2, columns: 3 })

    expect(cells).toHaveLength(6)
    expect(cells[0]).toEqual({
      row: 1,
      column: 1,
      x0: CELL_MARGIN,
      y0: CELL_MARGIN,
      x1: 1 - CELL_MARGIN,
      y1: 1 - CELL_MARGIN,
    })
    expect(cells[5]).toMatchObject({ row: 2, column: 3, x0: 2 + CELL_MARGIN })
  })
})

describe("gridToPhoto", () => {
  it("maps a whole-photo grid proportionally", () => {
    const map = gridToPhoto(whole, { rows: 2, columns: 4 })

    expect(map(2, 1)).toEqual({ x: 0.5, y: 0.5 })
    expect(map(4, 2)).toEqual({ x: 1, y: 1 })
  })

  it("sends the grid's corners to the photo's corners on a skewed quadrilateral", () => {
    const map = gridToPhoto(trapezoid, { rows: 1, columns: 1 })

    const corners = [map(0, 0), map(1, 0), map(1, 1), map(0, 1)]
    corners.forEach((p, i) => {
      expect(p.x).toBeCloseTo(trapezoid[i].x)
      expect(p.y).toBeCloseTo(trapezoid[i].y)
    })
  })
})

describe("cellSize", () => {
  it("is the cell's share of an upright photo, unscaled below the limit", () => {
    const grid = { rows: 2, columns: 2 }
    const size = cellSize(whole, grid, cellRegions(grid)[0], 1000, 800)

    // A cell is 500 by 400 pixels; the 3 percent margin leaves 94 percent of each side.
    expect(size.w).toBeCloseTo(470)
    expect(size.h).toBeCloseTo(376)
  })

  it("caps the long side at the limit", () => {
    const grid = { rows: 1, columns: 1 }
    const size = cellSize(whole, grid, cellRegions(grid)[0], 4000, 3000, 1024)

    expect(Math.max(size.w, size.h)).toBe(1024)
  })
})

describe("cutCellPixels", () => {
  // Left half black, right half white: a cell only samples its own half, the margin keeps it off the seam.
  const width = 40
  const height = 20
  const halves = new Uint8ClampedArray(width * height * 4)
  for (let at = 0; at < halves.length; at += 4) {
    const x = (at / 4) % width
    halves.set([x < width / 2 ? 0 : 255, 0, 0, 255], at)
  }
  const grid = { rows: 1, columns: 2 }

  it("returns the cell's size, the same one the app computes", () => {
    const cell = cellRegions(grid)[0]
    const cut = cutCellPixels(halves, width, height, whole, grid, cell)

    expect(cut.width).toBe(cellSize(whole, grid, cell, width, height).w)
    expect(cut.height).toBe(cellSize(whole, grid, cell, width, height).h)
    expect(cut.pixels).toHaveLength(cut.width * cut.height * 4)
  })

  it("cuts each cell from its own region of an axis-aligned photo", () => {
    const [left, right] = cellRegions(grid)

    const leftCut = cutCellPixels(halves, width, height, whole, grid, left)
    const rightCut = cutCellPixels(halves, width, height, whole, grid, right)

    expect(leftCut.width).toBe(19)
    expect(leftCut.height).toBe(19)
    for (let at = 0; at < leftCut.pixels.length; at += 4) {
      expect(leftCut.pixels[at]).toBe(0)
      expect(rightCut.pixels[at]).toBe(255)
      expect(rightCut.pixels[at + 3]).toBe(255)
    }
  })
})
