import { createFileRoute, useNavigate } from "@tanstack/react-router"
import { pageHead } from "@/lib/site"
import { PageContainer } from "@/components/layout/page-container"
import { PageHeader } from "@/components/layout/page-header"
import { MasterInventory } from "@/components/inventory/master-inventory"
import { catalogSearchSchema } from "@/lib/catalog-search"

export const Route = createFileRoute("/_authenticated/inventory")({
  head: () => pageHead("Master Inventory"),
  validateSearch: catalogSearchSchema,
  component: InventoryPage,
})

function InventoryPage() {
  const search = Route.useSearch()
  const navigate = useNavigate({ from: Route.fullPath })

  return (
    <PageContainer variant="wide">
      <PageHeader title="Master Inventory" />
      <MasterInventory search={search} onSearchChange={(next) => void navigate({ search: next })} />
    </PageContainer>
  )
}
