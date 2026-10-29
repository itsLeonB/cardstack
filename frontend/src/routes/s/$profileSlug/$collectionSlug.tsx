import { createFileRoute, useNavigate } from "@tanstack/react-router"
import { NOINDEX_META, pageHead } from "@/lib/site"
import { notFoundResource } from "@/components/layout/not-found"
import { PageContainer } from "@/components/layout/page-container"
import { PageHeader } from "@/components/layout/page-header"
import { CatalogFilterPanel } from "@/components/catalog/filter-panel"
import { InfiniteCardResults } from "@/components/catalog/infinite-card-results"
import { useListCatalogSeries } from "@/generated/endpoints/catalog/catalog"
import {
  getGetSharedCollectionQueryOptions,
  useListSharedCollectionFacets,
} from "@/generated/endpoints/shared/shared"
import { catalogFilterSchema, hasActiveFilters } from "@/lib/catalog-search"
import type { CatalogFilters } from "@/lib/catalog-search"
import { useInfiniteCardResultsProps } from "@/lib/infinite-catalog-cards"
import { useInfiniteSharedCollectionEntries } from "@/lib/infinite-inventory"

// Outside `_authenticated/`: a Guest reads it. Its entries are the owner's,
// read-only, so the page offers no edit controls and no cache patches.
export const Route = createFileRoute("/s/$profileSlug/$collectionSlug")({
  validateSearch: catalogFilterSchema,
  // A private, unshared or missing link is one identical not-found.
  loader: async ({ context: { queryClient }, params }) => {
    const response = await queryClient.ensureQueryData(
      getGetSharedCollectionQueryOptions(
        params.profileSlug,
        params.collectionSlug
      )
    )
    if (response.status === 404) throw notFoundResource("Collection")
    return response.status === 200 ? response.data.data : undefined
  },
  // Crawlers are kept out (ADR-0025): a shared page is in no index or sitemap.
  head: ({ loaderData }) => ({
    meta: [
      ...pageHead(loaderData?.title ?? "Shared Collection").meta,
      NOINDEX_META,
    ],
  }),
  component: SharedCollectionPage,
})

function SharedCollectionPage() {
  const { profileSlug, collectionSlug } = Route.useParams()
  const search = Route.useSearch()
  const shared = Route.useLoaderData()
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

  if (!shared) {
    return (
      <PageContainer>
        <p role="alert" className="text-sm text-destructive">
          Could not load this Collection.
        </p>
      </PageContainer>
    )
  }

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
