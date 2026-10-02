import { afterEach, describe, expect, it, vi } from "vitest"
import { cleanup, render, screen } from "@testing-library/react"
import { CatalogFilterPanel } from "./filter-panel"
import type { CatalogFilters } from "@/lib/catalog-search"

afterEach(() => cleanup())

function value(label: string) {
  const input = screen.getByLabelText(label)
  if (!(input instanceof HTMLInputElement))
    throw new Error(`${label} is not an input`)
  return input.value
}

function panel(search: CatalogFilters) {
  return (
    <CatalogFilterPanel
      search={search}
      facets={undefined}
      series={[]}
      onChange={vi.fn()}
      onClear={vi.fn()}
    />
  )
}

describe("CatalogFilterPanel", () => {
  it("resyncs the text inputs when the URL changes from outside the panel", () => {
    const { rerender } = render(panel({ name: "Pika", localId: "001" }))
    expect(value("Card name")).toBe("Pika")

    rerender(panel({ name: "Char" }))

    expect(value("Card name")).toBe("Char")
    expect(value("Card number")).toBe("")
  })
})
