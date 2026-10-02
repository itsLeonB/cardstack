import { createFileRoute, useNavigate } from "@tanstack/react-router"
import { Breadcrumbs } from "@/components/layout/breadcrumbs"
import { notFoundResource } from "@/components/layout/not-found"
import { PageContainer } from "@/components/layout/page-container"
import { PageHeader } from "@/components/layout/page-header"
import {
  getGetCollectionQueryOptions,
  useGetCollection,
} from "@/generated/endpoints/collections/collections"
import { AddCardsLink } from "@/components/collections/add-cards-link"
import { CollectionEntries } from "@/components/collections/collection-entries"
import { catalogSearchSchema } from "@/lib/catalog-search"
import { errorDetail, NETWORK_ERROR } from "@/lib/collections"

export const Route = createFileRoute("/_authenticated/collections/$collectionId/")({
  validateSearch: catalogSearchSchema,
  // Entries are deliberately not prefetched into the cache: the page refetches
  // them on mount so a card removed earlier doesn't reappear from stale data.
  loader: async ({ context: { queryClient }, params }) => {
    const response = await queryClient.ensureQueryData(
      getGetCollectionQueryOptions(params.collectionId)
    )
    if (response.status === 404) throw notFoundResource("Collection")
  },
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
    <PageContainer>
      <Breadcrumbs
        crumbs={[
          { label: "Collections", link: { to: "/collections" } },
          { label: collection?.title ?? "Collection" },
        ]}
      />
      {(query.isError || loadError) && (
        <p role="alert" className="text-sm text-destructive">
          {loadError ?? NETWORK_ERROR}
        </p>
      )}
      {collection && (
        <>
          <PageHeader
            title={collection.title}
            description={
              <>
                {collection.description && <p>{collection.description}</p>}
                {collection.maxCardCount > 0 && <p>Limit: {collection.maxCardCount} cards</p>}
              </>
            }
            actions={<AddCardsLink collectionId={collectionId} />}
          />
          <CollectionEntries
            collectionId={collectionId}
            search={search}
            onSearchChange={(next) => void navigate({ search: next })}
          />
        </>
      )}
    </PageContainer>
  )
}
