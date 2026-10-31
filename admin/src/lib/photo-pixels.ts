// The browser half of the review photo: reading a stored photo's pixels and
// painting a cut onto a canvas. Needs a browser (fetch, canvas), so the cut
// itself is kept in `binder-cell.ts`, where a test can check it.

export interface Pixels {
  width: number
  height: number
  pixels: Uint8ClampedArray
}

/**
 * Reads a photo's RGBA pixels from its presigned address. The bucket must
 * allow this origin to GET, or the read fails (the deployment docs say so).
 */
export async function readPixels(url: string): Promise<Pixels> {
  const response = await fetch(url)
  if (!response.ok) {
    throw new Error(`The photo could not be read (status ${response.status}).`)
  }
  const bitmap = await createImageBitmap(await response.blob())
  const { width, height } = bitmap
  const { context } = canvasOf(width, height)
  context.drawImage(bitmap, 0, 0)
  bitmap.close()
  return {
    width,
    height,
    pixels: context.getImageData(0, 0, width, height).data,
  }
}

/** Paints the pixels onto `canvas`, resized to fit them. */
export function paintPixels(canvas: HTMLCanvasElement, image: Pixels) {
  canvas.width = image.width
  canvas.height = image.height
  const context = canvas.getContext("2d")
  if (!context) throw new Error("Canvas is not available")
  const data = context.createImageData(image.width, image.height)
  data.data.set(image.pixels)
  context.putImageData(data, 0, 0)
}

/** Encodes the pixels as the JPEG the review uploads, at the quality the user app's cut uses (`page-cut.ts`). */
export function encodePixelsJpeg(image: Pixels): Promise<Blob> {
  const { canvas, context } = canvasOf(image.width, image.height)
  const data = context.createImageData(image.width, image.height)
  data.data.set(image.pixels)
  context.putImageData(data, 0, 0)
  return new Promise((resolve, reject) =>
    canvas.toBlob(
      (blob) => (blob ? resolve(blob) : reject(new Error("Encoding failed"))),
      "image/jpeg",
      0.9
    )
  )
}

function canvasOf(width: number, height: number) {
  const canvas = document.createElement("canvas")
  canvas.width = width
  canvas.height = height
  const context = canvas.getContext("2d")
  if (!context) throw new Error("Canvas is not available")
  return { canvas, context }
}
