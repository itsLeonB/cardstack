import { z } from "zod"

/**
 * A Draft Addition (GLOSSARY.md): the Cards a user scanned for one Collection,
 * as quantities to add on top of what it holds. It lives in memory and is
 * mirrored to local storage per Collection, as ids and quantities only; card
 * details are fetched when rendered.
 */
export interface DraftRow {
  cardId: string
  quantity: number
}

const rowsSchema = z.array(
  z.object({ cardId: z.string(), quantity: z.number().int().min(1) })
)

const storageKey = (collectionId: string) =>
  `cardstack:draft-addition:${collectionId}`

export function addOne(rows: DraftRow[], cardId: string): DraftRow[] {
  return rows.some((r) => r.cardId === cardId)
    ? rows.map((r) =>
        r.cardId === cardId ? { ...r, quantity: r.quantity + 1 } : r
      )
    : [...rows, { cardId, quantity: 1 }]
}

/** Stops at 1: dropping a row is `removeRow`. */
export function lowerOne(rows: DraftRow[], cardId: string): DraftRow[] {
  return rows.map((r) =>
    r.cardId === cardId ? { ...r, quantity: Math.max(1, r.quantity - 1) } : r
  )
}

export function removeRow(rows: DraftRow[], cardId: string): DraftRow[] {
  return rows.filter((r) => r.cardId !== cardId)
}

export function loadDraft(collectionId: string): DraftRow[] {
  try {
    const raw = localStorage.getItem(storageKey(collectionId))
    const parsed = rowsSchema.safeParse(raw === null ? [] : JSON.parse(raw))
    return parsed.success ? parsed.data : []
  } catch {
    return []
  }
}

export function saveDraft(collectionId: string, rows: DraftRow[]) {
  try {
    if (rows.length === 0) localStorage.removeItem(storageKey(collectionId))
    else localStorage.setItem(storageKey(collectionId), JSON.stringify(rows))
  } catch {
    // Storage full or blocked: the in-memory draft still works for this visit.
  }
}

export function clearDraft(collectionId: string) {
  saveDraft(collectionId, [])
}
