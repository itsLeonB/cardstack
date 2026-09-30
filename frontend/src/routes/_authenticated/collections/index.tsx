import { createFileRoute, Link } from "@tanstack/react-router"
import {
  getListCollectionsQueryOptions,
  useListCollections,
} from "@/generated/endpoints/collections/collections"
import {
  Card,
  CardAction,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"
import { Button } from "@/components/ui/button"
import { Skeleton } from "@/components/ui/skeleton"
import { DeleteCollectionDialog } from "@/components/collections/delete-collection-dialog"

export const Route = createFileRoute("/_authenticated/collections/")({
  loader: ({ context: { queryClient } }) =>
    queryClient.ensureQueryData(getListCollectionsQueryOptions()),
  component: CollectionsPage,
})

function CollectionsPage() {
  const query = useListCollections()
  const collections =
    query.data?.status === 200 ? (query.data.data.data ?? []) : []
  const errorMessage =
    query.data && query.data.status !== 200
      ? (query.data.data.detail ?? "Could not load your Collections.")
      : undefined

  return (
    <main className="mx-auto flex max-w-3xl flex-col gap-6 p-6">
      <header className="flex items-center justify-between gap-4">
        <h1 className="font-heading text-2xl font-medium">Collections</h1>
        <Button render={<Link to="/collections/new" />}>
          New Collection
        </Button>
      </header>

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
            <Card>
              <CardHeader>
                <CardTitle>{collection.title}</CardTitle>
                {collection.description && (
                  <CardDescription>{collection.description}</CardDescription>
                )}
                {collection.maxCardCount > 0 && (
                  <CardDescription>
                    Limit: {collection.maxCardCount} cards
                  </CardDescription>
                )}
                <CardAction className="flex items-center gap-2">
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
          </li>
        ))}
      </ul>
    </main>
  )
}
