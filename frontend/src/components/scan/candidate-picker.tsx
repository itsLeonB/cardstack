import type { ReactNode } from "react"
import { RiArrowDownSLine } from "@remixicon/react"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuRadioGroup,
  DropdownMenuRadioItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import { CardThumb } from "./card-thumb"
import { MISSING_NAME, useDraftCards } from "./use-draft-cards"
import type { Scan } from "@/lib/use-draft-addition"
import type { CardSummary } from "@/generated/models"

/**
 * The card of an unsure Scan, opening a dropdown of its candidates with the
 * current choice marked. The candidates are fetched only once it opens, since
 * the menu's content mounts on open.
 */
export function CandidatePicker({
  scan,
  seen,
  label,
  children,
  onChoose,
}: {
  scan: Scan
  seen: Map<string, CardSummary>
  label: string
  /** The card face the user taps to open the menu. */
  children: ReactNode
  onChoose: (cardId: string) => void
}) {
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
        <CandidateList
          candidateIds={scan.candidateIds}
          chosen={scan.cardId}
          seen={seen}
          onChoose={onChoose}
        />
      </DropdownMenuContent>
    </DropdownMenu>
  )
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
