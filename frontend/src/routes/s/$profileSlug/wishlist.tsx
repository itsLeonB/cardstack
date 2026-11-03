import { createFileRoute, useNavigate, useRouter } from "@tanstack/react-router"
import { Button } from "@/components/ui/button"
import { NOINDEX_META, pageHead } from "@/lib/site"
import { NotFoundView } from "@/components/layout/not-found"
import { PageContainer } from "@/components/layout/page-container"
import { PageHeader } from "@/components/layout/page-header"
import { CatalogFilterPanel } from "@/components/catalog/filter-panel"
import { InfiniteCardResults } from "@/components/catalog/infinite-card-results"
import { useListCatalogSeries } from "@/generated/endpoints/catalog/catalog"
import { useGetProfile } from "@/generated/endpoints/profile/profile"
import {
  getGetSharedWishlistQueryKey,
  getGetSharedWishlistQueryOptions,
  useListSharedWishlistFacets,
} from "@/generated/endpoints/shared/shared"
import type { ListSharedWishlistEntriesParams } from "@/generated/models/listSharedWishlistEntriesParams"
import { useInfiniteCardResultsProps } from "@/lib/infinite-catalog-cards"
import {
  hasActiveWishlistFilters,
  isWishlistStatus,
  useInfiniteSharedWishlist,
  wishlistSearchSchema,
  wishlistStatus,
  WISHLIST_STATUS_OPTIONS,
} from "@/lib/wishlist"
import { useIsGuest, useSession } from "@/lib/session"
import type { ListSharedWishlistEntriesViewerFilter } from "@/generated/models/listSharedWishlistEntriesViewerFilter"
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
  const { isAuthenticated } = useSession()
  const isGuestViewer = useIsGuest()
  const profile = useGetProfile()

  // The signed-in viewer's own profile Slug, only when the profile request
  // has succeeded. While pending or failed, the viewer overlay stays hidden.
  const profileLoaded = profile.data?.status === 200
  const viewerSlug =
    profile.data?.status === 200 ? profile.data.data.data.slug : undefined
  const isOwner = profileLoaded && isAuthenticated && viewerSlug === profileSlug
  // Only a signed-in non-owner viewer gets the viewer overlay.
  const showViewerOverlay = profileLoaded && isAuthenticated && !isOwner

  // A filter change starts a fresh list, so return to its top.
  async function changeSearch(next: WishlistFilters) {
    await navigate({ search: next })
    window.scrollTo({ top: 0, behavior: "instant" })
  }

  // Only a signed-in non-owner viewer sends the viewer filter to the entries
  // endpoint. The facets endpoint and the empty-state message never see it.
  const params: ListSharedWishlistEntriesParams = { ...search }
  // The viewer filter is only sent when the viewer overlay is active.
  if (!showViewerOverlay) {
    delete params.viewerFilter
  }
  const items = useInfiniteSharedWishlist(profileSlug, params)
  const results = useInfiniteCardResultsProps(items)
  const byCard = items.data?.items ?? {}
  const { viewerFilter: _, ...facetsSearch } = search
  const facetsQuery = useListSharedWishlistFacets(profileSlug, {
    ...facetsSearch,
    status: wishlistStatus(search),
  })
  const seriesQuery = useListCatalogSeries()
  const facets =
    facetsQuery.data?.status === 200 ? facetsQuery.data.data.data : undefined
  const series =
    seriesQuery.data?.status === 200
      ? (seriesQuery.data.data.data?.series ?? [])
      : []

  const viewerFilterValue = search.viewerFilter ?? ""
  const viewerFilterOptions = [
    { value: "", label: "All" },
    { value: "have", label: "I have it" },
    { value: "dont-have", label: "I do not have it" },
  ]

  // Filters that count as active for the empty-state message: the viewer
  // filter only counts when the viewer overlay is active.
  const activeSearch = showViewerOverlay
    ? search
    : { ...search, viewerFilter: undefined }

  return (
    <PageContainer>
      <PageHeader title="Wishlist" description={<p>Shared by {ownerName}</p>} />
      <section className="flex flex-col gap-6" aria-label="Wishlist contents">
        <CatalogFilterPanel
          search={search}
          facets={facets}
          series={series}
          onChange={(patch) => void changeSearch({ ...search, ...patch })}
          onClear={() =>
            void changeSearch({
              status: wishlistStatus(search),
              viewerFilter: undefined,
            })
          }
          statusFilter={{
            value: wishlistStatus(search),
            options: WISHLIST_STATUS_OPTIONS,
            onChange: (status) => {
              if (!isWishlistStatus(status)) return
              void changeSearch({ ...search, status })
            },
          }}
          viewerFilter={
            showViewerOverlay
              ? {
                  value: viewerFilterValue,
                  options: viewerFilterOptions,
                  onChange: (value) => {
                    // SAFETY: the "" All case is handled above; the remaining
                    // button values are exactly "have" / "dont-have".
                    void changeSearch({
                      ...search,
                      viewerFilter:
                        value === ""
                          ? undefined
                          : (value as ListSharedWishlistEntriesViewerFilter),
                    })
                  },
                }
              : undefined
          }
        />
        {isGuestViewer && (
          <p role="status" className="text-sm text-muted-foreground">
            Sign in to see what you own of these Cards.
          </p>
        )}
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
                {showViewerOverlay &&
                  item.viewerOwnedQuantity !== undefined && (
                    <p className="text-center text-xs font-medium">
                      You own {item.viewerOwnedQuantity}
                    </p>
                  )}
                <p className="text-center text-xs font-medium">
                  {item.fulfilled ? "Fulfilled" : "Still needed"}
                </p>
              </div>
            )
          }}
          emptyMessage={
            hasActiveWishlistFilters(activeSearch)
              ? "No Cards on this Wishlist match these filters."
              : "This Wishlist has no Cards yet."
          }
        />
      </section>
    </PageContainer>
  )
}
