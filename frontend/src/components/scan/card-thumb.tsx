import { useEffect, useState } from "react"
import type { ReactNode } from "react"
import type { CardSummary } from "@/generated/models"
import { imageSources } from "@/lib/image"

/** A small card image at the card's 5:7 ratio, or a blank of the same size when the card has none (or is unknown). */
export function CardThumb({ card }: { card?: CardSummary }) {
  return card?.imageUrl ? (
    <img
      src={imageSources(card.imageUrl, "cardTile").src}
      alt=""
      width={48}
      height={67}
      className="aspect-[5/7] w-12 rounded object-cover"
    />
  ) : (
    <div className="aspect-[5/7] w-12 shrink-0 rounded bg-muted" />
  )
}

/** The user's own photo in the card's slot, for a Scan that is still matching or has failed. `children` sit over it. */
export function PhotoThumb({
  photo,
  children,
}: {
  photo: Blob
  children?: ReactNode
}) {
  const [url, setUrl] = useState<string>()
  useEffect(() => {
    const next = URL.createObjectURL(photo)
    setUrl(next)
    return () => URL.revokeObjectURL(next)
  }, [photo])
  return (
    <div className="relative aspect-[5/7] w-12 shrink-0 overflow-hidden rounded bg-muted">
      {url && <img src={url} alt="" className="size-full object-cover" />}
      {children}
    </div>
  )
}
