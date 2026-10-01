import { createFileRoute, Link, useNavigate } from "@tanstack/react-router"
import { z } from "zod"
import { MasterInventory } from "@/components/inventory/master-inventory"

export const Route = createFileRoute("/_authenticated/inventory")({
  validateSearch: z.object({ page: z.number().int().min(1).catch(1).default(1) }),
  component: InventoryPage,
})

function InventoryPage() {
  const { page } = Route.useSearch()
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
      <MasterInventory page={page} onPageChange={(next) => void navigate({ search: { page: next } })} />
    </main>
  )
}
