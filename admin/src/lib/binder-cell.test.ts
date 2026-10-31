import { describe, expect, it } from "vitest"
import { binderCellCut } from "./binder-cell"
import { CELL_MARGIN } from "./page-cut"
import type { Pixels } from "./photo-pixels"

// The user app's cut fixtures (frontend/src/lib/page-cut.test.ts): a page
// photographed square-on, with the left half black and the right half white.
const whole = [
  { x: 0, y: 0 },
  { x: 1, y: 0 },
  { x: 1, y: 1 },
  { x: 0, y: 1 },
]

const width = 40
const height = 20
const halves = new Uint8ClampedArray(width * height * 4)
for (let at = 0; at < halves.length; at += 4) {
  const x = (at / 4) % width
  halves.set([x < width / 2 ? 0 : 255, 0, 0, 255], at)
}
const photo: Pixels = { width, height, pixels: halves }

const cell = {
  page: 1,
  slotRow: 1,
  slotColumn: 2,
  gridRows: 1,
  gridColumns: 2,
  corners: whole,
}

describe("binderCellCut", () => {
  it("cuts the cell at the row and column the item names, from its own half", () => {
    const left = binderCellCut(photo, { ...cell, slotColumn: 1 })
    const right = binderCellCut(photo, { ...cell, slotColumn: 2 })

    expect(left?.width).toBe(19)
    expect(left?.height).toBe(19)
    expect(left?.pixels[0]).toBe(0)
    expect(right?.pixels[0]).toBe(255)
  })

  it("keeps the margin the user app matched with", () => {
    const cut = binderCellCut(photo, { ...cell, slotColumn: 1 })

    // A 40 by 20 photo of a 2 by 1 grid: each cell is 20 by 20 pixels, less the margin on each side.
    expect(cut?.width).toBe(Math.round(20 * (1 - 2 * CELL_MARGIN)))
  })

  it("is null when the item has no corners, since there is nothing to cut", () => {
    expect(binderCellCut(photo, { ...cell, corners: null })).toBeNull()
  })

  it("is null when the corners cross, since the mapping would fold", () => {
    const crossed = [whole[0], whole[2], whole[1], whole[3]]

    expect(binderCellCut(photo, { ...cell, corners: crossed })).toBeNull()
  })
})
