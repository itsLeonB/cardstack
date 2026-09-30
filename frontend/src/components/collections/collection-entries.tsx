import { useState } from "react"
import { useQueryClient } from "@tanstack/react-query"
import {
  getListCollectionEntriesQueryKey,
  useListCollectionEntries,
  useRemoveCollectionEntry,
  useUpdateCollectionEntry,
} from "@/generated/endpoints/inventory/inventory"
import type { InventoryItem } from "@/generated/models"
import { AddEntry } from "@/components/collections/add-entry"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Skeleton } from "@/components/ui/skeleton"
import { errorDetail, NETWORK_ERROR, parseQuantity } from "@/lib/collections"

function EntryRow({
  collectionId,
  item,
  onError,
}: {
  collectionId: string
  item: InventoryItem
  onError: (message: string | null) => void
}) {
  const queryClient = useQueryClient()
  const [draft, setDraft] = useState(item.quantity.toString())
  const updateMutation = useUpdateCollectionEntry()
  const removeMutation = useRemoveCollectionEntry()
  const quantity = parseQuantity(draft)
  const busy = updateMutation.isPending || removeMutation.isPending

  function refresh() {
    void queryClient.invalidateQueries({
      queryKey: getListCollectionEntriesQueryKey(collectionId),
    })
  }

  function handleSave() {
    if (quantity === undefined) return
    onError(null)
    updateMutation.mutate(
      { id: collectionId, cardId: item.card.id, data: { quantity } },
      {
        onSuccess: (response) => {
          if (response.status === 200) return refresh()
          // Rejected (e.g. capacity): show the stored quantity again.
          setDraft(item.quantity.toString())
          onError(response.data.detail ?? "Could not update this quantity.")
        },
        onError: () => {
          setDraft(item.quantity.toString())
          onError(NETWORK_ERROR)
        },
      }
    )
  }

  function handleRemove() {
    onError(null)
    removeMutation.mutate(
      { id: collectionId, cardId: item.card.id },
      {
        onSuccess: (response) => {
          if (response.status === 204) return refresh()
          onError(response.data.detail ?? "Could not remove this Card.")
        },
        onError: () => onError(NETWORK_ERROR),
      }
    )
  }

  return (
    <li className="flex flex-wrap items-center gap-3 rounded-2xl border p-3">
      <div className="min-w-0 flex-1">
        <p className="truncate text-sm font-medium">{item.card.name}</p>
        <p className="text-xs text-muted-foreground">
          {item.card.expansionSet.code} · No. {item.card.localId} ·{" "}
          {item.card.rarity.name}
        </p>
      </div>
      <Input
        type="number"
        min={1}
        step={1}
        inputMode="numeric"
        className="w-20"
        aria-label={`Quantity of ${item.card.name}`}
        value={draft}
        onChange={(event) => setDraft(event.target.value)}
      />
      <Button
        type="button"
        variant="outline"
        size="sm"
        disabled={busy || quantity === undefined || quantity === item.quantity}
        aria-label={`Save quantity of ${item.card.name}`}
        onClick={handleSave}
      >
        Save
      </Button>
      <Button
        type="button"
        variant="destructive"
        size="sm"
        disabled={busy}
        aria-label={`Remove ${item.card.name}`}
        onClick={handleRemove}
      >
        Remove
      </Button>
    </li>
  )
}

export function CollectionEntries({ collectionId }: { collectionId: string }) {
  const [errorMessage, setErrorMessage] = useState<string | null>(null)
  const query = useListCollectionEntries(collectionId)
  const items = query.data?.status === 200 ? (query.data.data.data ?? []) : []
  const loadError = errorDetail(query.data, "Could not load this Collection's Cards.")

  return (
    <section className="flex flex-col gap-4" aria-label="Collection contents">
      <AddEntry collectionId={collectionId} onError={setErrorMessage} />

      {errorMessage && (
        <p role="alert" className="text-sm text-destructive">
          {errorMessage}
        </p>
      )}

      {query.isPending && (
        <div className="flex flex-col gap-3" aria-busy="true" aria-label="Loading Cards">
          <Skeleton className="h-16 w-full" />
          <Skeleton className="h-16 w-full" />
        </div>
      )}

      {(query.isError || loadError) && (
        <p role="alert" className="text-sm text-destructive">
          {loadError ?? "Could not reach the backend. Please try again."}
        </p>
      )}

      {query.data?.status === 200 && items.length === 0 && (
        <p className="text-sm text-muted-foreground">
          This Collection has no Cards yet. Search above to add one.
        </p>
      )}

      <ul className="flex flex-col gap-3">
        {items.map((item) => (
          <EntryRow
            key={`${item.card.id}:${item.quantity}`}
            collectionId={collectionId}
            item={item}
            onError={setErrorMessage}
          />
        ))}
      </ul>
    </section>
  )
}
