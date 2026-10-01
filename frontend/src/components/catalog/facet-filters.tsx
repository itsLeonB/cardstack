import type { CatalogFacets } from "@/generated/models"

export type FacetKey = "expansionSetId" | "rarityId" | "category" | "tag"
export type FacetSelection = Record<FacetKey, string[]>

interface Option {
  value: string
  label: string
  available: boolean
  seriesId?: string
}

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

function OptionGroup({
  legend,
  options,
  selected,
  onChange,
  nested,
}: {
  legend: string
  options: Option[]
  selected: string[]
  onChange: (values: string[]) => void
  nested?: boolean
}) {
  if (options.length === 0) return null
  return (
    <fieldset className="flex min-w-0 flex-col gap-1.5">
      <legend
        className={
          nested ? "text-xs text-muted-foreground" : "text-sm font-medium"
        }
      >
        {legend}
      </legend>
      {options.map((option) => {
        const checked = selected.includes(option.value)
        return (
          <label
            key={option.value}
            className={`flex items-center gap-2 text-sm ${option.available ? "" : "text-muted-foreground"}`}
          >
            <input
              type="checkbox"
              checked={checked}
              onChange={() =>
                onChange(
                  checked
                    ? selected.filter((value) => value !== option.value)
                    : [...selected, option.value]
                )
              }
            />
            {option.label}
            {!option.available && <span className="sr-only"> (unavailable)</span>}
          </label>
        )
      })}
    </fieldset>
  )
}

/**
 * Checkbox groups for the faceted catalog filters. Options come from the
 * facets endpoint, which already keeps selected-but-unavailable values in the
 * list (available: false); they stay checked and enabled so they can be unchecked.
 */
export function FacetFilters({ facets, series, selected, onChange }: FacetFiltersProps) {
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
  const ungrouped = sets.filter((set) => !set.seriesId || !knownSeriesIds.has(set.seriesId))
  const setHandler = (key: FacetKey) => (values: string[]) => onChange(key, values)

  return (
    <div className="grid gap-6 sm:grid-cols-2 lg:grid-cols-4">
      <fieldset className="flex min-w-0 flex-col gap-3">
        <legend className="text-sm font-medium">Expansion Sets</legend>
        {series.map((oneSeries) => (
          <OptionGroup
            key={oneSeries.id}
            nested
            legend={oneSeries.name}
            options={sets.filter((set) => set.seriesId === oneSeries.id)}
            selected={selected.expansionSetId}
            onChange={setHandler("expansionSetId")}
          />
        ))}
        <OptionGroup
          nested
          legend="Ungrouped"
          options={ungrouped}
          selected={selected.expansionSetId}
          onChange={setHandler("expansionSetId")}
        />
      </fieldset>
      <OptionGroup
        legend="Rarity"
        options={withSelected(
          (facets?.rarities ?? []).map((rarity) => ({
            value: rarity.id,
            label: rarity.name,
            available: rarity.available,
          })),
          selected.rarityId
        )}
        selected={selected.rarityId}
        onChange={setHandler("rarityId")}
      />
      <OptionGroup
        legend="Category"
        options={withSelected(
          (facets?.categories ?? []).map((c) => ({ value: c.value, label: c.value, available: c.available })),
          selected.category
        )}
        selected={selected.category}
        onChange={setHandler("category")}
      />
      <OptionGroup
        legend="Tag"
        options={withSelected(
          (facets?.tags ?? []).map((t) => ({ value: t.value, label: t.value, available: t.available })),
          selected.tag
        )}
        selected={selected.tag}
        onChange={setHandler("tag")}
      />
    </div>
  )
}
