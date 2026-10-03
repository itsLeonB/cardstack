import { createFileRoute, useNavigate } from "@tanstack/react-router"
import { pageHead } from "@/lib/site"
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
import { catalogFilterSchema } from "@/lib/catalog-search"
import type { CatalogFilters } from "@/lib/catalog-search"
import { errorDetail, NETWORK_ERROR } from "@/lib/collections"

export const Route = createFileRoute(
  "/_authenticated/collections/$collectionId/"
)({
  // Filters only: the entries list is infinite, so a stale `?page=` is stripped.
  validateSearch: catalogFilterSchema,
  // Entries are deliberately not prefetched into the cache: the page refetches
  // them on mount so a card removed earlier doesn't reappear from stale data.
  loader: async ({ context: { queryClient }, params }) => {
    const response = await queryClient.ensureQueryData(
      getGetCollectionQueryOptions(params.collectionId)
    )
    if (response.status === 404) throw notFoundResource("Collection")
    return response.status === 200 ? response.data.data.title : undefined
  },
  head: ({ loaderData }) => pageHead(loaderData ?? "Collection"),
  component: CollectionPage,
})

function CollectionPage() {
  const { collectionId } = Route.useParams()
  const search = Route.useSearch()
  const navigate = useNavigate({ from: Route.fullPath })

  // A filter change starts a fresh list, so return to its top.
  async function changeSearch(next: CatalogFilters) {
    await navigate({ search: next })
    window.scrollTo({ top: 0, behavior: "instant" })
  }
  const query = useGetCollection(collectionId)
  const collection =
    query.data?.status === 200 ? query.data.data.data : undefined
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
                {collection.maxCardCount > 0 && (
                  <p>Limit: {collection.maxCardCount} cards</p>
                )}
              </>
            }
            actions={<AddCardsLink collectionId={collectionId} />}
          />
          <CollectionEntries
            collectionId={collectionId}
            search={search}
            onSearchChange={(next) => void changeSearch(next)}
          />
        </>
      )}
    </PageContainer>
  )
}
