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

// R2 cannot cap a presigned upload's size, so these limits only catch an
// honest mistake from the one trusted editor; they are not enforcement
// (docs/adr/0019). The size is for a phone photo; the shape is a card's.
export const MAX_IMAGE_BYTES = 10 * 1024 * 1024
const MIN_WIDTH = 300
const MAX_SIDE = 6000
const CARD_ASPECT = 63 / 88
const ASPECT_TOLERANCE = 0.15

/**
 * Throws a CardUploadError saying why `file` should not be uploaded: not a
 * JPEG, PNG or WebP, over the size limit, or not card-shaped. Runs before
 * anything is sent.
 */
export async function validateCardImage(file: File): Promise<ImageType> {
  if (!isImageType(file.type)) {
    throw new CardUploadError("The image must be a JPEG, PNG or WebP file.")
  }
  if (file.size > MAX_IMAGE_BYTES) {
    throw new CardUploadError("The image must be at most 10 MB.")
  }

  let bitmap: ImageBitmap
  try {
    bitmap = await createImageBitmap(file)
  } catch {
    throw new CardUploadError("The image could not be read.")
  }
  const { width, height } = bitmap
  bitmap.close()

  if (Math.abs(width / height / CARD_ASPECT - 1) > ASPECT_TOLERANCE) {
    throw new CardUploadError(
      `The image must be a portrait card photo, about 63:88 (it is ${width}x${height}).`
    )
  }
  if (width < MIN_WIDTH) {
    throw new CardUploadError(
      `The image must be at least ${MIN_WIDTH} pixels wide (it is ${width}).`
    )
  }
  if (Math.max(width, height) > MAX_SIDE) {
    throw new CardUploadError(
      `The image must be at most ${MAX_SIDE} pixels on its longest side (it is ${width}x${height}).`
    )
  }
  return file.type
}

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
  const contentType = await validateCardImage(file)

  const signed = await requestAdminCardImageUpload(cardId, {
    contentType,
  })
  if (signed.status !== 200) {
    throw new CardUploadError(
      reasonOf(signed.data, `Could not start the upload (${signed.status}).`)
    )
  }

  await putToSignedUrl(signed.data.data.url, file, contentType)

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
