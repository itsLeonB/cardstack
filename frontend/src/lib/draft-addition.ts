import { z } from "zod"
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

/**
 * A resolved Scan: the chosen Card, its quantity, the ids of the other
 * candidates when the matcher was not confident (empty otherwise), its photo,
 * and the Detection. A Scan moved from the old local storage draft has neither
 * photo nor Detection.
 */
export interface ResolvedScan {
  cardId: string
  quantity: number
  candidateIds: string[]
  photo?: Blob
  /** The match response as the API sent it, so a field the API adds later is kept. */
  detection?: EnvelopeMatchResult
}

interface DraftRecord {
  userId: string
  collectionId: string
  scans: ResolvedScan[]
}

// Old drafts (card id and quantity only) fail this schema and are dropped.
const legacyScansSchema = z.array(
  z.object({
    cardId: z.string(),
    quantity: z.number().int().min(1),
    candidateIds: z.array(z.string()).max(5),
  })
)

const legacyKey = (collectionId: string) =>
  `cardstack:draft-addition:${collectionId}`

// Loads run one at a time, so a second load (a remounted screen) never moves
// a legacy draft that the first is still moving.
let queue: Promise<unknown> = Promise.resolve()

export function loadDraft(
  userId: string,
  collectionId: string
): Promise<ResolvedScan[]> {
  const loaded = queue.then(() => readDraft(userId, collectionId))
  queue = loaded
  return loaded
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

async function readDraft(
  userId: string,
  collectionId: string
): Promise<ResolvedScan[]> {
  const legacy = readLegacyDraft(collectionId)
  let saved: ResolvedScan[]
  try {
    const record = await transactStore<DraftRecord | undefined>(
      DB_NAME,
      STORE,
      KEY_PATH,
      "readonly",
      (store) => store.get([userId, collectionId])
    )
    saved = record?.scans ?? []
  } catch {
    return legacy
  }
  if (legacy.length === 0) {
    dropLegacyDraft(collectionId)
    return saved
  }
  const scans = [...saved, ...legacy]
  try {
    // The legacy draft is removed only once its Scans are safely stored.
    await writeDraft(userId, collectionId, scans)
  } catch {
    return scans
  }
  dropLegacyDraft(collectionId)
  return scans
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

function readLegacyDraft(collectionId: string): ResolvedScan[] {
  try {
    const raw = localStorage.getItem(legacyKey(collectionId))
    const parsed = legacyScansSchema.safeParse(
      raw === null ? [] : JSON.parse(raw)
    )
    return parsed.success ? parsed.data : []
  } catch {
    return []
  }
}

function dropLegacyDraft(collectionId: string) {
  try {
    localStorage.removeItem(legacyKey(collectionId))
  } catch {
    // Storage blocked: the next visit reads the same draft again.
  }
}
