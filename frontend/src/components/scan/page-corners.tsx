import { useRef } from "react"
import type { PointerEvent as ReactPointerEvent } from "react"
import type { Corners, Point } from "@/lib/page-cut"

const NAMES = ["Top left", "Top right", "Bottom right", "Bottom left"]
// Arrow keys nudge a corner by a fraction of the photo; Shift nudges further.
const NUDGE = 0.01
const NUDGE_FAST = 0.05

/** The photo with four corners to drag onto the outer corners of the pocket grid. */
export function PageCorners({
  src,
  corners,
  onChange,
}: {
  src: string
  corners: Corners
  onChange: (corners: Corners) => void
}) {
  const box = useRef<HTMLDivElement>(null)
  const dragging = useRef<number | null>(null)

  // The photo fills the box at its own aspect ratio, so 0 to 1 is the whole image on both axes.
  function pointAt(event: ReactPointerEvent) {
    const rect = box.current!.getBoundingClientRect()
    return {
      x: clamp((event.clientX - rect.left) / rect.width),
      y: clamp((event.clientY - rect.top) / rect.height),
    }
  }

  function move(index: number, point: Point) {
    // SAFETY: map keeps the four-corner tuple's length, so the cast holds.
    onChange(corners.map((c, i) => (i === index ? point : c)) as Corners)
  }

  return (
    <div
      ref={box}
      className="relative w-full touch-none select-none"
      onPointerMove={(event) => {
        if (dragging.current !== null) move(dragging.current, pointAt(event))
      }}
      onPointerUp={() => (dragging.current = null)}
      onPointerCancel={() => (dragging.current = null)}
    >
      <img src={src} alt="" draggable={false} className="block w-full" />
      <svg
        aria-hidden
        viewBox="0 0 1 1"
        preserveAspectRatio="none"
        className="pointer-events-none absolute inset-0 size-full"
      >
        <polygon
          points={corners.map((c) => `${c.x},${c.y}`).join(" ")}
          fill="none"
          stroke="white"
          strokeWidth={2}
          vectorEffect="non-scaling-stroke"
        />
      </svg>
      {corners.map((corner, index) => (
        <button
          key={NAMES[index]}
          type="button"
          aria-label={`${NAMES[index]} corner`}
          className="absolute size-7 -translate-x-1/2 -translate-y-1/2 touch-none rounded-full border-2 border-white bg-primary/80 focus-visible:ring-3 focus-visible:ring-ring/50 focus-visible:outline-none"
          style={{ left: `${corner.x * 100}%`, top: `${corner.y * 100}%` }}
          onPointerDown={(event) => {
            dragging.current = index
            event.currentTarget.setPointerCapture(event.pointerId)
          }}
          onKeyDown={(event) => {
            const step = event.shiftKey ? NUDGE_FAST : NUDGE
            const delta = {
              ArrowLeft: { x: -step, y: 0 },
              ArrowRight: { x: step, y: 0 },
              ArrowUp: { x: 0, y: -step },
              ArrowDown: { x: 0, y: step },
            }[event.key]
            if (!delta) return
            event.preventDefault()
            move(index, {
              x: clamp(corner.x + delta.x),
              y: clamp(corner.y + delta.y),
            })
          }}
        />
      ))}
    </div>
  )
}

function clamp(value: number) {
  return Math.min(1, Math.max(0, value))
}
