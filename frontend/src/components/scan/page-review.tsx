import { RiArrowDownSLine } from "@remixicon/react"
import { Button } from "@/components/ui/button"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuLabel,
  DropdownMenuRadioGroup,
  DropdownMenuRadioItem,
  DropdownMenuSeparator,
  DropdownMenuSub,
  DropdownMenuSubContent,
  DropdownMenuSubTrigger,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import { cardCode } from "@/lib/binder"
import { cellPosition } from "@/lib/page-cut"
import type { Grid } from "@/lib/page-cut"
import { cellMark, useObjectUrl } from "@/lib/use-page-scan"
import { OtherPrints } from "./other-prints"
import type { PageCell } from "@/lib/use-page-scan"
import type { CardSummary } from "@/generated/models"

type Mark = NonNullable<PageCell["mark"]>

const MARKS: { mark: Mark; text: string }[] = [
  { mark: "empty", text: "Empty" },
  { mark: "unidentified", text: "Unidentified" },
]

interface Choice {
  value: string
  card?: CardSummary
}

const isMark = (value: string): value is Mark =>
  MARKS.some(({ mark }) => mark === value)

/** What a cell is set to: a hand mark, else the chosen candidate; no candidate is Unidentified. */
function choiceOf(cell: PageCell): Choice {
  const mark = cellMark(cell)
  if (mark) return { value: mark }
  if (cell.reprintCard)
    return { value: cell.reprintCard.id, card: cell.reprintCard }
  const { match } = cell
  if (match.status === "matched") {
    const card =
      match.candidates.find((c) => c.id === cell.cardId) ?? match.candidates[0]
    if (card) return { value: card.id, card }
  }
  return { value: "unidentified" }
}

/** The cut cells in the Page's rows and columns, each showing how its match went. */
export function PageReview({
  cells,
  grid,
  changed,
  onRetry,
  onChoose,
  onReprint,
  onMark,
}: {
  cells: PageCell[]
  grid: Grid
  /** Whether each cell differs from the Slot the Page holds there, in grid order. */
  changed: boolean[]
  onRetry: (index: number) => void
  onChoose: (index: number, cardId: string) => void
  onReprint: (index: number, card: CardSummary, position: number) => void
  onMark: (index: number, mark: PageCell["mark"]) => void
}) {
  return (
    <div
      className="grid gap-3"
      style={{ gridTemplateColumns: `repeat(${grid.columns}, minmax(0, 1fr))` }}
    >
      {cells.map((cell, index) => {
        const { row, column } = cellPosition(index, grid.columns)
        return (
          <CellReview
            key={index}
            label={`Row ${row}, column ${column}`}
            cell={cell}
            changed={changed[index]}
            onRetry={() => onRetry(index)}
            onChoose={(cardId) => onChoose(index, cardId)}
            onReprint={(card, position) => onReprint(index, card, position)}
            onMark={(mark) => onMark(index, mark)}
          />
        )
      })}
    </div>
  )
}

// Dark and translucent so white text keeps 4.5:1 over bright card art; the
// same height in every Layout so a 6 by 6 Page reads like a 2 by 2.
const PILL =
  "flex h-11 min-w-0 flex-1 items-center gap-1 rounded-xl bg-black/80 px-2 text-left text-xs leading-tight text-white"

function CellReview({
  label,
  cell,
  changed,
  onRetry,
  onChoose,
  onReprint,
  onMark,
}: {
  label: string
  cell: PageCell
  changed: boolean
  onRetry: () => void
  onChoose: (cardId: string) => void
  onReprint: (card: CardSummary, position: number) => void
  onMark: (mark: PageCell["mark"]) => void
}) {
  const url = useObjectUrl(cell.photo)
  const { match } = cell
  const loading = match.status === "queued" || match.status === "loading"
  const failed = match.status === "failed"
  const choice = choiceOf(cell)
  const candidates = match.status === "matched" ? match.candidates : []
  // A Detection the matcher found empty has no Cards to find Reprints of.
  const offersReprints =
    candidates.length > 0 && match.status === "matched" && !match.empty
  const pill =
    failed && !cell.mark
      ? { first: "Couldn't match", second: "" }
      : textOf(choice)

  return (
    <div role="group" aria-label={label} className="relative min-w-0">
      <img
        src={url}
        alt={`Photo of ${label.toLowerCase()}`}
        data-changed={changed || undefined}
        className={`aspect-[5/7] w-full rounded-md bg-muted object-cover ${changed ? "ring-2 ring-primary" : ""}`}
      />
      <div className="absolute inset-x-1 bottom-1 flex items-center gap-1">
        {loading ? (
          <p className={PILL}>Matching…</p>
        ) : (
          <DropdownMenu>
            <DropdownMenuTrigger
              render={
                <button
                  type="button"
                  aria-label={pillName(label, choice, changed, failed)}
                  className={`${PILL} outline-none focus-visible:ring-3 focus-visible:ring-ring`}
                />
              }
            >
              <span className="flex min-w-0 flex-1 flex-col">
                <span className="truncate font-medium">{pill.first}</span>
                {pill.second && <span className="truncate">{pill.second}</span>}
              </span>
              <RiArrowDownSLine className="size-4 shrink-0" aria-hidden />
            </DropdownMenuTrigger>
            <DropdownMenuContent
              align="start"
              className="w-[min(30rem,calc(100vw-1rem))]"
            >
              <DropdownMenuRadioGroup
                value={choice.value}
                onValueChange={(value) =>
                  isMark(value) ? onMark(value) : onChoose(value)
                }
                className="grid gap-1 sm:grid-cols-[minmax(0,3fr)_minmax(0,2fr)]"
              >
                {candidates.length > 0 && (
                  <DropdownMenuGroup>
                    <DropdownMenuLabel>Candidates</DropdownMenuLabel>
                    {candidates.map((card) => (
                      <DropdownMenuRadioItem
                        key={card.id}
                        value={card.id}
                        className="py-1.5"
                        closeOnClick
                      >
                        <span className="truncate">{card.name}</span>
                        <span className="shrink-0 text-xs">
                          {cardCode(card)}
                        </span>
                      </DropdownMenuRadioItem>
                    ))}
                  </DropdownMenuGroup>
                )}
                <DropdownMenuGroup>
                  <DropdownMenuLabel>Mark as</DropdownMenuLabel>
                  {MARKS.map(({ mark, text }) => (
                    <DropdownMenuRadioItem
                      key={mark}
                      value={mark}
                      className="py-1.5"
                      closeOnClick
                    >
                      {text}
                    </DropdownMenuRadioItem>
                  ))}
                </DropdownMenuGroup>
              </DropdownMenuRadioGroup>
              {offersReprints && (
                <>
                  <DropdownMenuSeparator />
                  <DropdownMenuSub>
                    <DropdownMenuSubTrigger>
                      Other prints
                    </DropdownMenuSubTrigger>
                    <DropdownMenuSubContent className="w-72">
                      <OtherPrints
                        cardIds={candidates.map((card) => card.id)}
                        onPick={onReprint}
                      />
                    </DropdownMenuSubContent>
                  </DropdownMenuSub>
                </>
              )}
            </DropdownMenuContent>
          </DropdownMenu>
        )}
        {failed && (
          <Button
            size="sm"
            variant="secondary"
            className="h-8 shrink-0"
            onClick={onRetry}
          >
            Retry
          </Button>
        )}
      </div>
    </div>
  )
}

function textOf(choice: Choice) {
  if (choice.card)
    return { first: choice.card.name, second: cardCode(choice.card) }
  return {
    first: choice.value === "empty" ? "Empty" : "Unidentified",
    second: "",
  }
}

function pillName(
  label: string,
  choice: Choice,
  changed: boolean,
  failed: boolean
) {
  const what = choice.card
    ? `${choice.card.name}, ${choice.card.expansionSet.name} ${choice.card.localId}`
    : choice.value === "empty"
      ? "Empty"
      : failed
        ? "Couldn't match, Unidentified"
        : "Unidentified"
  return `${label}: ${what}${changed ? ". Changes this Slot" : ""}`
}
