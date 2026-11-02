import { createFileRoute, useNavigate, useRouter } from "@tanstack/react-router"
import { Button } from "@/components/ui/button"
import { NOINDEX_META, pageHead } from "@/lib/site"
import { NotFoundView } from "@/components/layout/not-found"
import { PageContainer } from "@/components/layout/page-container"
import { PageHeader } from "@/components/layout/page-header"
import { CatalogFilterPanel } from "@/components/catalog/filter-panel"
import { InfiniteCardResults } from "@/components/catalog/infinite-card-results"
import { useListCatalogSeries } from "@/generated/endpoints/catalog/catalog"
import {
  getGetSharedWishlistQueryKey,
  getGetSharedWishlistQueryOptions,
  useListSharedWishlistFacets,
} from "@/generated/endpoints/shared/shared"
import { useInfiniteCardResultsProps } from "@/lib/infinite-catalog-cards"
import {
  hasActiveWishlistFilters,
  isWishlistStatus,
  useInfiniteSharedWishlist,
  wishlistSearchSchema,
  wishlistStatus,
  WISHLIST_STATUS_OPTIONS,
} from "@/lib/wishlist"
import type { WishlistFilters } from "@/lib/wishlist"

// Outside `_authenticated/`: a Guest reads it. Its items are the owner's,
// read-only, so the page offers no edit controls and no cache patches.
//
// A private, unshared or missing link is one identical not-found, returned from
// the loader rather than thrown: a thrown notFound skips the route's head, and
// the head carries `noindex` (ADR-0025).
export const Route = createFileRoute("/s/$profileSlug/wishlist")({
  validateSearch: wishlistSearchSchema,
  // Read fresh on every visit: a cached header would keep showing a Wishlist
  // whose sharing was stopped since.
  loader: async ({ context: { queryClient }, params }) => {
    const response = await queryClient.fetchQuery({
      ...getGetSharedWishlistQueryOptions(params.profileSlug),
      staleTime: 0,
    })
    if (response.status === 404) {
      queryClient.removeQueries({
        queryKey: getGetSharedWishlistQueryKey(params.profileSlug),
      })
      return "missing" as const
    }
    return response.status === 200 ? response.data.data : null
  },
  head: ({ loaderData }) => {
    const name =
      loaderData === "missing"
        ? "Wishlist not found"
        : loaderData
          ? `${loaderData.ownerName}'s Wishlist`
          : "Shared Wishlist"
    return { meta: [...pageHead(name).meta, NOINDEX_META] }
  },
  component: SharedWishlistPage,
})

function SharedWishlistPage() {
  const shared = Route.useLoaderData()
  const router = useRouter()

  if (shared === "missing") return <NotFoundView resource="Wishlist" />

  function retry() {
    void router.invalidate()
  }

  if (!shared)
    return (
      <PageContainer>
        <p role="alert" className="text-sm text-destructive">
          Could not load this Wishlist.
        </p>
        <div>
          <Button variant="outline" onClick={retry}>
            Retry
          </Button>
        </div>
      </PageContainer>
    )

  return <SharedWishlistList ownerName={shared.ownerName} />
}

function SharedWishlistList({ ownerName }: { ownerName: string }) {
  const { profileSlug } = Route.useParams()
  const search = Route.useSearch()
  const navigate = useNavigate({ from: Route.fullPath })

  // A filter change starts a fresh list, so return to its top.
  async function changeSearch(next: WishlistFilters) {
    await navigate({ search: next })
    window.scrollTo({ top: 0, behavior: "instant" })
  }

  const items = useInfiniteSharedWishlist(profileSlug, search)
  const results = useInfiniteCardResultsProps(items)
  const byCard = items.data?.items ?? {}
  const facetsQuery = useListSharedWishlistFacets(profileSlug, {
    ...search,
    status: wishlistStatus(search),
  })
  const seriesQuery = useListCatalogSeries()
  const facets =
    facetsQuery.data?.status === 200 ? facetsQuery.data.data.data : undefined
  const series =
    seriesQuery.data?.status === 200
      ? (seriesQuery.data.data.data?.series ?? [])
      : []

  return (
    <PageContainer>
      <PageHeader title="Wishlist" description={<p>Shared by {ownerName}</p>} />
      <section className="flex flex-col gap-6" aria-label="Wishlist contents">
        <CatalogFilterPanel
          search={search}
          facets={facets}
          series={series}
          onChange={(patch) => void changeSearch({ ...search, ...patch })}
          onClear={() => void changeSearch({ status: wishlistStatus(search) })}
          statusFilter={{
            value: wishlistStatus(search),
            options: WISHLIST_STATUS_OPTIONS,
            onChange: (status) => {
              if (!isWishlistStatus(status)) return
              void changeSearch({ ...search, status })
            },
          }}
        />
        <InfiniteCardResults
          {...results}
          // Read-only: the owner's wanted and owned, in the slot their
          // stepper takes.
          renderControl={(card) => {
            const item = byCard[card.id]
            if (!item) return null
            return (
              <div className="flex flex-col gap-1">
                <p className="text-center text-sm">
                  Owns {item.ownedQuantity} of {item.wantedQuantity}
                </p>
                <p className="text-center text-xs font-medium">
                  {item.fulfilled ? "Fulfilled" : "Still needed"}
                </p>
              </div>
            )
          }}
          emptyMessage={
            hasActiveWishlistFilters(search)
              ? "No Cards on this Wishlist match these filters."
              : "This Wishlist has no Cards yet."
          }
        />
      </section>
    </PageContainer>
  )
}
