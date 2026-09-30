import { useState } from "react"
import { useQueryClient } from "@tanstack/react-query"
import { useSearchCatalogCards } from "@/generated/endpoints/catalog/catalog"
import {
  getListCollectionEntriesQueryKey,
  useAddCollectionEntry,
} from "@/generated/endpoints/inventory/inventory"
import type { CardSummary } from "@/generated/models"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { CardLine, QuantityInput } from "@/components/collections/entry-parts"
import { NETWORK_ERROR, parseQuantity } from "@/lib/collections"

function ResultRow({
  card,
  onAdd,
  disabled,
}: {
  card: CardSummary
  onAdd: (cardId: string, quantity: number) => void
  disabled: boolean
}) {
  const [draft, setDraft] = useState("1")
  const quantity = parseQuantity(draft)

  return (
    <li className="flex flex-wrap items-center gap-3 rounded-2xl border p-3">
      <CardLine card={card} />
      <QuantityInput
        label={`Quantity to add of ${card.name}`}
        value={draft}
        onChange={setDraft}
      />
      <Button
        type="button"
        size="sm"
        disabled={disabled || quantity === undefined}
        aria-label={`Add ${card.name}`}
        onClick={() => quantity !== undefined && onAdd(card.id, quantity)}
      >
        Add
      </Button>
    </li>
  )
}

/** Name search over the catalog; each result can be added with a quantity. */
export function AddEntry({
  collectionId,
  onError,
}: {
  collectionId: string
  onError: (message: string | null) => void
}) {
  const queryClient = useQueryClient()
  const [input, setInput] = useState("")
  const [name, setName] = useState("")
  const addMutation = useAddCollectionEntry()
  // ponytail: first page only (name search); refine the query if a name matches > 12 cards.
  const search = useSearchCatalogCards(
    { name, limit: 12 },
    { query: { enabled: name !== "" } }
  )
  const cards =
    name !== "" && search.data?.status === 200 ? (search.data.data.data ?? []) : []

  function handleAdd(cardId: string, quantity: number) {
    onError(null)
    addMutation.mutate(
      { id: collectionId, data: { cardId, quantity } },
      {
        onSuccess: (response) => {
          if (response.status === 201) {
            void queryClient.invalidateQueries({
              queryKey: getListCollectionEntriesQueryKey(collectionId),
            })
            // Hide results so the same Card can't be re-added (409).
            setName("")
            setInput("")
            return
          }
          onError(response.data.detail ?? "Could not add this Card.")
        },
        onError: () => onError(NETWORK_ERROR),
      }
    )
  }

  return (
    <div className="flex flex-col gap-3">
      <form
        className="flex gap-2"
        onSubmit={(event) => {
          event.preventDefault()
          setName(input.trim())
        }}
      >
        <Input
          type="search"
          placeholder="Search Cards by name to add"
          aria-label="Search Cards to add"
          value={input}
          onChange={(event) => setInput(event.target.value)}
        />
        <Button type="submit">Search</Button>
      </form>
      {name !== "" && search.isError && (
        <p role="alert" className="text-sm text-destructive">
          Could not search the catalog.
        </p>
      )}
      {name !== "" && search.data?.status === 200 && cards.length === 0 && (
        <p className="text-sm text-muted-foreground">No Cards match that name.</p>
      )}
      <ul className="flex flex-col gap-2">
        {cards.map((card) => (
          <ResultRow
            key={card.id}
            card={card}
            disabled={addMutation.isPending}
            onAdd={handleAdd}
          />
        ))}
      </ul>
    </div>
  )
}
