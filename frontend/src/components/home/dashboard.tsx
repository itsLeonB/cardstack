import { Link } from "@tanstack/react-router"
import { useListCollections } from "@/generated/endpoints/collections/collections"
import { useListMasterInventory } from "@/generated/endpoints/inventory/inventory"
import { PageContainer } from "@/components/layout/page-container"
import { PageHeader } from "@/components/layout/page-header"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Skeleton } from "@/components/ui/skeleton"
import { errorDetail } from "@/lib/collections"
import { useSession } from "@/lib/session"

const MAX_COLLECTIONS = 5

/** Home for signed-in users: their Collections, a Master Inventory total and shortcuts. */
export function Dashboard() {
  const { user } = useSession()

  return (
    <PageContainer>
      <PageHeader
        title="Welcome back"
        description={user ? `Signed in as ${user.email}` : undefined}
      />
      <MasterInventorySummary />
      <CollectionsSummary />
      <section className="flex flex-col gap-3" aria-labelledby="quick-actions">
        <h2 id="quick-actions" className="font-heading text-lg font-medium">
          Quick actions
        </h2>
        <div className="flex flex-col gap-3 sm:flex-row">
          <Button variant="outline" render={<Link to="/catalog/search" />}>
            Search cards
          </Button>
          <Button variant="outline" render={<Link to="/catalog" />}>
            Browse the catalog
          </Button>
          <Button variant="outline" render={<Link to="/inventory" />}>
            Open Master Inventory
          </Button>
        </div>
      </section>
    </PageContainer>
  )
}

function MasterInventorySummary() {
  // limit 1: only `meta.total` is read. That total counts distinct cards (the
  // backend groups by card and sums quantity), so it is not a quantity total.
  const query = useListMasterInventory(
    { page: 1, limit: 1 },
    { query: { refetchOnMount: "always" } }
  )
  const total =
    query.data?.status === 200 ? query.data.data.meta.total : undefined
  const failed =
    query.isError ||
    errorDetail(query.data, "Could not load your Master Inventory.") !==
      undefined

  return (
    <Card>
      <CardHeader>
        <CardTitle>
          <h2>Master Inventory</h2>
        </CardTitle>
      </CardHeader>
      <CardContent>
        {query.isPending && (
          <Skeleton
            className="h-9 w-24"
            role="status"
            aria-busy="true"
            aria-label="Loading Master Inventory total"
          />
        )}
        {total !== undefined && (
          <p>
            <span className="font-heading text-3xl font-medium">{total}</span>{" "}
            <span className="text-sm">distinct cards</span>
          </p>
        )}
        {failed && (
          <LoadError
            message="Could not load your Master Inventory total."
            onRetry={() => void query.refetch()}
          />
        )}
      </CardContent>
    </Card>
  )
}

/** Inline page-data failure: message plus a Retry that refetches the query. */
function LoadError({
  message,
  onRetry,
}: {
  message: string
  onRetry: () => void
}) {
  return (
    <div className="flex flex-wrap items-center gap-3">
      <p role="alert" className="text-sm text-destructive">
        {message}
      </p>
      <Button size="sm" variant="outline" onClick={onRetry}>
        Retry
      </Button>
    </div>
  )
}

function CollectionsSummary() {
  const query = useListCollections()
  const collections =
    query.data?.status === 200 ? (query.data.data.data ?? []) : []
  const failed =
    query.isError ||
    errorDetail(query.data, "Could not load your Collections.") !== undefined

  return (
    <section
      className="flex flex-col gap-3"
      aria-labelledby="collections-summary"
    >
      <div className="flex items-center justify-between gap-3">
        <h2
          id="collections-summary"
          className="font-heading text-lg font-medium"
        >
          Your Collections
        </h2>
        <Button size="sm" render={<Link to="/collections/new" />}>
          New collection
        </Button>
      </div>

      {query.isPending && (
        <Skeleton
          className="h-16 w-full"
          role="status"
          aria-busy="true"
          aria-label="Loading Collections"
        />
      )}

      {failed && (
        <LoadError
          message="Could not load your Collections."
          onRetry={() => void query.refetch()}
        />
      )}

      {!query.isPending && !failed && collections.length === 0 && (
        <p className="text-sm">
          Start by creating a Collection for a binder, box or deck. Then add
          Cards to it from the Catalog and they will show up in your Master
          Inventory.
        </p>
      )}

      {collections.length > 0 && (
        <ul className="flex flex-col gap-2">
          {collections.slice(0, MAX_COLLECTIONS).map((collection) => (
            <li key={collection.id}>
              <Link
                to="/collections/$collectionId"
                params={{ collectionId: collection.id }}
                className="block rounded-2xl bg-card px-4 py-3 font-medium ring-1 ring-foreground/5 hover:bg-muted"
              >
                {collection.title}
              </Link>
            </li>
          ))}
        </ul>
      )}
      {collections.length > MAX_COLLECTIONS && (
        <Link
          to="/collections"
          className="w-fit text-sm text-primary underline"
        >
          View all Collections
        </Link>
      )}
    </section>
  )
}
