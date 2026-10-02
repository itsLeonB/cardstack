import { RiCloseLine } from "@remixicon/react"
import { MultiSelect } from "@/components/ui/multi-select"
import type {
  MultiSelectGroup,
  MultiSelectOption,
} from "@/components/ui/multi-select"
import type { CatalogFacets } from "@/generated/models"

export type FacetKey = "expansionSetId" | "rarityId" | "category" | "tag"
export type FacetSelection = Record<FacetKey, string[]>

type Option = MultiSelectOption & { seriesId?: string }

const valueOptions = (
  items: { value: string; available: boolean }[]
): Option[] =>
  items.map(({ value, available }) => ({ value, label: value, available }))

// Selected values must never vanish: if the facets call failed or hasn't
// loaded, show the URL's selection as checked options (labelled by raw value).
function withSelected(options: Option[], selected: string[]): Option[] {
  const known = new Set(options.map((option) => option.value))
  const missing = selected
    .filter((value) => !known.has(value))
    .map((value) => ({ value, label: value, available: true }))
  return [...options, ...missing]
}

interface FacetFiltersProps {
  facets: CatalogFacets | undefined
  series: { id: string; name: string }[]
  selected: FacetSelection
  onChange: (key: FacetKey, values: string[]) => void
}

/**
 * Dropdown multi-selects (with removable chips) for the faceted catalog
 * filters. Options come from the facets endpoint, which already keeps selected-but-unavailable values in the
 * list (available: false); they stay checked and enabled so they can be unchecked.
 */
export function FacetFilters({
  facets,
  series,
  selected,
  onChange,
}: FacetFiltersProps) {
  const sets = withSelected(
    (facets?.expansionSets ?? []).map((set) => ({
      value: set.id,
      label: `${set.name} (${set.code})`,
      available: set.available,
      seriesId: set.seriesId,
    })),
    selected.expansionSetId
  )
  const knownSeriesIds = new Set(series.map((oneSeries) => oneSeries.id))
  const setGroups: MultiSelectGroup[] = [
    ...series.map((oneSeries) => ({
      label: oneSeries.name,
      options: sets.filter((set) => set.seriesId === oneSeries.id),
    })),
    {
      label: "Ungrouped",
      options: sets.filter(
        (set) => !set.seriesId || !knownSeriesIds.has(set.seriesId)
      ),
    },
  ].filter((group) => group.options.length > 0)

  const flat = (key: FacetKey, options: Option[]): MultiSelectGroup[] => [
    { options: withSelected(options, selected[key]) },
  ]
  const filters: {
    key: FacetKey
    label: string
    groups: MultiSelectGroup[]
    searchable?: boolean
  }[] = [
    {
      key: "expansionSetId",
      label: "Expansion Set",
      groups: setGroups,
      searchable: true,
    },
    {
      key: "rarityId",
      label: "Rarity",
      groups: flat(
        "rarityId",
        (facets?.rarities ?? []).map((r) => ({
          value: r.id,
          label: r.name,
          available: r.available,
        }))
      ),
    },
    {
      key: "category",
      label: "Category",
      groups: flat("category", valueOptions(facets?.categories ?? [])),
    },
    {
      key: "tag",
      label: "Tag",
      groups: flat("tag", valueOptions(facets?.tags ?? [])),
    },
  ]
  const chips = filters.flatMap(({ key, groups }) =>
    selected[key].map((value) => ({
      key,
      value,
      label:
        groups.flatMap((g) => g.options).find((o) => o.value === value)
          ?.label ?? value,
    }))
  )

  return (
    <div className="flex flex-col gap-3">
      <div className="grid gap-2 sm:flex sm:flex-wrap">
        {filters.map(({ key, label, groups, searchable }) => (
          <MultiSelect
            key={key}
            label={label}
            groups={groups}
            searchable={searchable}
            selected={selected[key]}
            onChange={(values) => onChange(key, values)}
          />
        ))}
      </div>
      {chips.length > 0 && (
        <div
          role="group"
          aria-label="Active filters"
          className="flex flex-wrap gap-1.5"
        >
          {chips.map((chip) => (
            <button
              key={`${chip.key}:${chip.value}`}
              type="button"
              aria-label={`Remove ${chip.label}`}
              onClick={() =>
                onChange(
                  chip.key,
                  selected[chip.key].filter((value) => value !== chip.value)
                )
              }
              className="inline-flex items-center gap-1 rounded-3xl bg-secondary px-2.5 py-1 text-xs text-secondary-foreground outline-none hover:bg-secondary/80 focus-visible:ring-3 focus-visible:ring-ring/30"
            >
              {chip.label}
              <RiCloseLine aria-hidden className="size-3" />
            </button>
          ))}
        </div>
      )}
    </div>
  )
}
