import { putToSignedUrl } from "@/lib/admin-card-upload"
import {
  approveReview,
  getReviewPhoto,
  requestReviewUpload,
} from "@/generated/endpoints/admin/admin"
import type { ReviewItem } from "@/generated/models"
import { binderCellCut } from "@/lib/binder-cell"
import { encodePixelsJpeg, readPixels } from "@/lib/photo-pixels"

// Approving a photo (ADR-0027): the image to embed is uploaded under the key the
// API names, and only then is that key approved. The API refuses an approval
// whose object is missing, so the order here is the contract.

const GONE = "This photo is no longer waiting for review."
const DECIDED =
  "Cannot approve: another Admin decided this photo, or its chosen Card was deleted."

/**
 * An approval the API or the bucket turned down, worded for the Admin. `refresh`
 * is set when the queue must be read again: the photo has left it, or another
 * Admin has decided it.
 */
export class ApprovalRefused extends Error {
  readonly refresh: boolean

  constructor(message: string, refresh: boolean) {
    super(message)
    this.refresh = refresh
  }
}

/**
 * Uploads the image to embed for `item`, then approves the key the upload
 * named. Throws an ApprovalRefused for a refusal the Admin can act on; a network
 * failure propagates as it is.
 */
export async function approveCorrection(item: ReviewItem): Promise<void> {
  const image = await imageToEmbed(item)

  const upload = await requestReviewUpload(item.id)
  if (upload.status !== 200) throw refusal(upload)
  await putToSignedUrl(upload.data.data.url, image, "image/jpeg")

  const approved = await approveReview(item.id, { key: upload.data.data.key })
  if (approved.status !== 204) throw refusal(approved)
}

// A binder cell is cut as the review shows it; a card-scan photo goes up as stored.
async function imageToEmbed(item: ReviewItem): Promise<Blob> {
  const photo = await getReviewPhoto(item.id)
  if (photo.status !== 200) throw refusal(photo)
  const url = photo.data.data.url

  if (item.source === "card_scan") {
    const stored = await fetch(url)
    if (!stored.ok) {
      throw new ApprovalRefused(
        `The photo could not be read (status ${stored.status}). Try again.`,
        false
      )
    }
    return stored.blob()
  }
  if (!item.binder) {
    throw new ApprovalRefused(
      "This Correction has no binder cell to cut.",
      false
    )
  }
  const cut = binderCellCut(await readPixels(url), item.binder)
  if (!cut) {
    throw new ApprovalRefused(
      "This cell has no corners to cut, so it can only be rejected.",
      false
    )
  }
  return encodePixelsJpeg(cut)
}

function refusal(response: {
  status: number
  data: { detail?: string }
}): ApprovalRefused {
  if (response.status === 404) return new ApprovalRefused(GONE, true)
  if (response.status === 409) return new ApprovalRefused(DECIDED, true)
  // A 500 can carry the raw error of the embedder or the bucket, so only the
  // API's own reasons (every other status) are shown.
  const { detail } = response.data
  const reason =
    detail && response.status !== 500
      ? `: ${detail}`
      : ` (status ${response.status})`
  return new ApprovalRefused(
    `Could not approve the photo${reason}. Try again.`,
    false
  )
}
