import { createFileRoute, Link } from "@tanstack/react-router"
import { pageHead } from "@/lib/site"
import {
  getListCollectionsQueryOptions,
  useListCollections,
} from "@/generated/endpoints/collections/collections"
import { PageContainer } from "@/components/layout/page-container"
import { PageHeader } from "@/components/layout/page-header"
import { Button } from "@/components/ui/button"
import { Skeleton } from "@/components/ui/skeleton"
import { errorDetail } from "@/lib/collections"
import { CollectionCard } from "@/components/collections/collection-card"

export const Route = createFileRoute("/_authenticated/collections/")({
  head: () => pageHead("Collections"),
  loader: ({ context: { queryClient } }) =>
    queryClient.ensureQueryData(getListCollectionsQueryOptions()),
  component: CollectionsPage,
})

function CollectionsPage() {
  const query = useListCollections()
  const collections =
    query.data?.status === 200 ? (query.data.data.data ?? []) : []
  const errorMessage = errorDetail(
    query.data,
    "Could not load your Collections."
  )

  return (
    <PageContainer>
      <PageHeader
        title="Collections"
        actions={
          <Button render={<Link to="/collections/new" />}>
            New Collection
          </Button>
        }
      />

      {query.isPending && (
        <div
          className="flex flex-col gap-3"
          aria-busy="true"
          aria-label="Loading Collections"
        >
          {Array.from({ length: 3 }).map((_, index) => (
            // oxlint-disable-next-line no-array-index-key -- fixed-size skeleton placeholders, never reordered
            <Skeleton key={index} className="h-20 w-full" />
          ))}
        </div>
      )}

      {(query.isError || errorMessage) && (
        <p role="alert" className="text-sm text-destructive">
          {errorMessage ?? "Could not reach the backend. Please try again."}
        </p>
      )}

      {!query.isPending &&
        !query.isError &&
        !errorMessage &&
        collections.length === 0 && (
          <p className="text-sm text-muted-foreground">
            You don&apos;t have any Collections yet.
          </p>
        )}

      <ul className="flex flex-col gap-3">
        {collections.map((collection) => (
          <li key={collection.id}>
            <CollectionCard collection={collection} />
          </li>
        ))}
      </ul>
    </PageContainer>
  )
}
