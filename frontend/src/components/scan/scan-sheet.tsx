import { RiArrowDownSLine, RiArrowUpSLine } from "@remixicon/react"
import { Button } from "@/components/ui/button"
import { ScanRow } from "./scan-row"
import { useDraftCards } from "./use-draft-cards"
import type { Mark } from "@/lib/draft-addition"
import type { Scan } from "@/lib/use-draft-addition"
import type { CardSummary } from "@/generated/models"

/**
 * The Draft Addition as a bottom sheet. Collapsed it shows the header and the
 * newest Scan; expanded it lists every Scan, newest first, in a list that
 * scrolls while the header and the Review button stay put.
 */
export function ScanSheet({
  scans,
  seen,
  pendingCount,
  open,
  onOpenChange,
  reviewable,
  onRaise,
  onLower,
  onChoose,
  onMark,
  onRetry,
  onRemove,
  onDiscard,
  onReview,
}: {
  scans: Scan[]
  seen: Map<string, CardSummary>
  pendingCount: number
  open: boolean
  onOpenChange: (open: boolean) => void
  /** Whether "Review and add" can open: some Scan is resolved and none is pending. */
  reviewable: boolean
  onRaise: (key: string) => void
  onLower: (key: string) => void
  onChoose: (key: string, cardId: string, reprintPosition?: number) => void
  onMark: (key: string, mark: Mark) => void
  onRetry: (key: string) => void
  onRemove: (key: string) => void
  onDiscard: () => void
  onReview: () => void
}) {
  const cardIds = scans.flatMap((s) =>
    s.status === "resolved" && s.cardId ? [s.cardId] : []
  )
  const { cards, missing, retry } = useDraftCards(cardIds, seen)
  const shown = open ? scans : scans.slice(0, 1)

  return (
    <section
      aria-labelledby="draft-heading"
      className={`flex min-h-0 flex-col border-t bg-background ${open ? "flex-[4] rounded-none" : "flex-none rounded-t-2xl"}`}
    >
      <div className="flex shrink-0 flex-wrap items-center gap-2 px-3 py-2">
        <h2
          id="draft-heading"
          className="flex-1 font-heading text-base font-medium"
        >
          <button
            type="button"
            aria-expanded={open}
            aria-controls="scan-list"
            className="flex items-center gap-1 rounded text-left outline-none focus-visible:ring-3 focus-visible:ring-ring/30"
            onClick={() => onOpenChange(!open)}
          >
            {open ? (
              <RiArrowDownSLine aria-hidden />
            ) : (
              <RiArrowUpSLine aria-hidden />
            )}
            Draft ({scans.length} {scans.length === 1 ? "scan" : "scans"})
          </button>
        </h2>
        {pendingCount > 0 && (
          <span className="text-sm">{pendingCount} still matching</span>
        )}
        <Button size="sm" disabled={!reviewable} onClick={onReview}>
          Review and add
        </Button>
      </div>
      {scans.length === 0 ? (
        <p id="scan-list" className="px-3 pb-3 text-sm">
          Scanned cards will appear here.
        </p>
      ) : (
        <ul
          id="scan-list"
          className={`flex min-h-0 flex-col gap-2 overflow-y-auto px-3 pb-3 ${open ? "flex-1" : ""}`}
        >
          {shown.map((scan) => (
            <ScanRow
              key={scan.key}
              scan={scan}
              card={scan.cardId ? cards.get(scan.cardId) : undefined}
              seen={seen}
              lookup={missing}
              onRaise={() => onRaise(scan.key)}
              onLower={() => onLower(scan.key)}
              onChoose={(cardId, reprintPosition) =>
                onChoose(scan.key, cardId, reprintPosition)
              }
              onMark={(mark) => onMark(scan.key, mark)}
              onRetry={() => onRetry(scan.key)}
              onRetryLookup={retry}
              onRemove={() => onRemove(scan.key)}
            />
          ))}
        </ul>
      )}
      {open && scans.length > 0 && (
        <div className="shrink-0 px-3 pb-3">
          <Button variant="outline" size="sm" onClick={onDiscard}>
            Discard draft
          </Button>
        </div>
      )}
    </section>
  )
}
