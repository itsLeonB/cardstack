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
