import { useState } from "react"
import { RiSearchLine } from "@remixicon/react"
import { FacetFilters } from "@/components/catalog/facet-filters"
import type { FacetKey } from "@/components/catalog/facet-filters"
import { Button } from "@/components/ui/button"
import {
  Field,
  FieldDescription,
  FieldGroup,
  FieldLabel,
  FieldLegend,
  FieldSet,
} from "@/components/ui/field"
import { Input } from "@/components/ui/input"
import type { CatalogFacets } from "@/generated/models"
import { hasActiveFilters } from "@/lib/catalog-search"
import type { CatalogFilters } from "@/lib/catalog-search"

interface StatusOption {
  value: string
  label: string
}

interface CatalogFilterPanelProps {
  search: CatalogFilters
  facets: CatalogFacets | undefined
  series: { id: string; name: string }[]
  /** Called with the changed filters; the caller resets the page. */
  onChange: (patch: Partial<CatalogFilters>) => void
  onClear: () => void
  /** A Guest: the rarity, category and tag filters are disabled with a sign-in prompt. */
  guest?: boolean
  /** Path (with search) the sign-in prompt returns to. */
  signInRedirect?: string
  /** A derived-status filter owned by the page (the Wishlist), shown above the facets. */
  statusFilter?: {
    value: string
    options: StatusOption[]
    onChange: (status: string) => void
  }
}

/** Name and card-number search plus the faceted filters, shared by the catalog and Collection pages. */
export function CatalogFilterPanel({
  search,
  facets,
  series,
  onChange,
  onClear,
  guest = false,
  signInRedirect,
  statusFilter,
}: CatalogFilterPanelProps) {
  const [nameInput, setNameInput] = useState(search.name ?? "")
  const [localIdInput, setLocalIdInput] = useState(search.localId ?? "")
  // Browser back/forward changes the URL from outside; resync so Search can't write stale text back.
  const [synced, setSynced] = useState({
    name: search.name,
    localId: search.localId,
  })
  if (synced.name !== search.name || synced.localId !== search.localId) {
    setSynced({ name: search.name, localId: search.localId })
    setNameInput(search.name ?? "")
    setLocalIdInput(search.localId ?? "")
  }

  return (
    <form
      onSubmit={(event) => {
        event.preventDefault()
        onChange({
          name: nameInput.trim() || undefined,
          localId: localIdInput.trim() || undefined,
        })
      }}
    >
      <FieldGroup>
        <div className="grid gap-4 sm:grid-cols-[1fr_auto]">
          <Field>
            <FieldLabel htmlFor="catalog-search-name">Card name</FieldLabel>
            <Input
              id="catalog-search-name"
              type="search"
              placeholder="e.g. Pikachu"
              value={nameInput}
              onChange={(event) => setNameInput(event.target.value)}
            />
          </Field>
          <Field className="sm:w-40">
            <FieldLabel htmlFor="catalog-search-local-id">
              Card number
            </FieldLabel>
            <Input
              id="catalog-search-local-id"
              placeholder="e.g. 048"
              value={localIdInput}
              onChange={(event) => setLocalIdInput(event.target.value)}
            />
            <FieldDescription>
              Pairs with an Expansion Set below.
            </FieldDescription>
          </Field>
        </div>

        {statusFilter && (
          <FieldSet>
            <FieldLegend>Status</FieldLegend>
            <div className="flex flex-wrap gap-2">
              {statusFilter.options.map((option) => (
                <Button
                  key={option.value}
                  type="button"
                  size="sm"
                  variant={
                    statusFilter.value === option.value ? "default" : "outline"
                  }
                  aria-pressed={statusFilter.value === option.value}
                  onClick={() => statusFilter.onChange(option.value)}
                >
                  {option.label}
                </Button>
              ))}
            </div>
          </FieldSet>
        )}

        <FacetFilters
          facets={facets}
          series={series}
          selected={{
            expansionSetId: search.expansionSetId ?? [],
            rarityId: search.rarityId ?? [],
            category: search.category ?? [],
            tag: search.tag ?? [],
          }}
          onChange={(key: FacetKey, values: string[]) =>
            onChange({ [key]: values.length > 0 ? values : undefined })
          }
          locked={guest}
          signInRedirect={signInRedirect}
        />

        <div className="flex flex-wrap items-center gap-3">
          <Button type="submit">
            <RiSearchLine data-icon="inline-start" />
            Search
          </Button>
          {hasActiveFilters(search) && (
            <Button
              type="button"
              variant="ghost"
              onClick={() => {
                setNameInput("")
                setLocalIdInput("")
                onClear()
              }}
            >
              Clear filters
            </Button>
          )}
        </div>
      </FieldGroup>
    </form>
  )
}
