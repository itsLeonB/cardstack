import { createFileRoute, Link } from "@tanstack/react-router"
import {
  getListCatalogSeriesQueryOptions,
  useListCatalogSeries,
} from "@/generated/endpoints/catalog/catalog"
import { Card, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Skeleton } from "@/components/ui/skeleton"
import { Separator } from "@/components/ui/separator"
import { formatReleaseDate } from "@/lib/catalog"

export const Route = createFileRoute("/catalog/")({
  loader: ({ context: { queryClient } }) =>
    queryClient.ensureQueryData(getListCatalogSeriesQueryOptions()),
  component: CatalogSeriesPage,
})

function CatalogSeriesPage() {
  const { data, isPending, isError } = useListCatalogSeries()
  const series = data?.status === 200 ? (data.data.data ?? []) : []
  const errorMessage =
    data && data.status !== 200
      ? (data.data.detail ?? "Could not load the catalog.")
      : null

  return (
    <main className="mx-auto flex max-w-4xl flex-col gap-8 p-6">
      <header className="flex flex-col gap-2">
        <h1 className="font-heading text-2xl font-medium">Catalog</h1>
        <p className="text-sm text-muted-foreground">
          Browse every Series and Expansion Set, or{" "}
          <Link to="/catalog/search" className="text-primary underline">
            search the catalog
          </Link>
          .
        </p>
      </header>

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

      {!isPending && !isError && !errorMessage && series.length === 0 && (
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
            <ul className="grid gap-3 sm:grid-cols-2">
              {(oneSeries.expansionSets ?? []).map((expansionSet) => {
                const releaseDate = formatReleaseDate(expansionSet.releaseDate)
                return (
                  <li key={expansionSet.id}>
                    <Link
                      to="/catalog/sets/$expansionSetId"
                      params={{ expansionSetId: expansionSet.id }}
                      className="block rounded-4xl focus-visible:outline-none focus-visible:ring-3 focus-visible:ring-ring/30"
                    >
                      <Card className="h-full transition-colors hover:bg-muted/50">
                        <CardHeader>
                          <CardTitle>{expansionSet.name}</CardTitle>
                          <CardDescription>
                            {expansionSet.code}
                            {releaseDate && ` · Released ${releaseDate}`}
                          </CardDescription>
                        </CardHeader>
                      </Card>
                    </Link>
                  </li>
                )
              })}
            </ul>
          </section>
        ))}
      </div>
    </main>
  )
}
