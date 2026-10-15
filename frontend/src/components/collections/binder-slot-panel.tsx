import { useEffect, useRef, useState } from "react"
import { useQueryClient } from "@tanstack/react-query"
import {
  useClearBinderSlot,
  useMarkBinderSlotUnidentified,
  usePlaceBinderCard,
} from "@/generated/endpoints/binder/binder"
import type { BinderSlot } from "@/generated/models"
import { InfiniteCardResults } from "@/components/catalog/infinite-card-results"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { refreshAfterSlotWrite } from "@/lib/binder"
import type { SlotRef } from "@/lib/binder"
import { errorDetail, NETWORK_ERROR } from "@/lib/collections"
import {
  useInfiniteCardResultsProps,
  useInfiniteCatalogCards,
} from "@/lib/infinite-catalog-cards"

type SlotAction = "place" | "replace" | "clear" | "markUnidentified"

// Menu order. A Move action (ticket 04) joins this list for a Slot that holds
// a Card or is Unidentified.
function slotActions(state: BinderSlot["state"]): SlotAction[] {
  switch (state) {
    case "card":
      return ["replace", "clear", "markUnidentified"]
    case "unidentified":
      return ["place", "clear"]
    default:
      return ["place"]
  }
}

const ACTION_LABELS = {
  place: "Place a Card",
  replace: "Replace card",
  clear: "Clear",
  markUnidentified: "Mark Unidentified",
} satisfies Record<SlotAction, string>

type Written = NonNullable<Parameters<typeof errorDetail>[0]>

/**
 * The actions for one tapped Slot. Mount with a key per Slot so a new tap
 * starts with no search open and no error shown.
 */
export function SlotPanel({
  collectionId,
  at,
  slot,
  onClose,
}: {
  collectionId: string
  /** Where the Slot sits, which the write addresses. */
  at: SlotRef
  slot: BinderSlot
  /** Closes the panel, after a write succeeds or from the Close button. */
  onClose: () => void
}) {
  const queryClient = useQueryClient()
  const [searching, setSearching] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const placeMutation = usePlaceBinderCard()
  const clearMutation = useClearBinderSlot()
  const markMutation = useMarkBinderSlotUnidentified()

  const target = { id: collectionId, ...at }

  // Focus moves into the panel when a Slot is tapped. On a small screen the
  // panel sits below the Page, so it also scrolls into view.
  const panelRef = useRef<HTMLElement>(null)
  useEffect(() => {
    panelRef.current?.focus({ preventScroll: true })
    panelRef.current?.scrollIntoView({ block: "nearest" })
  }, [])
  const busy =
    placeMutation.isPending || clearMutation.isPending || markMutation.isPending

  // A refused write stays inline so the Slot can be tried again; an accepted
  // one refreshes every query it changed.
  function settle(fallback: string) {
    return {
      onSuccess: (response: Written) => {
        if (response.status !== 200) {
          setError(errorDetail(response, fallback) ?? null)
          return
        }
        refreshAfterSlotWrite(queryClient, collectionId)
        onClose()
      },
      onError: () => setError(NETWORK_ERROR),
    }
  }

  function placeCard(cardId: string) {
    setError(null)
    placeMutation.mutate(
      { ...target, data: { cardId } },
      settle("Could not place this Card.")
    )
  }

  function runAction(action: SlotAction) {
    setError(null)
    if (action === "place" || action === "replace") {
      setSearching(true)
    } else if (action === "clear") {
      clearMutation.mutate(target, settle("Could not clear this Slot."))
    } else {
      markMutation.mutate(
        target,
        settle("Could not mark this Slot Unidentified.")
      )
    }
  }

  const stateLabel =
    slot.state === "card" && slot.card
      ? slot.card.name
      : slot.state === "unidentified"
        ? "Unidentified"
        : "empty"

  return (
    <section
      ref={panelRef}
      tabIndex={-1}
      aria-label="Slot actions"
      className="flex flex-col gap-3 rounded-md border p-3"
    >
      <h3 className="text-sm font-medium">
        Page {at.page}, row {at.row}, column {at.column}: {stateLabel}
      </h3>

      {!searching && (
        <div className="flex flex-wrap gap-2">
          {slotActions(slot.state).map((action) => (
            <Button
              key={action}
              variant="outline"
              disabled={busy}
              onClick={() => runAction(action)}
            >
              {ACTION_LABELS[action]}
            </Button>
          ))}
          <Button variant="ghost" onClick={onClose}>
            Close
          </Button>
        </div>
      )}

      {searching && (
        <CardPicker
          disabled={busy}
          onPick={placeCard}
          onCancel={() => setSearching(false)}
        />
      )}

      {error && (
        <p role="alert" className="text-sm text-destructive">
          {error}
        </p>
      )}
    </section>
  )
}

/** The catalog name search, with a Place button on each Card it finds. */
function CardPicker({
  disabled,
  onPick,
  onCancel,
}: {
  disabled: boolean
  onPick: (cardId: string) => void
  onCancel: () => void
}) {
  const [draft, setDraft] = useState("")
  const inputRef = useRef<HTMLInputElement>(null)
  // Opening the search puts the cursor in its name field.
  useEffect(() => {
    inputRef.current?.focus()
  }, [])
  // Results load only once a name is submitted, so an empty search never lists the catalog.
  const [name, setName] = useState<string | null>(null)

  function handleSubmit(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault()
    setName(draft.trim() || null)
  }

  return (
    <div className="flex flex-col gap-3">
      <form onSubmit={handleSubmit} className="flex flex-wrap items-end gap-2">
        <Input
          ref={inputRef}
          type="search"
          aria-label="Card name"
          placeholder="e.g. Pikachu"
          className="max-w-xs"
          value={draft}
          onChange={(event) => setDraft(event.target.value)}
        />
        <Button type="submit" variant="outline">
          Search cards
        </Button>
        <Button type="button" variant="ghost" onClick={onCancel}>
          Cancel
        </Button>
      </form>
      {name && <CardResults name={name} disabled={disabled} onPick={onPick} />}
    </div>
  )
}

function CardResults({
  name,
  disabled,
  onPick,
}: {
  name: string
  disabled: boolean
  onPick: (cardId: string) => void
}) {
  const query = useInfiniteCatalogCards({ name })
  const results = useInfiniteCardResultsProps(query)
  return (
    <InfiniteCardResults
      {...results}
      emptyMessage="No cards match this name."
      renderControl={(card) => (
        <Button
          size="sm"
          disabled={disabled}
          aria-label={`Place ${card.name}`}
          onClick={() => onPick(card.id)}
        >
          Place
        </Button>
      )}
    />
  )
}
