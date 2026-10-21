import type { EnvelopeMatchResult } from "@/generated/models"
import { transactStore } from "@/lib/indexed-db"

/**
 * The resolved Scans of the Draft Addition, in the browser's IndexedDB. One
 * record per signed-in user and Collection, so another login on the same
 * browser never sees them. Only resolved Scans are kept: pending and failed
 * Scans live in memory for the visit, as before.
 *
 * - loadDraft(userId, collectionId): the saved Scans, in saved order. A draft
 *   saved in local storage by the previous version is moved in here.
 * - saveDraft(userId, collectionId, scans): replaces the saved Scans; an empty
 *   list removes the record.
 *
 * Neither rejects. Without IndexedDB, loadDraft yields nothing and saveDraft
 * does nothing, so the draft works from memory for this visit.
 */
const DB_NAME = "cardstack-draft-additions"
const STORE = "drafts"
const KEY_PATH = ["userId", "collectionId"]

/** Why a Scan is set aside on confirm: no card in the photo, or a card none of its candidates fits. */
export type Mark = "empty" | "unidentified"

/**
 * A resolved Scan: the chosen Card, its quantity, the ids of the other
 * candidates when the matcher was not confident (empty otherwise), its photo,
 * and the Detection. A Scan with no Card (the matcher saw none) has only its
 * mark.
 */
export interface ResolvedScan {
  cardId?: string
  quantity: number
  candidateIds: string[]
  mark?: Mark
  photo?: Blob
  /** The match response as the API sent it, so a field the API adds later is kept. */
  detection?: EnvelopeMatchResult
  /** Where the chosen Card sits in the Reprints list, when the user picked it from "Other prints". */
  reprintPosition?: number
}

interface DraftRecord {
  userId: string
  collectionId: string
  scans: ResolvedScan[]
}

const legacyKey = (collectionId: string) =>
  `cardstack:draft-addition:${collectionId}`

export async function loadDraft(
  userId: string,
  collectionId: string
): Promise<ResolvedScan[]> {
  // Old drafts had no owner: delete one rather than move it, so no user inherits it.
  dropLegacyDraft(collectionId)
  try {
    const record = await transactStore<DraftRecord | undefined>(
      DB_NAME,
      STORE,
      KEY_PATH,
      "readonly",
      (store) => store.get([userId, collectionId])
    )
    return record?.scans ?? []
  } catch {
    return []
  }
}

export async function saveDraft(
  userId: string,
  collectionId: string,
  scans: ResolvedScan[]
): Promise<void> {
  try {
    await writeDraft(userId, collectionId, scans)
  } catch {
    // Storage missing, full or blocked: the in-memory draft still works for this visit.
  }
}

async function writeDraft(
  userId: string,
  collectionId: string,
  scans: ResolvedScan[]
): Promise<void> {
  const key = [userId, collectionId]
  if (scans.length === 0) {
    await transactStore(DB_NAME, STORE, KEY_PATH, "readwrite", (store) =>
      store.delete(key)
    )
    return
  }
  await transactStore(DB_NAME, STORE, KEY_PATH, "readwrite", (store) =>
    store.put({ userId, collectionId, scans } satisfies DraftRecord)
  )
}

function dropLegacyDraft(collectionId: string) {
  try {
    localStorage.removeItem(legacyKey(collectionId))
  } catch {
    // Storage blocked: the next load tries to delete it again.
  }
}
