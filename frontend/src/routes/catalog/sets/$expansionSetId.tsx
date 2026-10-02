import { createFileRoute, Link, useNavigate } from "@tanstack/react-router"
import { pageHead } from "@/lib/site"
import { z } from "zod"
import {
  getSearchCatalogCardsQueryOptions,
  useSearchCatalogCards,
} from "@/generated/endpoints/catalog/catalog"
import { CardResults } from "@/components/catalog/card-results"
import { Breadcrumbs } from "@/components/layout/breadcrumbs"
import { PageContainer } from "@/components/layout/page-container"
import { PageHeader } from "@/components/layout/page-header"
import { Skeleton } from "@/components/ui/skeleton"
import { formatReleaseDate } from "@/lib/date"

const expansionSetSearchSchema = z.object({
  page: z.number().int().min(1).default(1),
})

export const Route = createFileRoute("/catalog/sets/$expansionSetId")({
  validateSearch: expansionSetSearchSchema,
  loaderDeps: ({ search }) => ({ page: search.page }),
  loader: async ({ context: { queryClient }, params, deps }) => {
    const response = await queryClient.ensureQueryData(
      getSearchCatalogCardsQueryOptions({
        expansionSetId: [params.expansionSetId],
        page: deps.page,
      })
    )
    return response.status === 200
      ? response.data.data?.[0]?.expansionSet.name
      : undefined
  },
  head: ({ loaderData }) => pageHead(loaderData ?? "Expansion Set"),
  component: ExpansionSetCardsPage,
})

function ExpansionSetCardsPage() {
  const { expansionSetId } = Route.useParams()
  const { page } = Route.useSearch()
  const navigate = useNavigate({ from: Route.fullPath })

  const query = useSearchCatalogCards({
    expansionSetId: [expansionSetId],
    page,
  })
  const result = query.data?.status === 200 ? query.data.data : undefined
  const cards = result?.data ?? []
  const firstCard = cards[0]
  const releaseDate = formatReleaseDate(firstCard?.expansionSet.releaseDate)
  const setName = firstCard?.expansionSet.name ?? "Expansion Set"
  const errorMessage =
    query.data && query.data.status !== 200
      ? (query.data.data.detail ?? "Could not load this Expansion Set.")
      : undefined

  function handlePageChange(nextPage: number) {
    void navigate({ search: (prev) => ({ ...prev, page: nextPage }) })
  }

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

      {result && result.meta.total > 0 && (
        <Link
          to="/catalog/search"
          search={{ expansionSetId: [expansionSetId] }}
          className="w-fit text-sm text-primary underline-offset-2 hover:underline"
        >
          Search within this set
        </Link>
      )}

      <CardResults
        cards={cards}
        total={result?.meta.total ?? 0}
        page={page}
        limit={result?.meta.limit ?? 24}
        isPending={query.isPending}
        isError={query.isError || Boolean(errorMessage)}
        errorMessage={errorMessage}
        emptyMessage="This Expansion Set has no cards yet."
        onPageChange={handlePageChange}
      />
    </PageContainer>
  )
}
