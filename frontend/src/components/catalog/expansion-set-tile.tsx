import { useState } from "react"
import { Link } from "@tanstack/react-router"
import type { ExpansionSetSummary } from "@/generated/models"
import { Card, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { formatReleaseDate } from "@/lib/date"

/** One Expansion Set in the catalog browse grid: optional cover image beside name, code and release date. */
export function ExpansionSetTile({ expansionSet }: { expansionSet: ExpansionSetSummary }) {
  const [imageFailed, setImageFailed] = useState(false)
  const releaseDate = formatReleaseDate(expansionSet.releaseDate)

  return (
    <Link
      to="/catalog/sets/$expansionSetId"
      params={{ expansionSetId: expansionSet.id }}
      className="block rounded-4xl focus-visible:outline-none focus-visible:ring-3 focus-visible:ring-ring/30"
    >
      <Card className="h-full flex-row items-center transition-colors hover:bg-muted/50">
        {expansionSet.imageUrl && !imageFailed && (
          // Wrapped so Card's `img:first-child` rules (zero top padding) don't apply. Decorative (the name sits
          // beside it); fixed width/height reserve the slot so loading doesn't shift the tile.
          <div className="ml-6 shrink-0">
            <img
              src={expansionSet.imageUrl}
              alt=""
              width={64}
              height={64}
              loading="lazy"
              className="size-16 object-contain"
              onError={() => setImageFailed(true)}
            />
          </div>
        )}
        <CardHeader className="flex-1">
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
