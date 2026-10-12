// Uploads a Card's image from the admin (docs/adr/0019): ask the API for a
// short-lived signed address, PUT the file straight to the image bucket, then
// ask the API to confirm. The API checks the object's first bytes at confirm,
// so a mislabelled file is refused there and the Card keeps no image. Kept
// apart from the admin page so later checks and forms reuse the same flow.

import {
  confirmAdminCardImage,
  requestAdminCardImageUpload,
} from "@/generated/endpoints/admin/admin"
import type { AdminCard } from "@/generated/models"

export const IMAGE_TYPES = ["image/jpeg", "image/png", "image/webp"] as const
export type ImageType = (typeof IMAGE_TYPES)[number]

export function isImageType(type: string): type is ImageType {
  return IMAGE_TYPES.some((allowed) => allowed === type)
}

/** A failed step of the upload, worded for the person at the form. */
export class CardUploadError extends Error {}

function reasonOf(data: { detail?: string }, fallback: string): string {
  return data.detail ?? fallback
}

/**
 * PUTs `file` to a signed address. The bucket, not the API, receives it, so
 * the request carries no bearer token, and its Content-Type must equal the one
 * the address was signed for.
 */
export async function putToSignedUrl(
  url: string,
  file: Blob,
  contentType: ImageType
): Promise<void> {
  const res = await fetch(url, {
    method: "PUT",
    headers: { "Content-Type": contentType },
    body: file,
  })
  if (!res.ok) {
    throw new CardUploadError(
      `The image bucket refused the upload (${res.status}).`
    )
  }
}

/** Uploads `file` as the image of Card `cardId` and returns the confirmed Card. */
export async function uploadCardImage(
  cardId: string,
  file: File
): Promise<AdminCard> {
  if (!isImageType(file.type)) {
    throw new CardUploadError("The image must be a JPEG, PNG or WebP file.")
  }

  const signed = await requestAdminCardImageUpload(cardId, {
    contentType: file.type,
  })
  if (signed.status !== 200) {
    throw new CardUploadError(
      reasonOf(signed.data, `Could not start the upload (${signed.status}).`)
    )
  }

  await putToSignedUrl(signed.data.data.url, file, file.type)

  const confirmed = await confirmAdminCardImage(cardId)
  if (confirmed.status !== 200) {
    throw new CardUploadError(
      reasonOf(
        confirmed.data,
        `Could not confirm the upload (${confirmed.status}).`
      )
    )
  }
  return confirmed.data.data
}
