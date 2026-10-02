import { describe, expect, it } from "vitest"
import { chunkRows, columnsForWidth, shouldLoadMore } from "./grid-layout"

describe("columnsForWidth", () => {
  it.each([
    [0, 2],
    [575, 2],
    [576, 3],
    [703, 3],
    [704, 4],
    [959, 4],
    [960, 5],
    [1104, 5],
  ])("%ipx wide gives %i columns", (width, columns) => {
    expect(columnsForWidth(width)).toBe(columns)
  })
})

describe("chunkRows", () => {
  it("fills rows left to right and leaves the last one short", () => {
    expect(chunkRows([1, 2, 3, 4, 5], 2)).toEqual([[1, 2], [3, 4], [5]])
  })

  it("returns no rows for an empty list", () => {
    expect(chunkRows([], 3)).toEqual([])
  })
})

describe("shouldLoadMore", () => {
  const base = { lastRenderedRow: 9, rowCount: 10, canLoad: true }

  it("fires when the last row is rendered", () => {
    expect(shouldLoadMore(base)).toBe(true)
  })

  it("waits while rows below the window are unrendered", () => {
    expect(shouldLoadMore({ ...base, lastRenderedRow: 5 })).toBe(false)
  })

  it("never fires while loading is not allowed", () => {
    expect(shouldLoadMore({ ...base, canLoad: false })).toBe(false)
  })

  it("does nothing before any row renders", () => {
    expect(shouldLoadMore({ ...base, lastRenderedRow: undefined })).toBe(false)
  })
})
