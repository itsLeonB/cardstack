import type {
  ExpansionSetSummary,
  SeriesBrowseResult,
} from "@/generated/models"

// Same look as `Input`; a native select keeps the keyboard and screen-reader
// behaviour for free.
export const SELECT_CLASS =
  "h-9 w-full min-w-0 rounded-3xl border border-transparent bg-input/50 px-3 py-1 text-base outline-none focus-visible:border-ring focus-visible:ring-3 focus-visible:ring-ring/30 disabled:opacity-50 md:text-sm"

/** Every Expansion Set of the browse result, those under a Series first. */
export function flattenSets(
  browse: SeriesBrowseResult | null | undefined
): ExpansionSetSummary[] {
  return [
    ...(browse?.series ?? []).flatMap((s) => s.expansionSets ?? []),
    ...(browse?.ungroupedExpansionSets ?? []),
  ]
}
