import { afterEach, describe, expect, it, vi } from "vitest"
import { render, screen, cleanup } from "@testing-library/react"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { PresetSheet } from "./preset-sheet"
import type {
  ExpansionSetFacetOption,
  RarityFacetOption,
  StringFacetOption,
} from "@/generated/models"

const expansionSets: ExpansionSetFacetOption[] = [
  { id: "set-1", code: "S1", name: "Set 1", imageUrl: "", available: true, seriesId: null },
  { id: "set-2", code: "S2", name: "Set 2", imageUrl: "", available: true, seriesId: null },
]

const rarities: RarityFacetOption[] = [
  { id: "rarity-1", code: "C", name: "Common", available: true },
]

const categories: StringFacetOption[] = [
  { value: "Pokémon", available: true },
  { value: "Trainer", available: true },
]

const tags: StringFacetOption[] = []

function renderWithQuery(ui: React.ReactElement) {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: { retry: false },
      mutations: { retry: false },
    },
  })
  return render(<QueryClientProvider client={queryClient}>{ui}</QueryClientProvider>)
}

describe("PresetSheet", () => {
  afterEach(cleanup)

  it("renders with the correct heading", () => {
    renderWithQuery(
      <PresetSheet
        open={true}
        onOpenChange={vi.fn()}
        expansionSets={expansionSets}
        rarities={rarities}
        categories={categories}
        tags={tags}
      />
    )

    expect(screen.getByRole("heading", { name: "Fill from sets" })).toBeDefined()
  })

  it("shows expansion sets as wrapping chips", () => {
    renderWithQuery(
      <PresetSheet
        open={true}
        onOpenChange={vi.fn()}
        expansionSets={expansionSets}
        rarities={rarities}
        categories={categories}
        tags={tags}
      />
    )

    expect(screen.getByRole("button", { name: "Set 1" })).toBeDefined()
    expect(screen.getByRole("button", { name: "Set 2" })).toBeDefined()
  })

  it("shows optional category filters", () => {
    renderWithQuery(
      <PresetSheet
        open={true}
        onOpenChange={vi.fn()}
        expansionSets={expansionSets}
        rarities={rarities}
        categories={categories}
        tags={tags}
      />
    )

    expect(screen.getByRole("button", { name: "Pokémon" })).toBeDefined()
    expect(screen.getByRole("button", { name: "Trainer" })).toBeDefined()
  })

  it("has a quantity stepper that starts at 1", () => {
    renderWithQuery(
      <PresetSheet
        open={true}
        onOpenChange={vi.fn()}
        expansionSets={expansionSets}
        rarities={rarities}
        categories={categories}
        tags={tags}
      />
    )

    expect(screen.getByText("Quantity per Card")).toBeDefined()
    expect(screen.getByText("1")).toBeDefined()
  })
})
