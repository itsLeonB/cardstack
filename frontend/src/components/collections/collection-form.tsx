import { useState } from "react"
import { Button } from "@/components/ui/button"
import {
  Field,
  FieldDescription,
  FieldError,
  FieldGroup,
  FieldLabel,
} from "@/components/ui/field"
import { Input } from "@/components/ui/input"
import type { CollectionBody } from "@/generated/models"

// Backend caps maxCardCount at the int32 maximum.
const MAX_CARD_COUNT = 2147483647

export interface CollectionFormValues {
  title: string
  description: string
  maxCardCount: string
}

export const emptyCollectionFormValues: CollectionFormValues = {
  title: "",
  description: "",
  maxCardCount: "",
}

interface CollectionFormProps {
  initialValues?: CollectionFormValues
  submitLabel: string
  pendingLabel: string
  isPending: boolean
  errorMessage?: string | null
  onSubmit: (body: CollectionBody) => void
}

export function CollectionForm({
  initialValues,
  submitLabel,
  pendingLabel,
  isPending,
  errorMessage,
  onSubmit,
}: CollectionFormProps) {
  const [values, setValues] = useState<CollectionFormValues>(
    initialValues ?? emptyCollectionFormValues
  )
  const [validationError, setValidationError] = useState<string | null>(null)

  function handleSubmit(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault()
    setValidationError(null)

    const title = values.title.trim()
    if (!title) {
      setValidationError("Title is required.")
      return
    }

    // 0 means no limit; sending it on update clears an existing limit.
    let maxCardCount = 0
    const rawMaxCardCount = values.maxCardCount.trim()
    if (rawMaxCardCount) {
      const parsed = Number(rawMaxCardCount)
      if (!Number.isInteger(parsed) || parsed < 0) {
        setValidationError(
          "Max card count must be a non-negative whole number."
        )
        return
      }
      if (parsed > MAX_CARD_COUNT) {
        setValidationError(
          `Max card count can't exceed ${MAX_CARD_COUNT.toLocaleString("en-US")}.`
        )
        return
      }
      maxCardCount = parsed
    }

    onSubmit({
      title,
      description: values.description.trim() || undefined,
      maxCardCount,
    })
  }

  return (
    <form onSubmit={handleSubmit} noValidate>
      <FieldGroup>
        <Field>
          <FieldLabel htmlFor="title">Title</FieldLabel>
          <Input
            id="title"
            required
            value={values.title}
            onChange={(event) =>
              setValues((prev) => ({ ...prev, title: event.target.value }))
            }
          />
        </Field>
        <Field>
          <FieldLabel htmlFor="description">Description</FieldLabel>
          <Input
            id="description"
            value={values.description}
            onChange={(event) =>
              setValues((prev) => ({
                ...prev,
                description: event.target.value,
              }))
            }
          />
        </Field>
        <Field>
          <FieldLabel htmlFor="maxCardCount">Max card count</FieldLabel>
          <Input
            id="maxCardCount"
            type="number"
            min={0}
            step={1}
            inputMode="numeric"
            value={values.maxCardCount}
            onChange={(event) =>
              setValues((prev) => ({
                ...prev,
                maxCardCount: event.target.value,
              }))
            }
          />
          <FieldDescription>
            Optional hard cap on this Collection&apos;s summed card quantity.
          </FieldDescription>
        </Field>
        {(validationError ?? errorMessage) && (
          <FieldError>{validationError ?? errorMessage}</FieldError>
        )}
        <Field>
          <Button type="submit" disabled={isPending}>
            {isPending ? pendingLabel : submitLabel}
          </Button>
        </Field>
      </FieldGroup>
    </form>
  )
}
