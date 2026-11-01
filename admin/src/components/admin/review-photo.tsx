import { useEffect, useRef, useState } from "react"
import { Button } from "@/components/ui/button"
import { useGetReviewPhoto } from "@/generated/endpoints/admin/admin"
import type { ReviewBinderCell, ReviewItem } from "@/generated/models"
import { binderCellCut } from "@/lib/binder-cell"
import { paintPixels, readPixels } from "@/lib/photo-pixels"

/**
 * The user's photo for one item, read through its short-lived presigned
 * address. A card scan is shown as stored; a binder cell is cut from its Page
 * photo as the matcher would embed it. A stored address that has expired fails
 * to load, so the photo is asked for again, once: a fresh address that fails too
 * shows Retry instead of asking again and again.
 */
export function ReviewPhoto({ item }: { item: ReviewItem }) {
  const photo = useGetReviewPhoto(item.id)
  const [load, setLoad] = useState<"first" | "refetched" | "failed">("first")
  const url = photo.data?.status === 200 ? photo.data.data.data.url : undefined

  if (!url) {
    if (photo.isPending) return <p role="status">Loading the photo…</p>
    return <RetryPhoto onRetry={() => photo.refetch()} />
  }
  if (item.source === "card_scan") {
    if (load === "failed") {
      return (
        <RetryPhoto
          onRetry={() => {
            setLoad("first")
            void photo.refetch()
          }}
        />
      )
    }
    return (
      <img
        src={url}
        alt="Photo from the scan"
        onLoad={() => setLoad("first")}
        onError={() => {
          if (load === "refetched") return setLoad("failed")
          setLoad("refetched")
          void photo.refetch()
        }}
        className="max-h-96 rounded-md object-contain"
      />
    )
  }
  if (item.binder === undefined) {
    return <p>This Correction has no binder cell to cut.</p>
  }
  return (
    <BinderCellCut
      url={url}
      binder={item.binder}
      onRetry={() => photo.refetch()}
    />
  )
}

type CutState = "cutting" | "ready" | "no-corners" | "failed"

function BinderCellCut({
  url,
  binder,
  onRetry,
}: {
  url: string
  binder: ReviewBinderCell
  onRetry: () => void
}) {
  const canvas = useRef<HTMLCanvasElement>(null)
  const [state, setState] = useState<CutState>("cutting")

  useEffect(() => {
    let current = true
    setState("cutting")
    readPixels(url)
      .then((photo) => {
        if (!current || !canvas.current) return
        const cut = binderCellCut(photo, binder)
        if (!cut) return setState("no-corners")
        paintPixels(canvas.current, cut)
        setState("ready")
      })
      .catch(() => {
        if (current) setState("failed")
      })
    return () => {
      current = false
    }
  }, [url, binder])

  return (
    <div className="flex flex-col gap-2">
      <canvas
        ref={canvas}
        role="img"
        aria-label="The binder cell, cut as the matcher would embed it"
        hidden={state !== "ready"}
        className="max-h-96 w-auto max-w-full rounded-md"
      />
      {state === "cutting" && <p role="status">Cutting the cell…</p>}
      {state === "no-corners" && <p>This cell has no corners to cut.</p>}
      {state === "failed" && <RetryPhoto onRetry={onRetry} />}
    </div>
  )
}

function RetryPhoto({ onRetry }: { onRetry: () => void }) {
  return (
    <div role="alert" className="flex flex-col items-start gap-2">
      <p>The photo could not be loaded.</p>
      <Button variant="outline" size="sm" onClick={onRetry}>
        Retry
      </Button>
    </div>
  )
}
