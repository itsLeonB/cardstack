import { useState } from "react"
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog"
import {
  SELECT_CLASS,
  flattenSets,
  useCardLookup,
} from "@/components/admin/form-shared"
import { Button } from "@/components/ui/button"
import {
  Field,
  FieldDescription,
  FieldError,
  FieldGroup,
  FieldLabel,
} from "@/components/ui/field"
import { Input } from "@/components/ui/input"
import { useDeleteAdminCard } from "@/generated/endpoints/admin/admin"
import { useListCatalogSeries } from "@/generated/endpoints/catalog/catalog"

/**
 * Deletes a Card entered by mistake (docs/adr/0019). The Card is looked up by
 * the set and collector number it was entered under, so one whose upload
 * failed and that no catalog list shows can still be found. Asking to delete
 * opens a confirmation that names the Card, and only confirming sends the
 * delete, by the Card's id. The API removes the row, the hosted image and the
 * embeddings, and refuses while a Collection still holds the Card, which this
 * shows as its reason.
 */
export function DeleteCardForm() {
  const seriesQuery = useListCatalogSeries()
  const remove = useDeleteAdminCard()

  const [setId, setSetId] = useState("")
  const [localId, setLocalId] = useState("")
  const [confirming, setConfirming] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [deleted, setDeleted] = useState<string | null>(null)

  const browse =
    seriesQuery.data?.status === 200 ? seriesQuery.data.data.data : undefined
  const sets = flattenSets(browse)
  const setCode = sets.find((s) => s.id === setId)?.code ?? ""
  const number = localId.trim()
  const lookup = useCardLookup(setId, number)
  const card = lookup.card

  function handleAsk(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault()
    setError(null)
    setDeleted(null)
    if (!setId || !number) {
      setError("Choose an expansion set and enter a collector number.")
      return
    }
    if (lookup.state === "checking" || lookup.state === "failed") {
      setError(
        lookup.state === "checking"
          ? "Still looking for the card. Try again in a moment."
          : "Could not look the card up. Try again."
      )
      return
    }
    if (!card) {
      setError(`There is no card ${setCode} ${number}.`)
      return
    }
    setConfirming(true)
  }

  function handleConfirm() {
    if (!card) return
    remove.mutate(
      { id: card.id },
      {
        onSuccess: (response) => {
          setConfirming(false)
          if (response.status === 204) {
            setDeleted(`${setCode} ${number}`)
            setLocalId("")
            return
          }
          setError(response.data.detail ?? "Could not delete the card.")
        },
        onError: () => {
          setConfirming(false)
          setError("Could not reach the server. Please try again.")
        },
      }
    )
  }

  return (
    <form onSubmit={handleAsk} noValidate aria-label="Delete card">
      <FieldGroup>
        <Field>
          <FieldLabel htmlFor="delete-card-set">Expansion set</FieldLabel>
          <select
            id="delete-card-set"
            className={SELECT_CLASS}
            value={setId}
            onChange={(e) => setSetId(e.target.value)}
          >
            <option value="">Choose a set</option>
            {sets.map((s) => (
              <option key={s.id} value={s.id}>
                {s.code} — {s.name}
              </option>
            ))}
          </select>
        </Field>
        <Field>
          <FieldLabel htmlFor="delete-card-number">Collector number</FieldLabel>
          <Input
            id="delete-card-number"
            value={localId}
            onChange={(e) => setLocalId(e.target.value)}
          />
          <FieldDescription>
            {card ? `Found: ${card.name}. ` : ""}
            {lookup.state === "missing" ? "There is no card there. " : ""}
            Removes the card, its image and its embeddings for good.
          </FieldDescription>
        </Field>
        {error && <FieldError>{error}</FieldError>}
        {deleted && <p role="status">Deleted {deleted}.</p>}
        <Field>
          <Button type="submit" variant="destructive">
            Delete card
          </Button>
        </Field>
      </FieldGroup>
      <AlertDialog open={confirming} onOpenChange={setConfirming}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>
              Delete {setCode} {number} {card?.name}?
            </AlertDialogTitle>
            <AlertDialogDescription>
              This permanently deletes the card, its image and its embeddings.
              This cannot be undone.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Cancel</AlertDialogCancel>
            <AlertDialogAction
              variant="destructive"
              disabled={remove.isPending}
              onClick={handleConfirm}
            >
              {remove.isPending ? "Deleting..." : "Delete"}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </form>
  )
}
