import { useRef, useState } from "react"
import { useQueryClient } from "@tanstack/react-query"
import { useMoveBinderSlot } from "@/generated/endpoints/binder/binder"
import type { BinderSlot } from "@/generated/models"
import { refreshAfterMove, slotKey } from "@/lib/binder"
import type { SlotRef } from "@/lib/binder"
import { errorDetail, NETWORK_ERROR } from "@/lib/collections"

/**
 * The move of a binder view: pick a Slot as the source, then tap a target.
 * Tapping the source again cancels; any other Slot takes the card, or swaps
 * with the card it holds. Taps wait while a move is in flight.
 */
export function useBinderMove(collectionId: string) {
  const queryClient = useQueryClient()
  const moveMutation = useMoveBinderSlot()
  // The Slot picked as a move's source, with what it holds for the banner. It
  // stays picked while the user turns to another Page.
  const [moving, setMoving] = useState<{
    source: SlotRef
    label: string
  } | null>(null)
  const [moveError, setMoveError] = useState<string | null>(null)
  // A ref, not moveMutation.isPending: the pending state reaches render a tick
  // after the tap, too late for a second tap in the same frame.
  const moveInFlight = useRef(false)

  function startMove(slot: BinderSlot, source: SlotRef) {
    setMoveError(null)
    setMoving({
      source,
      label:
        slot.state === "card" && slot.card ? slot.card.name : "Unidentified",
    })
  }

  function cancelMove() {
    setMoving(null)
  }

  function moveTo(target: SlotRef) {
    if (!moving || moveInFlight.current) return
    if (slotKey(target) === slotKey(moving.source)) {
      setMoving(null)
      return
    }
    moveInFlight.current = true
    setMoveError(null)
    moveMutation.mutate(
      { id: collectionId, data: { from: moving.source, to: target } },
      {
        onSuccess: (response) => {
          if (response.status !== 200) {
            setMoveError(
              errorDetail(response, "Could not move this card.") ?? null
            )
            return
          }
          refreshAfterMove(queryClient, collectionId)
          setMoving(null)
        },
        onError: () => setMoveError(NETWORK_ERROR),
        onSettled: () => {
          moveInFlight.current = false
        },
      }
    )
  }

  return { moving, moveError, startMove, cancelMove, moveTo }
}
