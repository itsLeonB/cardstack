import { createFileRoute, Link, useNavigate } from "@tanstack/react-router"
import { z } from "zod"
import {
  getSearchCatalogCardsQueryOptions,
  useSearchCatalogCards,
} from "@/generated/endpoints/catalog/catalog"
import { CardResults } from "@/components/catalog/card-results"
import { Skeleton } from "@/components/ui/skeleton"
import { formatReleaseDate } from "@/lib/date"

const expansionSetSearchSchema = z.object({
  page: z.number().int().min(1).default(1),
})

export const Route = createFileRoute("/catalog/sets/$expansionSetId")({
  validateSearch: expansionSetSearchSchema,
  loaderDeps: ({ search }) => ({ page: search.page }),
  loader: ({ context: { queryClient }, params, deps }) =>
    queryClient.ensureQueryData(
      getSearchCatalogCardsQueryOptions({
        expansionSetId: params.expansionSetId,
        page: deps.page,
      })
    ),
  component: ExpansionSetCardsPage,
})

function ExpansionSetCardsPage() {
  const { expansionSetId } = Route.useParams()
  const { page } = Route.useSearch()
  const navigate = useNavigate({ from: Route.fullPath })

  const query = useSearchCatalogCards({ expansionSetId, page })
  const result = query.data?.status === 200 ? query.data.data : undefined
  const cards = result?.data ?? []
  const firstCard = cards[0]
  const releaseDate = formatReleaseDate(firstCard?.expansionSet.releaseDate)
  const errorMessage =
    query.data && query.data.status !== 200
      ? (query.data.data.detail ?? "Could not load this Expansion Set.")
      : undefined

  function handlePageChange(nextPage: number) {
    void navigate({ search: (prev) => ({ ...prev, page: nextPage }) })
  }

  return (
    <main className="mx-auto flex max-w-5xl flex-col gap-6 p-6">
      <Link
        to="/catalog"
        className="w-fit text-sm text-muted-foreground underline-offset-2 hover:underline"
      >
        ← All Series
      </Link>

      <header className="flex flex-col gap-1">
        {firstCard ? (
          <>
            <h1 className="font-heading text-2xl font-medium">
              {firstCard.expansionSet.name}
            </h1>
            <p className="text-sm text-muted-foreground">
              {firstCard.expansionSet.code}
              {releaseDate && ` · Released ${releaseDate}`}
            </p>
          </>
        ) : query.isPending ? (
          <>
            <Skeleton className="h-8 w-64" />
            <Skeleton className="h-4 w-40" />
          </>
        ) : (
          <h1 className="font-heading text-2xl font-medium">Expansion Set</h1>
        )}
      </header>

      {result && result.meta.total > 0 && (
        <Link
          to="/catalog/search"
          search={{ expansionSetId, page: 1 }}
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
    </main>
  )
}
