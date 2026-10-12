import { z } from "zod"

/**
 * A resolved Scan as kept in local storage: the chosen Card, its quantity, and
 * the ids of the other candidates when the matcher was not confident (empty
 * otherwise). Photos are never stored, and card details are fetched when shown.
 */
export interface ResolvedScan {
  cardId: string
  quantity: number
  candidateIds: string[]
}

// Old drafts (card id and quantity only) fail this schema and are dropped.
const scansSchema = z.array(
  z.object({
    cardId: z.string(),
    quantity: z.number().int().min(1),
    candidateIds: z.array(z.string()).max(5),
  })
)

const storageKey = (collectionId: string) =>
  `cardstack:draft-addition:${collectionId}`

export function loadDraft(collectionId: string): ResolvedScan[] {
  try {
    const raw = localStorage.getItem(storageKey(collectionId))
    const parsed = scansSchema.safeParse(raw === null ? [] : JSON.parse(raw))
    return parsed.success ? parsed.data : []
  } catch {
    return []
  }
}

export function saveDraft(collectionId: string, scans: ResolvedScan[]) {
  try {
    if (scans.length === 0) localStorage.removeItem(storageKey(collectionId))
    else localStorage.setItem(storageKey(collectionId), JSON.stringify(scans))
  } catch {
    // Storage full or blocked: the in-memory draft still works for this visit.
  }
}

export function clearDraft(collectionId: string) {
  saveDraft(collectionId, [])
}
