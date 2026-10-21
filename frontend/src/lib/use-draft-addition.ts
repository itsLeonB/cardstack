import { useCallback, useEffect, useRef, useState } from "react"
import { matchScannedCard } from "@/generated/endpoints/scan/scan"
import type { CardSummary, EnvelopeMatchResult } from "@/generated/models"
import { loadDraft, saveDraft } from "./draft-addition"
import { MAX_MATCHES_IN_FLIGHT, MatchQueue } from "./match-queue"

const MATCH_TIMEOUT_MS = 20_000
/** How many candidates an unsure Scan keeps for its dropdown. */
export const MAX_CANDIDATES = 5

export type ScanStatus = "queued" | "loading" | "resolved" | "failed"

/**
 * One photographed card in the Draft Addition. Only `resolved` Scans are kept
 * in local storage; the photo of a pending or failed one lives in memory so
 * Retry can re-send it.
 */
export interface Scan {
  key: string
  status: ScanStatus
  quantity: number
  photo?: Blob
  cardId?: string
  /** Up to MAX_CANDIDATES Cards when the matcher was not confident, top first; empty when it was. */
  candidateIds: string[]
  reason?: string
  /** The match response as the API sent it, kept with a resolved Scan. */
  detection?: EnvelopeMatchResult
  /** Where the chosen Card sits in the Reprints list, when the user picked it from "Other prints". */
  reprintPosition?: number
}

let lastKey = 0
const newKey = () => `scan-${++lastKey}`

/** The Draft Addition for one Collection (remount per Collection). Matches run through a queue of at most MAX_MATCHES_IN_FLIGHT. */
export function useScanDraft(
  userId: string,
  collectionId: string,
  announce: (message: string) => void
) {
  const [scans, setScans] = useState<Scan[]>([])
  // Nothing is saved before the saved Scans are back, so the empty first render cannot overwrite them.
  const [restored, setRestored] = useState(false)
  // Cards the matcher has returned this visit, so a row shows at once.
  const seen = useRef(new Map<string, CardSummary>())
  const cancelled = useRef(new Set<string>())
  const announceRef = useRef(announce)
  announceRef.current = announce
  // A Scan removed or discarded mid-match must not announce its result.
  const say = (key: string, message: string) => {
    if (!cancelled.current.has(key)) announceRef.current(message)
  }

  const patch = useCallback(
    (key: string, change: Partial<Scan>) =>
      setScans((list) =>
        list.map((s) => (s.key === key ? { ...s, ...change } : s))
      ),
    []
  )

  const [queue] = useState(
    () =>
      new MatchQueue<Blob>(
        MAX_MATCHES_IN_FLIGHT,
        async (key, photo, signal) => {
          patch(key, { status: "loading" })
          try {
            const response = await matchScannedCard(photo, { signal })
            if (response.status !== 200) throw new Error("match failed")
            const { confident, candidates } = response.data.data
            for (const { card } of candidates ?? [])
              seen.current.set(card.id, card)
            const top = candidates?.[0]?.card
            if (!top) {
              patch(key, { status: "failed", reason: "No card matched." })
              say(key, "Scan failed: no card matched.")
            } else if (confident) {
              patch(key, {
                status: "resolved",
                cardId: top.id,
                candidateIds: [],
                detection: response.data,
              })
              say(key, `Matched ${top.name}.`)
            } else {
              patch(key, {
                status: "resolved",
                cardId: top.id,
                detection: response.data,
                candidateIds: candidates!
                  .slice(0, MAX_CANDIDATES)
                  .map((c) => c.card.id),
              })
              say(
                key,
                `Possible match: ${top.name}. Choose another card if it is wrong.`
              )
            }
          } catch {
            if (cancelled.current.has(key)) return
            patch(key, {
              status: "failed",
              reason: "Could not match this photo.",
            })
            say(key, "Scan failed: could not match this photo.")
          }
        },
        MATCH_TIMEOUT_MS
      )
  )

  useEffect(() => () => queue.cancelAll(), [queue])
  useEffect(() => {
    if (!restored) return
    void saveDraft(
      userId,
      collectionId,
      scans.flatMap((s) =>
        s.status === "resolved"
          ? [
              {
                cardId: s.cardId!,
                quantity: s.quantity,
                candidateIds: s.candidateIds,
                photo: s.photo,
                detection: s.detection,
                reprintPosition: s.reprintPosition,
              },
            ]
          : []
      )
    )
  }, [userId, collectionId, restored, scans])

  useEffect(() => {
    let live = true
    void loadDraft(userId, collectionId).then((saved) => {
      if (!live) return
      const restoredScans = saved.map((s): Scan => ({
        ...s,
        key: newKey(),
        status: "resolved",
      }))
      setScans((list) => [...list, ...restoredScans])
      setRestored(true)
    })
    return () => {
      live = false
    }
  }, [userId, collectionId])

  return {
    scans,
    seen: seen.current,
    pendingCount: scans.filter(
      (s) => s.status === "queued" || s.status === "loading"
    ).length,
    capture(photo: Blob) {
      const key = newKey()
      setScans((list) => [
        { key, status: "queued", quantity: 1, photo, candidateIds: [] },
        ...list,
      ])
      queue.add(key, photo)
    },
    retry(key: string) {
      const scan = scans.find((s) => s.key === key)
      if (!scan?.photo) return
      patch(key, { status: "queued", reason: undefined })
      queue.add(key, scan.photo)
    },
    remove(key: string) {
      cancelled.current.add(key)
      queue.cancel(key)
      setScans((list) => list.filter((s) => s.key !== key))
    },
    raise: (key: string) =>
      setScans((list) =>
        list.map((s) =>
          s.key === key && s.status === "resolved"
            ? { ...s, quantity: s.quantity + 1 }
            : s
        )
      ),
    lower: (key: string) =>
      setScans((list) =>
        list.map((s) =>
          s.key === key && s.status === "resolved"
            ? { ...s, quantity: Math.max(1, s.quantity - 1) }
            : s
        )
      ),
    choose: (key: string, cardId: string, reprintPosition?: number) =>
      patch(key, { cardId, reprintPosition }),
    discard() {
      for (const s of scans) cancelled.current.add(s.key)
      queue.cancelAll()
      setScans([])
    },
    /** Takes the copies the Collection has placed out of the draft; returns how many Scans are still in it. */
    removeCopies(placed: Record<string, number>) {
      const owed = { ...placed }
      const remaining = scans.flatMap((s) => {
        if (s.status !== "resolved" || !owed[s.cardId!]) return [s]
        const taken = Math.min(owed[s.cardId!], s.quantity)
        owed[s.cardId!] -= taken
        return s.quantity > taken
          ? [{ ...s, quantity: s.quantity - taken }]
          : []
      })
      setScans(remaining)
      return remaining.length
    },
  }
}

/** One row per Card: the quantities of its resolved Scans, added together, in first-seen order. */
export function summedRows(scans: Scan[]) {
  const rows = new Map<string, number>()
  for (const s of scans)
    if (s.status === "resolved")
      rows.set(s.cardId!, (rows.get(s.cardId!) ?? 0) + s.quantity)
  return [...rows].map(([cardId, quantity]) => ({ cardId, quantity }))
}
