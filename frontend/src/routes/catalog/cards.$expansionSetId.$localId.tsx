import { createFileRoute } from "@tanstack/react-router"
import { pageHead } from "@/lib/site"
import {
  getSearchCatalogCardsQueryOptions,
  useSearchCatalogCards,
} from "@/generated/endpoints/catalog/catalog"
import { Breadcrumbs } from "@/components/layout/breadcrumbs"
import { notFoundResource } from "@/components/layout/not-found"
import { PageContainer } from "@/components/layout/page-container"
import { PageHeader } from "@/components/layout/page-header"
import { CardHoldings } from "@/components/catalog/card-holdings"
import { CardTile } from "@/components/catalog/card-tile"
import { Skeleton } from "@/components/ui/skeleton"

// No single-card endpoint: (set, localId) is unique, so a filtered search finds it.
const cardParams = (expansionSetId: string, localId: string) => ({
  expansionSetId: [expansionSetId],
  localId,
  limit: 1,
})

export const Route = createFileRoute("/catalog/cards/$expansionSetId/$localId")(
  {
    loader: async ({ context: { queryClient }, params }) => {
      const response = await queryClient.ensureQueryData(
        getSearchCatalogCardsQueryOptions(
          cardParams(params.expansionSetId, params.localId)
        )
      )
      const card = response.status === 200 ? response.data.data?.[0] : undefined
      if (response.status === 200 && !card) throw notFoundResource("Card")
      return card?.name
    },
    head: ({ loaderData }) => pageHead(loaderData ?? "Card"),
    component: CardDetailPage,
  }
)

function CardDetailPage() {
  const { expansionSetId, localId } = Route.useParams()
  const query = useSearchCatalogCards(cardParams(expansionSetId, localId))
  const card =
    query.data?.status === 200 ? query.data.data.data?.[0] : undefined
  const cardName = card?.name ?? "Card"
  const setName = card?.expansionSet.name ?? "Expansion Set"
  const failed =
    query.isError || (query.data !== undefined && query.data.status !== 200)

  return (
    <PageContainer>
      <Breadcrumbs
        crumbs={[
          { label: "Catalog", link: { to: "/catalog" } },
          {
            label: setName,
            link: {
              to: "/catalog/sets/$expansionSetId",
              params: { expansionSetId },
              search: { page: 1 },
            },
          },
          { label: cardName },
        ]}
      />
      <PageHeader title={cardName} />
      {query.isPending ? (
        <Skeleton
          className="aspect-[5/7] w-full max-w-xs rounded-4xl"
          aria-label="Loading card"
        />
      ) : failed ? (
        <p role="alert" className="text-sm text-destructive">
          Could not load this Card. Please try again.
        </p>
      ) : !card ? (
        <p className="text-sm text-muted-foreground">
          We couldn&apos;t find that Card.
        </p>
      ) : (
        <div className="grid gap-6 sm:grid-cols-[minmax(0,20rem)_1fr]">
          <CardTile card={card} />
          <CardHoldings cardId={card.id} />
        </div>
      )}
    </PageContainer>
  )
}
