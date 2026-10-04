import { useState } from "react"
import { cn } from "cn"
import { Link } from "@tanstack/react-router"
import type { ExpansionSetSummary } from "@/generated/models"
import {
  Card,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"
import { formatReleaseDate } from "@/lib/date"
import { imageSources } from "@/lib/image"

export function ExpansionSetTile({
  expansionSet,
}: {
  expansionSet: ExpansionSetSummary
}) {
  const [imageFailed, setImageFailed] = useState(false)
  const releaseDate = formatReleaseDate(expansionSet.releaseDate)
  const showImage = Boolean(expansionSet.imageUrl) && !imageFailed
  const { src } = imageSources(expansionSet.imageUrl, "setCover")

  return (
    <Link
      to="/catalog/sets/$expansionSetId"
      params={{ expansionSetId: expansionSet.id }}
      className="block rounded-4xl focus-visible:ring-3 focus-visible:ring-ring/30 focus-visible:outline-none"
    >
      <Card
        className={cn(
          "h-full transition-colors hover:bg-muted/50",
          showImage && "flex-row items-center"
        )}
      >
        {showImage && (
          // Wrapped so Card's `img:first-child` rules (zero top padding) don't apply. Decorative (the name sits
          // beside it); fixed width/height reserve the slot so loading doesn't shift the tile.
          <div className="ml-6 shrink-0">
            <img
              src={src}
              alt=""
              width={64}
              height={64}
              loading="lazy"
              className="size-16 object-contain"
              onError={() => setImageFailed(true)}
            />
          </div>
        )}
        <CardHeader className={showImage ? "flex-1" : undefined}>
          <CardTitle>{expansionSet.name}</CardTitle>
          <CardDescription>
            {expansionSet.code}
            {releaseDate && ` · Released ${releaseDate}`}
          </CardDescription>
        </CardHeader>
      </Card>
    </Link>
  )
}
