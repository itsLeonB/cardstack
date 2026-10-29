import { createFileRoute, useNavigate, useRouter } from "@tanstack/react-router"
import { Button } from "@/components/ui/button"
import { NOINDEX_META, pageHead } from "@/lib/site"
import { NotFoundView } from "@/components/layout/not-found"
import { PageContainer } from "@/components/layout/page-container"
import { PageHeader } from "@/components/layout/page-header"
import { CatalogFilterPanel } from "@/components/catalog/filter-panel"
import { InfiniteCardResults } from "@/components/catalog/infinite-card-results"
import { SharedBinderView } from "@/components/collections/shared-binder-view"
import { useListCatalogSeries } from "@/generated/endpoints/catalog/catalog"
import {
  getGetSharedCollectionQueryKey,
  getGetSharedCollectionQueryOptions,
  useListSharedCollectionFacets,
} from "@/generated/endpoints/shared/shared"
import type { SharedCollectionSummary } from "@/generated/models"
import { catalogFilterSchema, hasActiveFilters } from "@/lib/catalog-search"
import type { CatalogFilters } from "@/lib/catalog-search"
import { useInfiniteCardResultsProps } from "@/lib/infinite-catalog-cards"
import { useInfiniteSharedCollectionEntries } from "@/lib/infinite-inventory"

// Outside `_authenticated/`: a Guest reads it. Its entries are the owner's,
// read-only, so the page offers no edit controls and no cache patches.
//
// A private, unshared or missing link is one identical not-found, returned from
// the loader rather than thrown: a thrown notFound skips the route's head, and
// the head carries `noindex` (ADR-0025).
export const Route = createFileRoute("/s/$profileSlug/$collectionSlug")({
  validateSearch: catalogFilterSchema,
  // Read fresh on every visit: a cached header would keep showing a Collection
  // whose sharing was stopped since.
  loader: async ({ context: { queryClient }, params }) => {
    const response = await queryClient.fetchQuery({
      ...getGetSharedCollectionQueryOptions(
        params.profileSlug,
        params.collectionSlug
      ),
      staleTime: 0,
    })
    if (response.status === 404) {
      queryClient.removeQueries({
        queryKey: getGetSharedCollectionQueryKey(
          params.profileSlug,
          params.collectionSlug
        ),
      })
      return "missing" as const
    }
    return response.status === 200 ? response.data.data : null
  },
  head: ({ loaderData }) => {
    const name =
      loaderData === "missing"
        ? "Collection not found"
        : (loaderData?.title ?? "Shared Collection")
    return { meta: [...pageHead(name).meta, NOINDEX_META] }
  },
  component: SharedCollectionPage,
})

function SharedCollectionPage() {
  const shared = Route.useLoaderData()
  const router = useRouter()

  if (shared === "missing") return <NotFoundView resource="Collection" />

  function retry() {
    void router.invalidate()
  }

  if (!shared)
    return (
      <PageContainer>
        <p role="alert" className="text-sm text-destructive">
          Could not load this Collection.
        </p>
        <div>
          <Button variant="outline" onClick={retry}>
            Retry
          </Button>
        </div>
      </PageContainer>
    )

  return <SharedCollectionList shared={shared} />
}

function SharedCollectionList({ shared }: { shared: SharedCollectionSummary }) {
  const { profileSlug, collectionSlug } = Route.useParams()
  const search = Route.useSearch()
  const navigate = useNavigate({ from: Route.fullPath })

  // A filter change starts a fresh list, so return to its top.
  async function changeSearch(next: CatalogFilters) {
    await navigate({ search: next })
    window.scrollTo({ top: 0, behavior: "instant" })
  }

  const entries = useInfiniteSharedCollectionEntries(
    profileSlug,
    collectionSlug,
    search
  )
  const results = useInfiniteCardResultsProps(entries)
  const quantities = entries.data?.quantities ?? {}
  const facetsQuery = useListSharedCollectionFacets(
    profileSlug,
    collectionSlug,
    search
  )
  const seriesQuery = useListCatalogSeries()
  const facets =
    facetsQuery.data?.status === 200 ? facetsQuery.data.data.data : undefined
  const series =
    seriesQuery.data?.status === 200
      ? (seriesQuery.data.data.data?.series ?? [])
      : []

  return (
    <PageContainer>
      <PageHeader
        title={shared.title}
        description={
          <>
            <p>Shared by {shared.ownerName}</p>
            {shared.description && <p>{shared.description}</p>}
          </>
        }
      />
      {shared.binder && (
        <SharedBinderView
          key={`${profileSlug}/${collectionSlug}`}
          profileSlug={profileSlug}
          collectionSlug={collectionSlug}
          binder={shared.binder}
        />
      )}
      <section className="flex flex-col gap-6" aria-label="Collection contents">
        <CatalogFilterPanel
          search={search}
          facets={facets}
          series={series}
          onChange={(patch) => void changeSearch({ ...search, ...patch })}
          onClear={() => void changeSearch({})}
        />
        <InfiniteCardResults
          {...results}
          // Read-only: the owner's quantity, in the slot their stepper takes.
          renderControl={(card) => (
            <div className="flex flex-col gap-1">
              <p className="text-center text-sm">
                Quantity: {quantities[card.id] ?? 0}
              </p>
            </div>
          )}
          emptyMessage={
            hasActiveFilters(search)
              ? "No Cards in this Collection match these filters."
              : "This Collection has no Cards yet."
          }
        />
      </section>
    </PageContainer>
  )
}
