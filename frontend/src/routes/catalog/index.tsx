import { createFileRoute, Link } from "@tanstack/react-router"
import { pageHead } from "@/lib/site"
import {
  getListCatalogSeriesQueryOptions,
  useListCatalogSeries,
} from "@/generated/endpoints/catalog/catalog"
import type { ExpansionSetSummary } from "@/generated/models"
import { PageContainer } from "@/components/layout/page-container"
import { PageHeader } from "@/components/layout/page-header"
import { ExpansionSetTile } from "@/components/catalog/expansion-set-tile"
import { Skeleton } from "@/components/ui/skeleton"
import { Separator } from "@/components/ui/separator"

export const Route = createFileRoute("/catalog/")({
  head: () =>
    pageHead(
      "Catalog",
      "Browse every Pokémon TCG series and set in the Cardstack catalog. No account needed."
    ),
  loader: ({ context: { queryClient } }) =>
    queryClient.ensureQueryData(getListCatalogSeriesQueryOptions()),
  component: CatalogSeriesPage,
})

function CatalogSeriesPage() {
  const { data, isPending, isError } = useListCatalogSeries()
  const seriesBrowseResult = data?.status === 200 ? data.data.data : undefined
  const series = seriesBrowseResult?.series ?? []
  const ungroupedExpansionSets = seriesBrowseResult?.ungroupedExpansionSets ?? []
  const errorMessage =
    data && data.status !== 200
      ? (data.data.detail ?? "Could not load the catalog.")
      : null

  return (
    <PageContainer>
      <PageHeader
        title="Catalog"
        description={
          <>
            Browse every Series and Expansion Set, or{" "}
            <Link to="/catalog/search" className="text-primary underline">
              search the catalog
            </Link>
            .
          </>
        }
      />

      {isPending && (
        <div className="flex flex-col gap-4" aria-busy="true" aria-label="Loading catalog">
          {Array.from({ length: 3 }).map((_, index) => (
            // oxlint-disable-next-line no-array-index-key -- fixed-size skeleton placeholders, never reordered
            <Skeleton key={index} className="h-24 w-full" />
          ))}
        </div>
      )}

      {(isError || errorMessage) && (
        <p role="alert" className="text-sm text-destructive">
          {errorMessage ?? "Could not reach the backend. Please try again."}
        </p>
      )}

      {!isPending &&
        !isError &&
        !errorMessage &&
        series.length === 0 &&
        ungroupedExpansionSets.length === 0 && (
          <p className="text-sm text-muted-foreground">No Series found yet.</p>
        )}

      <div className="flex flex-col gap-8">
        {series.map((oneSeries) => (
          <section
            key={oneSeries.id}
            aria-labelledby={`series-${oneSeries.id}`}
            className="flex flex-col gap-3"
          >
            <h2
              id={`series-${oneSeries.id}`}
              className="font-heading text-lg font-medium"
            >
              {oneSeries.name}
            </h2>
            <Separator />
            <ExpansionSetList expansionSets={oneSeries.expansionSets ?? []} />
          </section>
        ))}

        {ungroupedExpansionSets.length > 0 && (
          <section
            aria-labelledby="series-ungrouped"
            className="flex flex-col gap-3"
          >
            <h2 id="series-ungrouped" className="font-heading text-lg font-medium">
              Ungrouped Expansion Sets
            </h2>
            <Separator />
            <ExpansionSetList expansionSets={ungroupedExpansionSets} />
          </section>
        )}
      </div>
    </PageContainer>
  )
}

function ExpansionSetList({
  expansionSets,
}: {
  expansionSets: ExpansionSetSummary[]
}) {
  return (
    <ul className="grid gap-3 sm:grid-cols-2">
      {expansionSets.map((expansionSet) => (
        <li key={expansionSet.id}>
          <ExpansionSetTile expansionSet={expansionSet} />
        </li>
      ))}
    </ul>
  )
}
