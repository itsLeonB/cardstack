import { createFileRoute, Link, useNavigate } from "@tanstack/react-router"
import {
  getGetCollectionQueryOptions,
  useGetCollection,
} from "@/generated/endpoints/collections/collections"
import { CollectionEntries } from "@/components/collections/collection-entries"
import { catalogSearchSchema } from "@/lib/catalog-search"
import { errorDetail, NETWORK_ERROR } from "@/lib/collections"

export const Route = createFileRoute("/_authenticated/collections/$collectionId/")({
  validateSearch: catalogSearchSchema,
  // Entries are deliberately not prefetched into the cache: the page refetches
  // them on mount so a card removed earlier doesn't reappear from stale data.
  loader: ({ context: { queryClient }, params }) =>
    queryClient.ensureQueryData(getGetCollectionQueryOptions(params.collectionId)),
  component: CollectionPage,
})

function CollectionPage() {
  const { collectionId } = Route.useParams()
  const search = Route.useSearch()
  const navigate = useNavigate({ from: Route.fullPath })
  const query = useGetCollection(collectionId)
  const collection = query.data?.status === 200 ? query.data.data.data : undefined
  const loadError = errorDetail(query.data, "Could not load this Collection.")

  return (
    <main className="mx-auto flex max-w-5xl flex-col gap-6 p-6">
      <Link
        to="/collections"
        className="w-fit text-sm text-muted-foreground underline-offset-2 hover:underline"
      >
        ← Back to Collections
      </Link>
      {(query.isError || loadError) && (
        <p role="alert" className="text-sm text-destructive">
          {loadError ?? NETWORK_ERROR}
        </p>
      )}
      {collection && (
        <>
          <header>
            <h1 className="font-heading text-2xl font-medium">{collection.title}</h1>
            {collection.description && (
              <p className="text-sm text-muted-foreground">{collection.description}</p>
            )}
            {collection.maxCardCount > 0 && (
              <p className="text-sm text-muted-foreground">
                Limit: {collection.maxCardCount} cards
              </p>
            )}
          </header>
          <CollectionEntries
            collectionId={collectionId}
            search={search}
            onSearchChange={(next) => void navigate({ search: next })}
          />
        </>
      )}
    </main>
  )
}
