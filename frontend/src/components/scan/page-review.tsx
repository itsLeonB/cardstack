import { Button } from "@/components/ui/button"
import type { Grid } from "@/lib/page-cut"
import { useObjectUrl } from "@/lib/use-page-scan"
import type { PageCell } from "@/lib/use-page-scan"

/** The cut cells in the Page's rows and columns, each showing how its match went. */
// The states a cell can be set to by hand; pressing a set one again goes back to the scan's result.
const MARKS = [
  { mark: "empty", text: "Empty" },
  { mark: "unidentified", text: "Unidentified" },
] as const

export function PageReview({
  cells,
  grid,
  changed,
  onRetry,
  onChoose,
  onMark,
}: {
  cells: PageCell[]
  grid: Grid
  /** Whether each cell differs from the Slot the Page holds there, in grid order. */
  changed: boolean[]
  onRetry: (index: number) => void
  onChoose: (index: number, cardId: string) => void
  onMark: (index: number, mark: PageCell["mark"]) => void
}) {
  return (
    <div
      className="grid gap-2"
      style={{ gridTemplateColumns: `repeat(${grid.columns}, minmax(0, 1fr))` }}
    >
      {cells.map((cell, index) => {
        const row = Math.floor(index / grid.columns) + 1
        const column = (index % grid.columns) + 1
        return (
          <CellReview
            key={index}
            label={`Row ${row}, column ${column}`}
            cell={cell}
            changed={changed[index]}
            onRetry={() => onRetry(index)}
            onChoose={(cardId) => onChoose(index, cardId)}
            onMark={(mark) => onMark(index, mark)}
          />
        )
      })}
    </div>
  )
}

function CellReview({
  label,
  cell,
  changed,
  onRetry,
  onChoose,
  onMark,
}: {
  label: string
  cell: PageCell
  changed: boolean
  onRetry: () => void
  onChoose: (cardId: string) => void
  onMark: (mark: PageCell["mark"]) => void
}) {
  const url = useObjectUrl(cell.photo)
  const { match } = cell
  return (
    <div
      role="group"
      aria-label={label}
      className={`flex min-w-0 flex-col gap-1 rounded-md border p-1 text-xs ${changed ? "ring-2 ring-primary" : ""}`}
    >
      <img
        src={url}
        alt=""
        className="aspect-[5/7] w-full rounded bg-muted object-cover"
      />
      {match.status === "queued" || match.status === "loading" ? (
        <p>Matching…</p>
      ) : match.status === "failed" ? (
        <div className="flex flex-col items-start gap-1">
          <p role="alert" className="text-destructive">
            Could not match this cell.
          </p>
          <Button size="sm" variant="outline" onClick={onRetry}>
            Retry
          </Button>
        </div>
      ) : match.candidates.length === 0 ? (
        <p>No good candidate</p>
      ) : match.confident ? (
        <p className="font-medium">{match.candidates[0].name}</p>
      ) : (
        <label className="flex flex-col gap-1">
          <span>Unsure: choose the card</span>
          <select
            value={cell.cardId}
            onChange={(event) => onChoose(event.currentTarget.value)}
            className="h-7 rounded-md border bg-background px-1"
          >
            {match.candidates.map((card) => (
              <option key={card.id} value={card.id}>
                {card.name} #{card.localId}
              </option>
            ))}
          </select>
        </label>
      )}
      {changed && <p className="font-medium">Changes this Slot</p>}
      <div className="flex flex-wrap gap-1">
        {MARKS.map(({ mark, text }) => (
          <Button
            key={mark}
            size="sm"
            variant={cell.mark === mark ? "default" : "outline"}
            aria-pressed={cell.mark === mark}
            onClick={() => onMark(cell.mark === mark ? undefined : mark)}
          >
            {text}
          </Button>
        ))}
      </div>
    </div>
  )
}
