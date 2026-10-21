import type { ReactNode } from "react"
import { RiArrowDownSLine } from "@remixicon/react"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuCheckboxItem,
  DropdownMenuRadioGroup,
  DropdownMenuRadioItem,
  DropdownMenuSeparator,
  DropdownMenuSub,
  DropdownMenuSubContent,
  DropdownMenuSubTrigger,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import { CardThumb } from "./card-thumb"
import { OtherPrints } from "./other-prints"
import { MISSING_NAME, useDraftCards } from "./use-draft-cards"
import type { Mark } from "@/lib/draft-addition"
import type { Scan } from "@/lib/use-draft-addition"
import type { CardSummary } from "@/generated/models"

/** The marks, worded for a photo. A marked Scan's row says the lowercased wording. */
export const MARKS = {
  empty: "No card in the photo",
  unidentified: "A card, but none of these",
} satisfies Record<Mark, string>
const MARK_ORDER: Mark[] = ["empty", "unidentified"]

/**
 * The card of a resolved Scan, opening a menu of its candidates with the
 * current choice marked, and the marks. The candidates are fetched only once it
 * opens, since the menu's content mounts on open.
 */
export function CandidatePicker({
  scan,
  seen,
  label,
  children,
  onChoose,
  onMark,
}: {
  scan: Scan
  seen: Map<string, CardSummary>
  label: string
  /** The card face the user taps to open the menu. */
  children: ReactNode
  onChoose: (cardId: string, reprintPosition?: number) => void
  onMark: (mark: Mark) => void
}) {
  const ids = offeredIds(scan)
  return (
    <DropdownMenu>
      <DropdownMenuTrigger
        render={
          <button
            type="button"
            aria-label={`Change card for ${label}`}
            className="flex min-w-0 flex-1 items-center gap-2 rounded-lg text-left outline-none focus-visible:ring-3 focus-visible:ring-ring/30"
          />
        }
      >
        {children}
        <RiArrowDownSLine className="shrink-0" aria-hidden />
      </DropdownMenuTrigger>
      <DropdownMenuContent align="start" className="w-72">
        {ids.length > 0 && (
          <>
            <CandidateList
              candidateIds={ids}
              chosen={scan.mark ? undefined : scan.cardId}
              seen={seen}
              onChoose={onChoose}
            />
            <DropdownMenuSeparator />
            <DropdownMenuSub>
              <DropdownMenuSubTrigger>Other prints</DropdownMenuSubTrigger>
              <DropdownMenuSubContent className="w-72">
                <OtherPrints
                  cardIds={ids}
                  onPick={(card, position) => {
                    seen.set(card.id, card)
                    onChoose(card.id, position)
                  }}
                />
              </DropdownMenuSubContent>
            </DropdownMenuSub>
            <DropdownMenuSeparator />
          </>
        )}
        {MARK_ORDER.map((mark) => (
          <DropdownMenuCheckboxItem
            key={mark}
            checked={scan.mark === mark}
            closeOnClick
            onCheckedChange={() => onMark(mark)}
          >
            {MARKS[mark]}
          </DropdownMenuCheckboxItem>
        ))}
      </DropdownMenuContent>
    </DropdownMenu>
  )
}

/**
 * The Cards the matcher offered for a Scan, which stay the same when the user
 * picks another. A confident Scan keeps no candidate ids, so its one Card is
 * read from the Detection; the Scan's own choice may differ from it.
 */
function offeredIds(scan: Scan): string[] {
  if (scan.candidateIds.length > 0) return scan.candidateIds
  const top = scan.detection?.data.candidates?.[0]?.card.id
  const only = top ?? scan.cardId
  return only ? [only] : []
}

function CandidateList({
  candidateIds,
  chosen,
  seen,
  onChoose,
}: {
  candidateIds: string[]
  chosen?: string
  seen: Map<string, CardSummary>
  onChoose: (cardId: string) => void
}) {
  const { cards, missing } = useDraftCards(candidateIds, seen)
  return (
    <DropdownMenuRadioGroup
      value={chosen ?? ""}
      onValueChange={(cardId) => {
        // The picked card shows at once, before the sheet's own lookup answers.
        const card = cards.get(cardId)
        if (card) seen.set(cardId, card)
        onChoose(cardId)
      }}
    >
      {candidateIds.map((id) => {
        const card = cards.get(id)
        return (
          <DropdownMenuRadioItem key={id} value={id} className="py-1.5">
            <CardThumb card={card} />
            <CardText card={card} name={card?.name ?? MISSING_NAME[missing]} />
          </DropdownMenuRadioItem>
        )
      })}
    </DropdownMenuRadioGroup>
  )
}

function CardText({ card, name }: { card?: CardSummary; name: string }) {
  return (
    <span className="flex min-w-0 flex-1 flex-col">
      <span className="truncate text-sm font-medium">{name}</span>
      {card && (
        <span className="text-xs">
          {card.expansionSet.code} · No. {card.localId}
        </span>
      )}
    </span>
  )
}
