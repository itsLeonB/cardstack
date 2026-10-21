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
  /** No candidates at all is a no-good-candidate cell; `confident` says whether the top one is sure. `empty` is the matcher's no-card answer, and is absent in reviews saved before it existed. */
  | {
      status: "matched"
      confident: boolean
      candidates: CardSummary[]
      empty?: boolean
    }

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
  /** Empty or Unidentified, picked by hand in the cell's menu, which overrides the chosen Card. */
  mark?: "empty" | "unidentified"
  /** The Reprint the user picked from "Other prints", kept whole so the pill shows it on resume. Set with reprintPosition, cleared by any other pick. */
  reprintCard?: CardSummary
  /** Where that Reprint sat in the Reprints list when picked, in the order the list showed them. */
  reprintPosition?: number
}

// A cell that has no result yet, so it still has to match.
const isUnfinished = (cell: PageCell) =>
  cell.match.status === "queued" || cell.match.status === "loading"

/**
 * The matches of one Page scan. Each cell's photo goes through the Phase 1
 * `MatchQueue`, so at most MAX_MATCHES_IN_FLIGHT run at once. A resumed review
 * passes its saved cells: the unfinished ones match again, the rest keep their result.
 */
export function usePageMatches(resumed: PageCell[] = []) {
  const [cells, setCells] = useState<PageCell[]>(() =>
    resumed.map((cell): PageCell =>
      isUnfinished(cell) ? { ...cell, match: { status: "queued" } } : cell
    )
  )
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
            const { confident, candidates, empty } = response.data.data
            const cards = (candidates ?? [])
              .slice(0, MAX_CANDIDATES)
              .map((c) => c.card)
            if (!stale())
              patch(index, {
                match: {
                  status: "matched",
                  confident,
                  candidates: cards,
                  empty,
                },
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

  // Runs once on mount: the unfinished cells of a resumed review match again.
  useEffect(() => {
    resumed.forEach((cell, index) => {
      if (isUnfinished(cell))
        queue.add(String(index), {
          photo: cell.photo,
          generation: generation.current,
        })
    })
  }, [])

  return {
    cells,
    pending: cells.some(isUnfinished),
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
    /** Picks a candidate, which also clears a hand mark and a Reprint; the other candidates stay for a later pick. */
    choose(index: number, cardId: string) {
      patch(index, {
        cardId,
        mark: undefined,
        reprintCard: undefined,
        reprintPosition: undefined,
      })
    },
    mark(index: number, mark: PageCell["mark"]) {
      patch(index, { mark, reprintCard: undefined, reprintPosition: undefined })
    },
    /** Picks a Reprint from "Other prints", which becomes the cell's Card and remembers where it sat in the list. */
    reprint(index: number, card: CardSummary, position: number) {
      patch(index, {
        cardId: card.id,
        mark: undefined,
        reprintCard: card,
        reprintPosition: position,
      })
    },
  }
}

/** The state a cell is saved as, or null while its match is still running. */
/** The mark a cell is shown and saved with: the hand mark, else Empty when the matcher saw no card. */
export function cellMark(cell: PageCell): PageCell["mark"] {
  if (cell.mark) return cell.mark
  return cell.match.status === "matched" && cell.match.empty
    ? "empty"
    : undefined
}

export function slotTarget(cell: PageCell): BinderPageSlotBody | null {
  const mark = cellMark(cell)
  if (mark) return { state: mark }
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
  const [made, setMade] = useState<{ blob: Blob; url: string }>()
  useEffect(() => {
    if (!blob) return
    const url = URL.createObjectURL(blob)
    setMade({ blob, url })
    return () => URL.revokeObjectURL(url)
  }, [blob])
  // Only the URL made for this blob: the one for an older blob is revoked by now.
  return made && made.blob === blob ? made.url : undefined
}
