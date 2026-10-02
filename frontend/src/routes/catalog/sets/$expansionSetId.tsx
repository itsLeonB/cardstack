import { createFileRoute, Link } from "@tanstack/react-router"
import { pageHead } from "@/lib/site"
import { InfiniteCardResults } from "@/components/catalog/infinite-card-results"
import { Breadcrumbs } from "@/components/layout/breadcrumbs"
import { PageContainer } from "@/components/layout/page-container"
import { PageHeader } from "@/components/layout/page-header"
import { Skeleton } from "@/components/ui/skeleton"
import { formatReleaseDate } from "@/lib/date"
import {
  prefetchInfiniteCatalogCards,
  useInfiniteCardResultsProps,
  useInfiniteCatalogCards,
} from "@/lib/infinite-catalog-cards"

export const Route = createFileRoute("/catalog/sets/$expansionSetId")({
  // No search schema: the list is infinite, so a stale `?page=` in an old link
  // is simply ignored and the first page opens.
  loader: async ({ context: { queryClient }, params }) => {
    const data = await prefetchInfiniteCatalogCards(queryClient, {
      expansionSetId: [params.expansionSetId],
    })
    const first = data?.pages[0]
    return first?.status === 200
      ? first.data.data?.[0]?.expansionSet.name
      : undefined
  },
  head: ({ loaderData }) => pageHead(loaderData ?? "Expansion Set"),
  component: ExpansionSetCardsPage,
})

function ExpansionSetCardsPage() {
  const { expansionSetId } = Route.useParams()

  const query = useInfiniteCatalogCards({ expansionSetId: [expansionSetId] })
  const results = useInfiniteCardResultsProps(query)
  const { cards, total } = results
  const firstCard = cards[0]
  const releaseDate = formatReleaseDate(firstCard?.expansionSet.releaseDate)
  const setName = firstCard?.expansionSet.name ?? "Expansion Set"

  return (
    <PageContainer>
      <Breadcrumbs
        crumbs={[
          { label: "Catalog", link: { to: "/catalog" } },
          { label: setName },
        ]}
      />
      {firstCard || !query.isPending ? (
        <PageHeader
          title={setName}
          description={
            firstCard &&
            `${firstCard.expansionSet.code}${releaseDate ? ` · Released ${releaseDate}` : ""}`
          }
        />
      ) : (
        <div className="flex flex-col gap-1">
          <Skeleton className="h-8 w-64" />
          <Skeleton className="h-4 w-40" />
        </div>
      )}

      {total > 0 && (
        <Link
          to="/catalog/search"
          search={{ expansionSetId: [expansionSetId] }}
          className="w-fit text-sm text-primary underline-offset-2 hover:underline"
        >
          Search within this set
        </Link>
      )}

      <InfiniteCardResults
        {...results}
        emptyMessage="This Expansion Set has no cards yet."
      />
    </PageContainer>
  )
}
