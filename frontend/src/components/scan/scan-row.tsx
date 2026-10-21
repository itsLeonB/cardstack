import {
  RiAddLine,
  RiCloseLine,
  RiDeleteBinLine,
  RiLoader4Line,
  RiSubtractLine,
} from "@remixicon/react"
import { Button } from "@/components/ui/button"
import { CandidatePicker, MARKS } from "./candidate-picker"
import { CardThumb, PhotoThumb } from "./card-thumb"
import { MISSING_NAME } from "./use-draft-cards"
import type { Missing } from "./use-draft-cards"
import type { Mark } from "@/lib/draft-addition"
import type { Scan } from "@/lib/use-draft-addition"
import type { CardSummary } from "@/generated/models"

/**
 * One Scan in the tray. Resolved Scans show their card with quantity controls
 * and a dropdown of candidates and marks. A Scan with no card shows "No card
 * found." with Retry. Pending Scans show the user's photo under a spinner,
 * failed ones under a red cross with Retry.
 */
export function ScanRow({
  scan,
  card,
  seen,
  lookup,
  onRaise,
  onLower,
  onChoose,
  onMark,
  onRetry,
  onRetryLookup,
  onRemove,
}: {
  scan: Scan
  card?: CardSummary
  seen: Map<string, CardSummary>
  /** Why the Card is still absent, when it is; from the sheet's lookup. */
  lookup: Missing
  onRaise: () => void
  onLower: () => void
  onChoose: (cardId: string, reprintPosition?: number) => void
  onMark: (mark: Mark) => void
  onRetry: () => void
  onRetryLookup: () => void
  onRemove: () => void
}) {
  const label = card?.name ?? "scan"
  const resolved = scan.status === "resolved"
  const face = resolved ? (
    <>
      <CardThumb card={card} />
      <span className="flex min-w-0 flex-1 flex-col text-left">
        <span className="truncate text-sm font-medium">
          {scan.cardId
            ? (card?.name ?? MISSING_NAME[lookup])
            : "No card found."}
        </span>
        {card && (
          <span className="truncate text-xs">
            {card.expansionSet.name} · No. {card.localId}
          </span>
        )}
        {scan.mark && (
          <span className="truncate text-xs">
            Marked: {MARKS[scan.mark].toLowerCase()}
          </span>
        )}
      </span>
    </>
  ) : (
    <>
      <PhotoThumb photo={scan.photo!}>
        {scan.status === "failed" ? (
          <span className="absolute inset-0 grid place-items-center bg-black/50">
            <RiCloseLine className="size-6 text-red-400" aria-hidden />
          </span>
        ) : (
          <span className="absolute inset-0 grid place-items-center bg-black/40">
            <RiLoader4Line
              className="size-5 animate-spin text-white"
              aria-hidden
            />
          </span>
        )}
      </PhotoThumb>
      <span className="flex min-w-0 flex-1 flex-col text-left">
        <span className="text-sm font-medium">
          {scan.status === "failed"
            ? (scan.reason ?? "Could not match this photo.")
            : scan.status === "loading"
              ? "Matching…"
              : "Queued"}
        </span>
      </span>
    </>
  )

  return (
    <li className="flex items-center gap-2 rounded-xl border p-2">
      {resolved ? (
        <CandidatePicker
          scan={scan}
          seen={seen}
          label={label}
          onChoose={onChoose}
          onMark={onMark}
        >
          {face}
        </CandidatePicker>
      ) : (
        <div className="flex min-w-0 flex-1 items-center gap-2">{face}</div>
      )}
      {resolved && scan.cardId && (
        <>
          <Button
            variant="outline"
            size="icon-sm"
            aria-label={`Lower quantity of ${label}`}
            disabled={scan.quantity <= 1}
            onClick={onLower}
          >
            <RiSubtractLine />
          </Button>
          <span className="w-8 text-center tabular-nums">{scan.quantity}</span>
          <Button
            variant="outline"
            size="icon-sm"
            aria-label={`Raise quantity of ${label}`}
            onClick={onRaise}
          >
            <RiAddLine />
          </Button>
        </>
      )}
      {(scan.status === "failed" || (resolved && !scan.cardId)) && (
        <Button variant="outline" size="xs" onClick={onRetry}>
          Retry
        </Button>
      )}
      {resolved && scan.cardId && !card && lookup === "failed" && (
        <Button
          variant="outline"
          size="xs"
          aria-label={`Retry loading ${label}`}
          onClick={onRetryLookup}
        >
          Retry
        </Button>
      )}
      <Button
        variant="ghost"
        size="icon-sm"
        aria-label={`Remove ${label}`}
        onClick={onRemove}
      >
        <RiDeleteBinLine />
      </Button>
    </li>
  )
}
