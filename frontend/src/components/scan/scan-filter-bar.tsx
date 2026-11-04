import { useState } from "react"
import { keepPreviousData } from "@tanstack/react-query"
import { Button } from "@/components/ui/button"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import { useListCatalogFacets } from "@/generated/endpoints/catalog/catalog"
import type { CatalogFacets } from "@/generated/models"
import {
  loadScanFilter,
  NO_SCAN_FILTER,
  saveScanFilter,
} from "@/lib/scan-filter"
import type { ScanFilter } from "@/lib/scan-filter"

/** The Scan Filter for this scan session, remembered on the device as it changes. */
export function useScanFilter() {
  const [filter, setFilter] = useState(loadScanFilter)
  const query = useListCatalogFacets(undefined, {
    query: { placeholderData: keepPreviousData },
  })
  const facets = query.data?.status === 200 ? query.data.data.data : undefined
  return {
    filter,
    facets,
    setFilter(next: ScanFilter) {
      setFilter(next)
      saveScanFilter(next)
    },
  }
}

const toggleClass = (selected: boolean) =>
  `rounded-full px-3 py-1 text-sm ${selected ? "bg-primary text-primary-foreground" : "bg-muted"}`

const toggle = (values: string[], value: string) =>
  values.includes(value)
    ? values.filter((one) => one !== value)
    : [...values, value]

/**
 * The "Limit search" row above the camera: the chosen Scan Filter as wrapping
 * chips, and a sheet with the Expansion Sets and Rarities to limit to. The sheet
 * applies when it closes.
 */
export function ScanFilterBar({
  filter,
  facets,
  onChange,
}: {
  filter: ScanFilter
  facets: CatalogFacets | undefined
  onChange: (filter: ScanFilter) => void
}) {
  const [open, setOpen] = useState(false)
  const [draft, setDraft] = useState(filter)
  const sets = facets?.expansionSets ?? []
  const rarities = facets?.rarities ?? []
  const close = () => {
    setOpen(false)
    onChange(draft)
  }
  const chips = [
    ...filter.expansionSetIds.map((id) => ({
      key: `set:${id}`,
      label: sets.find((set) => set.id === id)?.code ?? id,
    })),
    ...filter.rarityIds.map((id) => ({
      key: `rarity:${id}`,
      label: rarities.find((rarity) => rarity.id === id)?.name ?? id,
    })),
  ]

  return (
    <>
      <button
        type="button"
        aria-haspopup="dialog"
        aria-expanded={open}
        className="flex w-full flex-col gap-1 rounded-xl px-3 py-2 text-left outline-none hover:bg-muted/50 focus-visible:ring-3 focus-visible:ring-ring/30"
        onClick={() => {
          setDraft(filter)
          setOpen(true)
        }}
      >
        <span className="text-xs font-medium">Limit search</span>
        {chips.length === 0 ? (
          <span className="text-sm">All sets · All rarities</span>
        ) : (
          <span className="flex flex-wrap gap-1.5">
            {chips.map((chip) => (
              <span
                key={chip.key}
                className="rounded-3xl bg-secondary px-2.5 py-1 text-xs text-secondary-foreground"
              >
                {chip.label}
              </span>
            ))}
          </span>
        )}
      </button>

      <Dialog
        open={open}
        onOpenChange={(next) => {
          setOpen(next)
          if (!next) onChange(draft)
        }}
      >
        <DialogContent className="top-auto right-0 bottom-0 left-0 flex max-h-[85dvh] max-w-none translate-x-0 translate-y-0 flex-col rounded-b-none">
          <DialogHeader>
            <DialogTitle>Limit search</DialogTitle>
            <DialogDescription>
              Only match Cards in the chosen Expansion Sets and Rarities. Photos
              taken before a change keep the filter they were scanned with.
            </DialogDescription>
          </DialogHeader>

          <div className="flex min-h-0 flex-1 flex-col gap-4 overflow-y-auto">
            <div role="group" aria-label="Expansion Sets">
              <p className="mb-2 text-sm font-medium">Expansion Sets</p>
              <div className="flex flex-wrap gap-2">
                {sets.map((set) => (
                  <button
                    key={set.id}
                    type="button"
                    aria-pressed={draft.expansionSetIds.includes(set.id)}
                    className={toggleClass(
                      draft.expansionSetIds.includes(set.id)
                    )}
                    onClick={() =>
                      setDraft((one) => ({
                        ...one,
                        expansionSetIds: toggle(one.expansionSetIds, set.id),
                      }))
                    }
                  >
                    {set.name} ({set.code})
                  </button>
                ))}
              </div>
            </div>

            <div role="group" aria-label="Rarities">
              <p className="mb-2 text-sm font-medium">Rarities</p>
              <div className="flex flex-wrap gap-2">
                {rarities.map((rarity) => (
                  <button
                    key={rarity.id}
                    type="button"
                    aria-pressed={draft.rarityIds.includes(rarity.id)}
                    className={toggleClass(draft.rarityIds.includes(rarity.id))}
                    onClick={() =>
                      setDraft((one) => ({
                        ...one,
                        rarityIds: toggle(one.rarityIds, rarity.id),
                      }))
                    }
                  >
                    {rarity.name}
                  </button>
                ))}
              </div>
            </div>

            <div className="flex flex-wrap gap-2">
              <Button
                type="button"
                variant="outline"
                size="sm"
                onClick={() => setDraft(NO_SCAN_FILTER)}
              >
                Clear
              </Button>
              <Button type="button" size="sm" onClick={close}>
                Done
              </Button>
            </div>
          </div>
        </DialogContent>
      </Dialog>
    </>
  )
}
