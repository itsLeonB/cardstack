import { createFileRoute, useNavigate } from "@tanstack/react-router"
import { pageHead } from "@/lib/site"
import { PageContainer } from "@/components/layout/page-container"
import { PageHeader } from "@/components/layout/page-header"
import { Wishlist } from "@/components/wishlist/wishlist"
import { wishlistSearchSchema } from "@/lib/wishlist"
import type { WishlistFilters } from "@/lib/wishlist"

export const Route = createFileRoute("/_authenticated/wishlist")({
  head: () => pageHead("Wishlist"),
  // Filters only: the list is infinite, so a stale `?page=` is stripped.
  validateSearch: wishlistSearchSchema,
  component: WishlistPage,
})

function WishlistPage() {
  const search = Route.useSearch()
  const navigate = useNavigate({ from: Route.fullPath })

  // A filter change starts a fresh list, so return to its top.
  async function changeSearch(next: WishlistFilters) {
    await navigate({ search: next })
    window.scrollTo({ top: 0, behavior: "instant" })
  }

  return (
    <PageContainer variant="wide">
      <PageHeader title="Wishlist" />
      <Wishlist
        search={search}
        onSearchChange={(next) => void changeSearch(next)}
      />
    </PageContainer>
  )
}
