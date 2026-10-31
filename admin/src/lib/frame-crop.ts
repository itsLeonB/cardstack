// The one frame helper `page-cut.ts` needs, copied from `frontend/src/lib/frame-crop.ts`:
// the Admin app keeps its own copy of the cut (ADR-0024), so both must stay in step.
export const UPLOAD_LONG_SIDE = 1024

export function fitLongSide(w: number, h: number, limit: number) {
  const scale = Math.min(1, limit / Math.max(w, h))
  return { w: Math.round(w * scale), h: Math.round(h * scale) }
}
