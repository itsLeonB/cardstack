import { useState } from "react"
import type { ReactNode } from "react"
import { Link } from "@tanstack/react-router"
import type { CardSummary } from "@/generated/models"
import { Badge } from "@/components/ui/badge"
import { Card, CardContent } from "@/components/ui/card"
import { imageSources } from "@/lib/image"

// Intrinsic sizes match the requested widths at the card's 5:7 ratio, so layout is reserved before the image loads.
const IMAGE_SIZE = {
  tile: { variant: "cardTile", width: 240, height: 336 },
  detail: { variant: "cardDetail", width: 720, height: 1008 },
} as const

/** One Card in a browse/search results grid: art, name, set/number, rarity, category, and tags. `imageVariant="detail"` asks for the larger image the detail view shows. */
export function CardTile({
  card,
  control,
  imageVariant = "tile",
}: {
  card: CardSummary
  control?: ReactNode
  imageVariant?: keyof typeof IMAGE_SIZE
}) {
  const [imageFailed, setImageFailed] = useState(false)
  const { variant, width, height } = IMAGE_SIZE[imageVariant]
  const { src, srcSet } = imageSources(card.imageUrl, variant)
  const tags = card.tags ?? []

  return (
    <Card size="sm" className="h-full">
      {card.imageUrl && !imageFailed ? (
        <img
          src={src}
          srcSet={srcSet}
          width={width}
          height={height}
          alt={card.name}
          loading="lazy"
          className="aspect-[5/7] w-full object-cover"
          onError={() => setImageFailed(true)}
        />
      ) : (
        <div className="flex aspect-[5/7] w-full items-center justify-center rounded-t-4xl bg-muted p-3 text-center text-sm text-muted-foreground">
          {card.name}
        </div>
      )}
      <CardContent className="flex flex-col gap-1.5">
        <p className="truncate text-sm font-medium" title={card.name}>
          <Link
            to="/catalog/cards/$expansionSetId/$localId"
            params={{
              expansionSetId: card.expansionSet.id,
              localId: card.localId,
            }}
            className="underline-offset-2 hover:underline"
          >
            {card.name}
          </Link>
        </p>
        <p className="text-xs text-muted-foreground">
          <Link
            to="/catalog/sets/$expansionSetId"
            params={{ expansionSetId: card.expansionSet.id }}
            className="underline-offset-2 hover:underline"
          >
            {card.expansionSet.code}
          </Link>{" "}
          · No. {card.localId}
        </p>
        <div className="flex flex-wrap gap-1">
          <Badge variant="secondary">{card.rarity.name}</Badge>
          <Badge variant="outline">{card.category}</Badge>
        </div>
        {tags.length > 0 && (
          <div className="flex flex-wrap gap-1 pt-0.5">
            {tags.map((tag) => (
              <Badge
                key={tag}
                variant="outline"
                className="text-muted-foreground"
              >
                {tag}
              </Badge>
            ))}
          </div>
        )}
        {control}
      </CardContent>
    </Card>
  )
}
