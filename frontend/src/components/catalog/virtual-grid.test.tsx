import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"
import { act, cleanup, render, screen } from "@testing-library/react"
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

const items = (count: number) =>
  Array.from({ length: count }, (_, index) => `item-${index}`)

function renderGrid(count: number, canLoadMore = true) {
  const onLoadMore = vi.fn()
  render(
    <VirtualGrid
      items={items(count)}
      getKey={(item) => item}
      renderItem={(item) => <span>{item}</span>}
      canLoadMore={canLoadMore}
      onLoadMore={onLoadMore}
    />
  )
  return onLoadMore
}

describe("VirtualGrid", () => {
  it("renders only a window of rows, with one tile per column", () => {
    renderGrid(2000)

    const tiles = screen.getAllByRole("listitem")
    expect(tiles.length).toBeGreaterThan(0)
    expect(tiles.length).toBeLessThan(60)
    expect(screen.getAllByRole("list")[0]!.children).toHaveLength(5)
    expect(screen.getByText("item-0")).toBeTruthy()
    expect(screen.queryByText("item-1999")).toBeNull()
  })

  it("re-chunks the rows when the container width changes", () => {
    renderGrid(2000)
    expect(screen.getAllByRole("list")[0]!.children).toHaveLength(5)

    act(() => layout.resize(600))

    expect(screen.getAllByRole("list")[0]!.children).toHaveLength(3)
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
})
