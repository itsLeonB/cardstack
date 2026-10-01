import { keepPreviousData } from "@tanstack/react-query"
import { useListMasterInventory } from "@/generated/endpoints/inventory/inventory"
import { CardResults } from "@/components/catalog/card-results"
import { errorDetail } from "@/lib/collections"

/** The user's Cards with the total quantity owned across all their Collections (read-only). */
export function MasterInventory({ page, onPageChange }: { page: number; onPageChange: (page: number) => void }) {
  // gcTime 0 + refetchOnMount: no cached page can show totals from before a Collection change.
  const query = useListMasterInventory(
    { page },
    { query: { gcTime: 0, refetchOnMount: "always", placeholderData: keepPreviousData } }
  )
  const result = query.data?.status === 200 ? query.data.data : undefined
  const items = result?.data ?? []
  const quantities = new Map(items.map((item) => [item.card.id, item.quantity]))
  const loadError = errorDetail(query.data, "Could not load your inventory.")

  return (
    <CardResults
      cards={items.map((item) => item.card)}
      total={result?.meta.total ?? 0}
      page={page}
      limit={result?.meta.limit ?? 24}
      isPending={query.isPending}
      isError={query.isError || Boolean(loadError)}
      errorMessage={loadError}
      emptyMessage="You don't own any Cards yet. Add some to a Collection from the catalog."
      onPageChange={onPageChange}
      renderControl={(card) => (
        <p className="text-sm font-medium text-muted-foreground">×{quantities.get(card.id) ?? 0}</p>
      )}
    />
  )
}
