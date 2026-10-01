import { createFileRoute, Link, useNavigate } from "@tanstack/react-router"
import { MasterInventory } from "@/components/inventory/master-inventory"
import { catalogSearchSchema } from "@/lib/catalog-search"

export const Route = createFileRoute("/_authenticated/inventory")({
  validateSearch: catalogSearchSchema,
  component: InventoryPage,
})

function InventoryPage() {
  const search = Route.useSearch()
  const navigate = useNavigate({ from: Route.fullPath })

  return (
    <main className="mx-auto flex max-w-5xl flex-col gap-6 p-6">
      <Link
        to="/account"
        className="w-fit text-sm text-muted-foreground underline-offset-2 hover:underline"
      >
        ← Back to Account
      </Link>
      <h1 className="font-heading text-2xl font-medium">Master Inventory</h1>
      <MasterInventory search={search} onSearchChange={(next) => void navigate({ search: next })} />
    </main>
  )
}
