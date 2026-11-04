import type { CatalogFacets, MatchScannedCardParams } from "@/generated/models"

/** The Expansion Sets and rarities a scan is limited to. Empty means no limit. */
export interface ScanFilter {
  expansionSetIds: string[]
  rarityIds: string[]
}

export const NO_SCAN_FILTER: ScanFilter = { expansionSetIds: [], rarityIds: [] }

const STORAGE_KEY = "cardstack.scan-filter"

const strings = (value: unknown): string[] =>
  Array.isArray(value) ? value.filter((one) => typeof one === "string") : []

/** The last Scan Filter chosen on this device, or no limit. */
export function loadScanFilter(): ScanFilter {
  try {
    const saved = JSON.parse(localStorage.getItem(STORAGE_KEY) ?? "null") as {
      expansionSetIds?: unknown
      rarityIds?: unknown
    } | null
    return {
      expansionSetIds: strings(saved?.expansionSetIds),
      rarityIds: strings(saved?.rarityIds),
    }
  } catch {
    return NO_SCAN_FILTER
  }
}

/** Remembers the Scan Filter on this device; the backend never stores it. */
export function saveScanFilter(filter: ScanFilter) {
  try {
    localStorage.setItem(STORAGE_KEY, JSON.stringify(filter))
  } catch {
    // Storage blocked: the filter still applies for this visit.
  }
}

export const hasScanFilter = (filter: ScanFilter) =>
  filter.expansionSetIds.length > 0 || filter.rarityIds.length > 0

/** The query parameters of a Scan Filter, or none when it is empty. */
export function scanFilterParams(filter: ScanFilter): MatchScannedCardParams {
  return {
    ...(filter.expansionSetIds.length > 0 && {
      expansionSetId: filter.expansionSetIds,
    }),
    ...(filter.rarityIds.length > 0 && { rarityId: filter.rarityIds }),
  }
}

/** What a photo was searched against, e.g. "Searched MA5, MA6 only". */
export function scanFilterNote(
  filter: ScanFilter,
  facets: CatalogFacets | undefined
): string | undefined {
  if (!hasScanFilter(filter)) return undefined
  const sets = filter.expansionSetIds.map(
    (id) => facets?.expansionSets?.find((set) => set.id === id)?.code ?? id
  )
  const rarities = filter.rarityIds.map(
    (id) => facets?.rarities?.find((rarity) => rarity.id === id)?.name ?? id
  )
  return `Searched ${[...sets, ...rarities].join(", ")} only`
}
