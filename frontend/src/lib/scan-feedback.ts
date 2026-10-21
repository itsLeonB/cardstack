import {
  recordScanFeedback,
  requestScanPhotoUploads,
} from "@/generated/endpoints/scan/scan"
import type {
  EnvelopeMatchResult,
  ScanFeedbackRowInput,
} from "@/generated/models"
import type { Mark } from "@/lib/draft-addition"

/**
 * What the user did with a Detection (ADR-0022), recorded after a placement
 * goes through. Framework-free and generic over `source`, so Page Review sends
 * its rows through the same upload and post flow.
 *
 * - feedbackChoice(detection, choice): the kind, rank and rank source of one
 *   Scan. Undefined when no honest row exists.
 * - sendScanFeedback(rows, photos): uploads the photos the rows name, then
 *   posts the rows. Never rejects: a failure loses analytics, not the cards.
 */

/** The final choice of a Scan or cell. */
export interface Choice {
  cardId?: string
  mark?: Mark
  /** Where the chosen Card sits in the Reprints list, counted from 0 across pages. */
  reprintPosition?: number
}

export type FeedbackPick = Pick<
  ScanFeedbackRowInput,
  "kind" | "cardId" | "rank" | "rankSource"
>

/** A row whose photo, if any, is named by `photoId` until it is uploaded. */
export type PendingFeedbackRow = Omit<ScanFeedbackRowInput, "imageKey"> & {
  photoId?: string
}

// The API takes at most 100 rows a batch, and as many photo addresses in one call.
const MAX_ROWS = 100

export function feedbackChoice(
  detection: EnvelopeMatchResult | undefined,
  choice: Choice
): FeedbackPick | undefined {
  // Without a Detection (a draft saved by the old version) there is nothing to compare against.
  if (!detection) return undefined
  const offered = detection.data.candidates ?? []
  // What the matcher said: a Card, an empty photo, or no card and no candidate.
  const said =
    offered.length > 0 ? "card" : detection.data.empty ? "empty" : "none"

  if (choice.mark === "empty")
    return { kind: said === "empty" ? "confirmed" : "empty" }
  if (choice.mark === "unidentified")
    return { kind: said === "none" ? "confirmed" : "unidentified" }
  if (!choice.cardId) return undefined

  if (choice.reprintPosition !== undefined)
    return {
      kind: "reprint",
      cardId: choice.cardId,
      rank: choice.reprintPosition + 1,
      rankSource: "reprint",
    }
  // A Card where the matcher offered none is `placed`. The API needs a rank
  // with a Card and there is no list to count in, so no row is made; nothing
  // in the review can make this choice yet.
  const index = offered.findIndex((c) => c.card.id === choice.cardId)
  if (index < 0) return undefined
  return {
    kind: index === 0 ? "confirmed" : "switched",
    cardId: choice.cardId,
    rank: index + 1,
    rankSource: "top",
  }
}

export async function sendScanFeedback(
  rows: PendingFeedbackRow[],
  photos: Map<string, Blob>
): Promise<void> {
  try {
    for (let i = 0; i < rows.length; i += MAX_ROWS) {
      const batch = rows.slice(i, i + MAX_ROWS)
      const keys = await uploadPhotos(batch, photos)
      // The bucket refused the addresses: nothing is recorded.
      if (!keys) return
      await recordScanFeedback({
        rows: batch.map(({ photoId, ...row }) => ({
          ...row,
          imageKey: photoId ? keys.get(photoId) : undefined,
        })),
      })
    }
  } catch {
    // Silent by design: the placement has already gone through.
  }
}

/**
 * The keys of the photos the batch names, by photo id. A photo that fails to
 * upload gets no key, so its row is still recorded. Undefined when the API
 * refused the addresses, which means the batch is not recorded at all.
 */
async function uploadPhotos(
  rows: PendingFeedbackRow[],
  photos: Map<string, Blob>
): Promise<Map<string, string> | undefined> {
  const ids = new Set<string>()
  for (const row of rows)
    if (row.photoId && photos.has(row.photoId)) ids.add(row.photoId)
  const keys = new Map<string, string>()
  if (ids.size === 0) return keys

  const list = [...ids]
  const response = await requestScanPhotoUploads({ count: list.length })
  if (response.status !== 200) return undefined
  // One address per photo, in the order asked.
  const uploads = response.data.data.uploads ?? []
  await Promise.all(
    list.map(async (id, i) => {
      const upload = uploads[i]
      const photo = photos.get(id)
      if (!upload || !photo) return
      try {
        await putPhoto(upload.url, photo)
        keys.set(id, upload.key)
      } catch {
        // This photo goes without its key; the row is still recorded.
      }
    })
  )
  return keys
}

// The address is signed for JPEG, and the card-scan photos are JPEG, so the
// header must say so.
async function putPhoto(url: string, photo: Blob): Promise<void> {
  const res = await fetch(url, {
    method: "PUT",
    headers: { "Content-Type": "image/jpeg" },
    body: photo,
  })
  if (!res.ok) throw new Error(`photo upload refused: ${res.status}`)
}
