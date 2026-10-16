import { useEffect, useRef, useState } from "react"
import { matchScannedCard } from "@/generated/endpoints/scan/scan"
import type {
  BinderPageSlotBody,
  BinderSlot,
  CardSummary,
} from "@/generated/models"
import { MAX_MATCHES_IN_FLIGHT, MatchQueue } from "./match-queue"
import { MAX_CANDIDATES } from "./use-draft-addition"

// The same budget the Phase 1 scan gives its matcher.
const MATCH_TIMEOUT_MS = 20_000

export type CellMatch =
  | { status: "queued" }
  | { status: "loading" }
  | { status: "failed" }
  /** No candidates at all is a no-good-candidate cell; `confident` says whether the top one is sure. */
  | { status: "matched"; confident: boolean; candidates: CardSummary[] }

interface Job {
  photo: Blob
  generation: number
}

/** One pocket of the photographed Page, in grid order. */
export interface PageCell {
  photo: Blob
  match: CellMatch
  /** The Card the user chose from the candidates; the top one until they choose. */
  cardId?: string
  /** A state the user set by hand, which overrides what the scan found. */
  mark?: "empty" | "unidentified"
}

/**
 * The matches of one Page scan, held in memory only. Each cell's photo goes
 * through the Phase 1 `MatchQueue`, so at most MAX_MATCHES_IN_FLIGHT run at once.
 */
export function usePageMatches() {
  const [cells, setCells] = useState<PageCell[]>([])
  // Each start bumps this, so a match still running from the last start
  // cannot write into the new cells.
  const generation = useRef(0)
  const [queue] = useState(
    () =>
      new MatchQueue<Job>(
        MAX_MATCHES_IN_FLIGHT,
        async (id, { photo, generation: started }, signal) => {
          const index = Number(id)
          const stale = () => started !== generation.current
          patch(index, { match: { status: "loading" } })
          try {
            const response = await matchScannedCard(photo, { signal })
            if (response.status !== 200) throw new Error("match failed")
            const { confident, candidates } = response.data.data
            const cards = (candidates ?? [])
              .slice(0, MAX_CANDIDATES)
              .map((c) => c.card)
            if (!stale())
              patch(index, {
                match: { status: "matched", confident, candidates: cards },
                cardId: cards[0]?.id,
              })
          } catch {
            if (!stale()) patch(index, { match: { status: "failed" } })
          }
        },
        MATCH_TIMEOUT_MS
      )
  )

  function patch(index: number, change: Partial<PageCell>) {
    setCells((list) =>
      list.map((cell, i) => (i === index ? { ...cell, ...change } : cell))
    )
  }

  useEffect(() => () => queue.cancelAll(), [queue])

  return {
    cells,
    pending: cells.some(
      (c) => c.match.status === "queued" || c.match.status === "loading"
    ),
    /** Starts the matches for the cells, in grid order, replacing any still running. */
    start(photos: Blob[]) {
      generation.current++
      queue.cancelAll()
      setCells(photos.map((photo) => ({ photo, match: { status: "queued" } })))
      photos.forEach((photo, index) =>
        queue.add(String(index), { photo, generation: generation.current })
      )
    },
    retry(index: number) {
      patch(index, { match: { status: "queued" } })
      queue.add(String(index), {
        photo: cells[index].photo,
        generation: generation.current,
      })
    },
    choose(index: number, cardId: string) {
      patch(index, { cardId })
    },
    mark(index: number, mark: PageCell["mark"]) {
      patch(index, { mark })
    },
  }
}

/** The state a cell is saved as, or null while its match is still running. */
export function slotTarget(cell: PageCell): BinderPageSlotBody | null {
  if (cell.mark) return { state: cell.mark }
  const { match } = cell
  if (match.status === "queued" || match.status === "loading") return null
  // A failed match and a cell with no candidates both end as Unidentified.
  if (match.status === "failed" || match.candidates.length === 0)
    return { state: "unidentified" }
  return { state: "card", cardId: cell.cardId }
}

/** Whether a cell's target is different from the Slot the Page holds there. */
export function changesSlot(
  target: BinderPageSlotBody,
  slot: BinderSlot | undefined
): boolean {
  const state = slot?.state ?? "empty"
  if (target.state !== state) return true
  return target.state === "card" && target.cardId !== slot?.card.id
}

/** An object URL for a blob, revoked when the blob changes or the component unmounts. */
export function useObjectUrl(blob: Blob | undefined) {
  const [url, setUrl] = useState<string>()
  useEffect(() => {
    if (!blob) return
    const next = URL.createObjectURL(blob)
    setUrl(next)
    return () => URL.revokeObjectURL(next)
  }, [blob])
  return blob ? url : undefined
}
