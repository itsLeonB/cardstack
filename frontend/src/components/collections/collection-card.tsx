import { Link } from "@tanstack/react-router"
import type { CollectionSummary } from "@/generated/models"
import {
  Card,
  CardAction,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"
import { Button } from "@/components/ui/button"
import { DeleteCollectionDialog } from "@/components/collections/delete-collection-dialog"

export function cardCountLabel(count: number) {
  return `${count} ${count === 1 ? "card" : "cards"}`
}

/**
 * The title link's `after` overlay stretches over the whole card, so a click
 * anywhere navigates while the card keeps one accessible name (the title).
 * Actions sit above the overlay (`relative z-10`) so they stay independent.
 */
export function CollectionCard({
  collection,
}: {
  collection: CollectionSummary
}) {
  return (
    <Card className="relative transition-shadow hover:shadow-lg">
      <CardHeader>
        <CardTitle>
          <Link
            to="/collections/$collectionId"
            params={{ collectionId: collection.id }}
            className="outline-none after:absolute after:inset-0 after:rounded-4xl focus-visible:after:ring-2 focus-visible:after:ring-ring focus-visible:after:ring-inset"
          >
            {collection.title}
          </Link>
        </CardTitle>
        {collection.description && (
          <CardDescription>{collection.description}</CardDescription>
        )}
        <CardDescription>
          {cardCountLabel(collection.cardCount)}
        </CardDescription>
        {collection.maxCardCount > 0 && (
          <CardDescription>
            Limit: {collection.maxCardCount} cards
          </CardDescription>
        )}
        <CardAction className="relative z-10 flex items-center gap-2">
          <Button
            variant="outline"
            size="sm"
            render={
              <Link
                to="/collections/$collectionId/edit"
                params={{ collectionId: collection.id }}
              />
            }
          >
            Edit
          </Button>
          <DeleteCollectionDialog
            collectionId={collection.id}
            collectionTitle={collection.title}
          />
        </CardAction>
      </CardHeader>
    </Card>
  )
}
