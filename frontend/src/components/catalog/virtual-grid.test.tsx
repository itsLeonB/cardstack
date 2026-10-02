import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"
import { act, cleanup, render, screen } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { VirtualGrid } from "./virtual-grid"
import { stubGridLayout } from "@/test-grid-layout"

afterEach(() => {
  cleanup()
  vi.restoreAllMocks()
  vi.unstubAllGlobals()
})

let layout: ReturnType<typeof stubGridLayout>

beforeEach(() => {
  layout = stubGridLayout()
})

const items = (count: number, prefix = "item") =>
  Array.from({ length: count }, (_, index) => `${prefix}-${index}`)

function renderGrid(count: number, canLoadMore = true, totalCount?: number) {
  const onLoadMore = vi.fn()
  render(
    <VirtualGrid
      items={items(count)}
      getKey={(item) => item}
      renderItem={(item) => <button type="button">{item}</button>}
      totalCount={totalCount}
      canLoadMore={canLoadMore}
      onLoadMore={onLoadMore}
    />
  )
  return onLoadMore
}

// Holds animation frames until the test runs them one at a time: the
// virtualizer reschedules its own frames, so running them all never ends.
function queueFrames() {
  const frames: FrameRequestCallback[] = []
  vi.spyOn(window, "requestAnimationFrame").mockImplementation((run) =>
    frames.push(run)
  )
  return frames
}

// SAFETY: focus only moves to HTML elements in these tests.
const blurActive = () => (document.activeElement as HTMLElement).blur()

// Rows are the presentational wrappers carrying `data-index`.
const rowTiles = (index: number) =>
  Array.from(document.querySelector(`[data-index="${index}"]`)?.children ?? [])

describe("VirtualGrid", () => {
  it("renders only a window of rows, with one tile per column", () => {
    renderGrid(2000)

    const tiles = screen.getAllByRole("listitem")
    expect(tiles.length).toBeGreaterThan(0)
    expect(tiles.length).toBeLessThan(60)
    expect(rowTiles(0)).toHaveLength(5)
    expect(screen.getByText("item-0")).toBeTruthy()
    expect(screen.queryByText("item-1999")).toBeNull()
  })

  it("re-chunks the rows when the container width changes", () => {
    renderGrid(2000)
    expect(rowTiles(0)).toHaveLength(5)

    act(() => layout.resize(600))

    expect(rowTiles(0)).toHaveLength(3)
    expect(screen.getByText("item-0")).toBeTruthy()
  })

  it("asks for more once the last row is rendered", () => {
    expect(renderGrid(20)).toHaveBeenCalled()
  })

  it("does not ask while rows below the window are unrendered", () => {
    expect(renderGrid(2000)).not.toHaveBeenCalled()
  })

  it("never asks while loading is not allowed", () => {
    expect(renderGrid(20, false)).not.toHaveBeenCalled()
  })

  it("exposes each tile's position and the total through the list markup", () => {
    renderGrid(2000, false, 12_000)

    expect(screen.getByRole("list")).toBeTruthy()
    // Rows are presentational: only the tiles are list items.
    const rowWrapper = document.querySelector('[data-index="0"]')
    expect(rowWrapper?.getAttribute("role")).toBe("presentation")
    expect(
      rowTiles(0).map((tile) => tile.getAttribute("aria-posinset"))
    ).toEqual(["1", "2", "3", "4", "5"])
    expect(rowTiles(1)[0]!.getAttribute("aria-posinset")).toBe("6")
    expect(
      screen
        .getAllByRole("listitem")
        .every((tile) => tile.getAttribute("aria-setsize") === "12000")
    ).toBe(true)
  })

  it("defaults the set size to the loaded count", () => {
    renderGrid(20, false)
    expect(rowTiles(0)[0]!.getAttribute("aria-setsize")).toBe("20")
  })

  it("keeps the focused row rendered after scrolling away, then lets it go", async () => {
    renderGrid(2000, false)
    const focused = screen.getByRole("button", { name: "item-0" })
    await userEvent.click(focused)
    expect(document.activeElement).toBe(focused)

    act(() => layout.scrollTo(20_000))
    expect(screen.queryByRole("button", { name: "item-1999" })).toBeNull()
    expect(document.activeElement).toBe(focused)
    expect(document.contains(focused)).toBe(true)

    // Focus leaving the grid releases the row, so it unmounts as usual.
    act(() => focused.blur())
    expect(document.contains(focused)).toBe(false)
  })

  it("does not read a pinned row far below as having reached the end", async () => {
    const onLoadMore = vi.fn()
    const grid = (canLoadMore: boolean) => (
      <VirtualGrid
        items={items(2000)}
        getKey={(item) => item}
        renderItem={(item) => <button type="button">{item}</button>}
        canLoadMore={canLoadMore}
        onLoadMore={onLoadMore}
      />
    )
    const { rerender } = render(grid(false))
    act(() => layout.scrollTo(2000 * 100))
    await userEvent.click(screen.getByRole("button", { name: "item-1999" }))

    // Back at the top the last row stays rendered (it has focus) but is nowhere
    // near the viewport, so allowing a load must not fire one.
    act(() => layout.scrollTo(0))
    rerender(grid(true))

    expect(screen.getByRole("button", { name: "item-1999" })).toBeTruthy()
    expect(onLoadMore).not.toHaveBeenCalled()
  })

  it("re-anchors on the card at the top when the column count changes", () => {
    const frames = queueFrames()
    renderGrid(2000, false)
    // 416px per row: row 9 (cards 45 to 49) is at the top at 4000px.
    act(() => layout.scrollTo(4000))
    const scrollTo = vi.mocked(window.scrollTo)
    scrollTo.mockClear()

    act(() => layout.resize(600))
    // Deferred: nothing scrolls in the commit that changed the columns.
    expect(scrollTo).not.toHaveBeenCalled()
    act(() => frames.shift()?.(0))

    // 3 columns put card 45 in row 15. Rows there are unmeasured, so each is
    // the 415px estimate plus the 16px gap; the 56px sticky header stays clear.
    expect(scrollTo).toHaveBeenCalledWith(
      expect.objectContaining({ top: 15 * 431 - 56 })
    )
  })

  it("does not scroll on the first column settle when still at the top", () => {
    const frames = queueFrames()
    renderGrid(2000, false)
    vi.mocked(window.scrollTo).mockClear()

    act(() => layout.resize(600))
    act(() => frames.shift()?.(0))

    expect(window.scrollTo).not.toHaveBeenCalled()
  })

  it("pauses scroll-triggered loading while a tile has keyboard focus", async () => {
    const onLoadMore = vi.fn()
    const grid = (canLoadMore: boolean) => (
      <VirtualGrid
        items={items(20, "keyboard")}
        getKey={(item) => item}
        renderItem={(item) => <button type="button">{item}</button>}
        canLoadMore={canLoadMore}
        onLoadMore={onLoadMore}
      />
    )
    const { rerender } = render(grid(false))
    await userEvent.tab()
    expect(document.activeElement?.textContent).toBe("keyboard-0")

    rerender(grid(true))
    expect(onLoadMore).not.toHaveBeenCalled()

    // Tabbing out of the grid hands loading back to scrolling.
    act(() => blurActive())
    expect(onLoadMore).toHaveBeenCalledTimes(1)
  })

  it("survives a column change while a late row has focus", async () => {
    vi.stubGlobal("innerHeight", 5000)
    layout.resize(600)
    renderGrid(30, false)
    expect(rowTiles(0)).toHaveLength(3)
    await userEvent.click(screen.getByRole("button", { name: "item-29" }))

    // 10 rows of 3 become 6 of 5: the pinned row 9 no longer exists.
    expect(() => act(() => layout.resize(1000))).not.toThrow()

    expect(rowTiles(0)).toHaveLength(5)
    expect(screen.getByRole("button", { name: "item-0" })).toBeTruthy()
  })
})
