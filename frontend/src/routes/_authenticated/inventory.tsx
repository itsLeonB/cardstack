import { createFileRoute, useNavigate } from "@tanstack/react-router"
import { pageHead } from "@/lib/site"
import { PageContainer } from "@/components/layout/page-container"
import { PageHeader } from "@/components/layout/page-header"
import { MasterInventory } from "@/components/inventory/master-inventory"
import { catalogFilterSchema } from "@/lib/catalog-search"
import type { CatalogFilters } from "@/lib/catalog-search"

export const Route = createFileRoute("/_authenticated/inventory")({
  head: () => pageHead("Master Inventory"),
  // Filters only: the list is infinite, so a stale `?page=` is stripped.
  validateSearch: catalogFilterSchema,
  component: InventoryPage,
})

function InventoryPage() {
  const search = Route.useSearch()
  const navigate = useNavigate({ from: Route.fullPath })

  // A filter change starts a fresh list, so return to its top.
  async function changeSearch(next: CatalogFilters) {
    await navigate({ search: next })
    window.scrollTo({ top: 0, behavior: "instant" })
  }

  return (
    <PageContainer variant="wide">
      <PageHeader title="Master Inventory" />
      <MasterInventory
        search={search}
        onSearchChange={(next) => void changeSearch(next)}
      />
    </PageContainer>
  )
}
