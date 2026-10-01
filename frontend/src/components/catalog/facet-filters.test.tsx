import { afterEach, describe, expect, it, vi } from "vitest"
import { cleanup, fireEvent, render, screen } from "@testing-library/react"
import { FacetFilters } from "./facet-filters"
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
const none = { expansionSetId: [], rarityId: [], category: [], tag: [] }

describe("FacetFilters", () => {
  it("groups Expansion Sets under their Series, with ungrouped sets apart", () => {
    render(
      <FacetFilters
        facets={facets}
        series={series}
        selected={none}
        onChange={vi.fn()}
      />
    )

    const scarlet = screen.getByRole("group", { name: "Scarlet" })
    expect(scarlet.textContent).toContain("Set One")
    expect(scarlet.textContent).not.toContain("Set Two")
    expect(
      screen.getByRole("group", { name: "Ungrouped" }).textContent
    ).toContain("Set Two")
  })

  it("adds a value to the existing selection when checked", () => {
    const onChange = vi.fn()
    render(
      <FacetFilters
        facets={facets}
        series={series}
        selected={{ ...none, rarityId: ["r2"] }}
        onChange={onChange}
      />
    )

    fireEvent.click(screen.getByLabelText("Common"))
    expect(onChange).toHaveBeenCalledWith("rarityId", ["r2", "r1"])
  })

  it("keeps a selected but unavailable value checked so it can be unchecked", () => {
    const onChange = vi.fn()
    render(
      <FacetFilters
        facets={facets}
        series={series}
        selected={{ ...none, rarityId: ["r2"] }}
        onChange={onChange}
      />
    )

    const box = screen.getByRole("checkbox", { name: "Special Art" })
    expect(box).toHaveProperty("checked", true)
    fireEvent.click(box)
    expect(onChange).toHaveBeenCalledWith("rarityId", [])
  })
})
