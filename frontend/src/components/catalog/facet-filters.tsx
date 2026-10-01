import type { CatalogFacets } from "@/generated/models"

export type FacetKey = "expansionSetId" | "rarityId" | "category" | "tag"
export type FacetSelection = Record<FacetKey, string[]>

interface Option {
  value: string
  label: string
  available: boolean
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
export function FacetFilters({
  facets,
  series,
  selected,
  onChange,
}: FacetFiltersProps) {
  const sets = facets?.expansionSets ?? []
  const toOption = (set: (typeof sets)[number]): Option => ({
    value: set.id,
    label: `${set.name} (${set.code})`,
    available: set.available,
  })
  const knownSeriesIds = new Set(series.map((oneSeries) => oneSeries.id))
  const ungrouped = sets.filter(
    (set) => !set.seriesId || !knownSeriesIds.has(set.seriesId)
  )
  const setHandler = (key: FacetKey) => (values: string[]) =>
    onChange(key, values)

  return (
    <div className="grid gap-6 sm:grid-cols-2 lg:grid-cols-4">
      <div className="flex flex-col gap-3">
        <h2 className="text-sm font-medium">Expansion Sets</h2>
        {series.map((oneSeries) => (
          <OptionGroup
            key={oneSeries.id}
            nested
            legend={oneSeries.name}
            options={sets
              .filter((set) => set.seriesId === oneSeries.id)
              .map(toOption)}
            selected={selected.expansionSetId}
            onChange={setHandler("expansionSetId")}
          />
        ))}
        <OptionGroup
          nested
          legend="Ungrouped"
          options={ungrouped.map(toOption)}
          selected={selected.expansionSetId}
          onChange={setHandler("expansionSetId")}
        />
      </div>
      <OptionGroup
        legend="Rarity"
        options={(facets?.rarities ?? []).map((rarity) => ({
          value: rarity.id,
          label: rarity.name,
          available: rarity.available,
        }))}
        selected={selected.rarityId}
        onChange={setHandler("rarityId")}
      />
      <OptionGroup
        legend="Category"
        options={(facets?.categories ?? []).map((c) => ({
          value: c.value,
          label: c.value,
          available: c.available,
        }))}
        selected={selected.category}
        onChange={setHandler("category")}
      />
      <OptionGroup
        legend="Tag"
        options={(facets?.tags ?? []).map((t) => ({
          value: t.value,
          label: t.value,
          available: t.available,
        }))}
        selected={selected.tag}
        onChange={setHandler("tag")}
      />
    </div>
  )
}
