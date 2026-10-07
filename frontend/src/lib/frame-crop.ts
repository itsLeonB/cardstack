// The viewfinder shows the frame cover-fitted into a VIEW_ASPECT box (width /
// height), with a centred card-shaped guide that is GUIDE_HEIGHT of the box's
// height. These constants place the on-screen guide and drive the crop, so the
// photo is cut where the user saw the guide. A picked file takes the same crop.
export const VIEW_ASPECT = 3 / 4
export const GUIDE_HEIGHT = 0.85
export const CARD_ASPECT = 63 / 88
export const UPLOAD_LONG_SIDE = 1024

export interface Rect {
  x: number
  y: number
  w: number
  h: number
}

/** The part of a width x height frame that sits under the guide, in frame pixels. */
export function guideCrop(width: number, height: number): Rect {
  const visibleH = width / height > VIEW_ASPECT ? height : width / VIEW_ASPECT
  const h = Math.round(visibleH * GUIDE_HEIGHT)
  const w = Math.round(h * CARD_ASPECT)
  return {
    x: Math.round((width - w) / 2),
    y: Math.round((height - h) / 2),
    w,
    h,
  }
}

export function fitLongSide(w: number, h: number, limit: number) {
  const scale = Math.min(1, limit / Math.max(w, h))
  return { w: Math.round(w * scale), h: Math.round(h * scale) }
}
