import { afterEach, describe, expect, it, vi } from "vitest"
import { cleanup, fireEvent, render, screen, within } from "@testing-library/react"
import { FacetFilters } from "./facet-filters"
import type { FacetSelection } from "./facet-filters"
import type { CatalogFacets } from "@/generated/models"

afterEach(() => cleanup())

const facets: CatalogFacets = {
  expansionSets: [
    {
      id: "s1",
      code: "SV1",
      name: "Set One",
      seriesId: "series-1",
      available: true,
    },
    { id: "s2", code: "SV2", name: "Set Two", available: true },
  ],
  rarities: [
    { id: "r1", code: "C", name: "Common", available: true },
    { id: "r2", code: "SAR", name: "Special Art", available: false },
  ],
  categories: [{ value: "Pokémon", available: true }],
  tags: [{ value: "V", available: true }],
}
const series = [{ id: "series-1", name: "Scarlet" }]
const none: FacetSelection = { expansionSetId: [], rarityId: [], category: [], tag: [] }

function open(name: RegExp | string) {
  fireEvent.click(screen.getByRole("button", { name }))
}

function renderFilters(selected = none, onChange = vi.fn()) {
  render(<FacetFilters facets={facets} series={series} selected={selected} onChange={onChange} />)
  return onChange
}

describe("FacetFilters", () => {
  it("renders closed dropdowns with a count of selected values", () => {
    renderFilters({ ...none, rarityId: ["r1", "r2"] })

    expect(screen.queryByRole("checkbox")).toBeNull()
    expect(screen.getByRole("button", { name: /^Rarity\s*selected\s*2$/ })).toBeTruthy()
    expect(screen.getByRole("button", { name: "Tag" })).toBeTruthy()
  })

  it("groups Expansion Sets under their Series, with ungrouped sets apart", () => {
    renderFilters()
    open("Expansion Set")

    const scarlet = screen.getByRole("group", { name: "Scarlet" })
    expect(scarlet.textContent).toContain("Set One")
    expect(scarlet.textContent).not.toContain("Set Two")
    expect(screen.getByRole("group", { name: "Ungrouped" }).textContent).toContain("Set Two")
  })

  it("adds a value to the existing selection and stays open", () => {
    const onChange = renderFilters({ ...none, rarityId: ["r2"] })
    open(/^Rarity/)

    fireEvent.click(screen.getByLabelText("Common"))
    expect(onChange).toHaveBeenCalledWith("rarityId", ["r2", "r1"])
    expect(screen.getByLabelText("Common")).toBeTruthy()
  })

  it("narrows Expansion Set options with the search box", () => {
    renderFilters()
    open("Expansion Set")

    fireEvent.change(screen.getByLabelText("Search Expansion Set"), { target: { value: "two" } })
    expect(screen.queryByLabelText(/Set One/)).toBeNull()
    expect(screen.getByLabelText(/Set Two/)).toBeTruthy()
    expect(screen.queryByRole("group", { name: "Scarlet" })).toBeNull()
  })

  it("has no search box on the other dropdowns", () => {
    renderFilters()
    open("Rarity")

    expect(screen.queryByRole("searchbox")).toBeNull()
  })

  it("shows selected values as chips across filters and removes one on click", () => {
    const onChange = renderFilters({ ...none, rarityId: ["r1", "r2"], tag: ["V"] })

    const chips = within(screen.getByRole("group", { name: "Active filters" }))
    expect(chips.getAllByRole("button")).toHaveLength(3)
    fireEvent.click(chips.getByRole("button", { name: "Remove Common" }))
    expect(onChange).toHaveBeenCalledWith("rarityId", ["r2"])
  })

  it("keeps a selected but unavailable value checked so it can be unchecked", () => {
    const onChange = renderFilters({ ...none, rarityId: ["r2"] })
    open(/^Rarity/)

    const box = screen.getByRole("checkbox", { name: /^Special Art/ })
    expect(box).toHaveProperty("checked", true)
    fireEvent.click(box)
    expect(onChange).toHaveBeenCalledWith("rarityId", [])
  })

  it("still shows the selected values, checked, when facet data is missing", () => {
    // renderFilters would swap an explicit undefined for its default facets
    render(
      <FacetFilters
        facets={undefined}
        series={[]}
        selected={{ ...none, expansionSetId: ["s1"] }}
        onChange={vi.fn()}
      />
    )

    open(/^Expansion Set/)
    expect(screen.getByRole("checkbox", { name: "s1" })).toHaveProperty("checked", true)
  })

  it("flags unavailable options for screen readers", () => {
    renderFilters()
    open("Rarity")

    expect(screen.getByRole("checkbox", { name: /Special Art\s*\(unavailable\)/ })).toBeTruthy()
  })
})
