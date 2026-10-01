import { createFileRoute, Link } from "@tanstack/react-router"
import {
  getSearchCatalogCardsQueryOptions,
  useSearchCatalogCards,
} from "@/generated/endpoints/catalog/catalog"
import { CardHoldings } from "@/components/catalog/card-holdings"
import { CardTile } from "@/components/catalog/card-tile"
import { Skeleton } from "@/components/ui/skeleton"

// No single-card endpoint: (set, localId) is unique, so a filtered search finds it.
const cardParams = (expansionSetId: string, localId: string) => ({
  expansionSetId: [expansionSetId],
  localId,
  limit: 1,
})

export const Route = createFileRoute("/catalog/cards/$expansionSetId/$localId")({
  loader: ({ context: { queryClient }, params }) =>
    queryClient.ensureQueryData(
      getSearchCatalogCardsQueryOptions(cardParams(params.expansionSetId, params.localId))
    ),
  component: CardDetailPage,
})

function CardDetailPage() {
  const { expansionSetId, localId } = Route.useParams()
  const query = useSearchCatalogCards(cardParams(expansionSetId, localId))
  const card = query.data?.status === 200 ? query.data.data.data?.[0] : undefined
  const failed = query.isError || (query.data !== undefined && query.data.status !== 200)

  return (
    <main className="mx-auto flex max-w-3xl flex-col gap-6 p-6">
      <Link
        to="/catalog/sets/$expansionSetId"
        params={{ expansionSetId }}
        search={{ page: 1 }}
        className="w-fit text-sm text-muted-foreground underline-offset-2 hover:underline"
      >
        ← Back to set
      </Link>
      {query.isPending ? (
        <Skeleton className="aspect-[5/7] w-full max-w-xs rounded-4xl" aria-label="Loading card" />
      ) : failed ? (
        <p role="alert" className="text-sm text-destructive">
          Could not load this Card. Please try again.
        </p>
      ) : !card ? (
        <p className="text-sm text-muted-foreground">We couldn&apos;t find that Card.</p>
      ) : (
        <div className="grid gap-6 sm:grid-cols-[minmax(0,20rem)_1fr]">
          <CardTile card={card} />
          <CardHoldings cardId={card.id} />
        </div>
      )}
    </main>
  )
}
