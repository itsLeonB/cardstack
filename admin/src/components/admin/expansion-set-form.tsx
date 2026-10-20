import { useState } from "react"
import { useQueryClient } from "@tanstack/react-query"
import { SELECT_CLASS } from "@/components/admin/form-shared"
import { Button } from "@/components/ui/button"
import {
  Field,
  FieldDescription,
  FieldError,
  FieldGroup,
  FieldLabel,
} from "@/components/ui/field"
import { Input } from "@/components/ui/input"
import {
  useCreateExpansionSet,
  useUpdateExpansionSet,
} from "@/generated/endpoints/admin/admin"
import {
  getListCatalogSeriesQueryKey,
  useListCatalogSeries,
} from "@/generated/endpoints/catalog/catalog"
import type { ExpansionSetSummary } from "@/generated/models"

// Mirrors the API's `code` pattern, so a bad code is caught before sending.
const CODE_PATTERN = /^[A-Za-z0-9][A-Za-z0-9_-]*$/

interface SetOption {
  id: string
  code: string
  name: string
  releaseDate: string
  seriesId: string
}

/**
 * Creates or edits one Expansion Set (ADR-0019); the API marks it Curated.
 * "New set" lets a code be typed and creates the set (POST), and a code that
 * already exists is refused. Picking an existing set, including one the Asia
 * site ingested, loads its name, release date and Series into the form and
 * locks its code; saving replaces them with the form (PUT by id).
 */
export function ExpansionSetForm() {
  const queryClient = useQueryClient()
  const seriesQuery = useListCatalogSeries()
  const create = useCreateExpansionSet()
  const update = useUpdateExpansionSet()

  const [existing, setExisting] = useState("")
  const [code, setCode] = useState("")
  const [name, setName] = useState("")
  const [releaseDate, setReleaseDate] = useState("")
  const [seriesId, setSeriesId] = useState("")
  const [error, setError] = useState<string | null>(null)
  const [saved, setSaved] = useState<string | null>(null)

  const browse =
    seriesQuery.data?.status === 200 ? seriesQuery.data.data.data : undefined
  const series = browse?.series ?? []
  const toOption = (
    set: ExpansionSetSummary,
    parentSeriesId: string
  ): SetOption => ({
    id: set.id,
    code: set.code,
    name: set.name,
    // The API sends a timestamp; the date input and the API want YYYY-MM-DD.
    releaseDate: set.releaseDate?.slice(0, 10) ?? "",
    seriesId: parentSeriesId,
  })
  // A set with no Series loads with an empty Series, so validate() makes the
  // Admin pick one before the PUT, which requires it.
  const sets: SetOption[] = [
    ...series.flatMap((s) =>
      (s.expansionSets ?? []).map((set) => toOption(set, s.id))
    ),
    ...(browse?.ungroupedExpansionSets ?? []).map((set) => toOption(set, "")),
  ]
  const pending = create.isPending || update.isPending

  function chooseExisting(value: string) {
    setExisting(value)
    setSaved(null)
    const set = sets.find((s) => s.id === value)
    setCode(set?.code ?? "")
    setName(set?.name ?? "")
    setReleaseDate(set?.releaseDate ?? "")
    setSeriesId(set?.seriesId ?? "")
  }

  function validate(): string | null {
    if (!code) return "Code is required."
    if (!CODE_PATTERN.test(code) || code.length > 32)
      return "Code may contain only letters, digits, - and _ (up to 32)."
    if (!name.trim()) return "Name is required."
    if (!releaseDate) return "Release date is required."
    if (!seriesId) return "Series is required."
    return null
  }

  function handleSubmit(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault()
    setSaved(null)
    const problem = validate()
    setError(problem)
    if (problem) return

    const fields = { name: name.trim(), releaseDate, seriesId }
    const finish = () => {
      setSaved(`Saved ${code}.`)
      void queryClient.invalidateQueries({
        queryKey: getListCatalogSeriesQueryKey(),
      })
    }
    const failed = () => setError("Could not save the set.")

    if (existing) {
      update.mutate(
        { id: existing, data: fields },
        {
          onSuccess: (response) =>
            response.status === 200
              ? finish()
              : setError(response.data.detail ?? "Could not save the set."),
          onError: failed,
        }
      )
      return
    }
    create.mutate(
      { data: { code, ...fields } },
      {
        onSuccess: (response) =>
          response.status === 201
            ? finish()
            : setError(response.data.detail ?? "Could not save the set."),
        onError: failed,
      }
    )
  }

  return (
    <form onSubmit={handleSubmit} noValidate aria-label="Expansion Set">
      <FieldGroup>
        <Field>
          <FieldLabel htmlFor="existing-set">Existing set</FieldLabel>
          <select
            id="existing-set"
            className={SELECT_CLASS}
            value={existing}
            onChange={(event) => chooseExisting(event.target.value)}
          >
            <option value="">New set</option>
            {sets.map((set) => (
              <option key={set.id} value={set.id}>
                {set.code} - {set.name}
              </option>
            ))}
          </select>
          <FieldDescription>
            Pick a set to load it and edit it. Saving replaces its name, release
            date and Series with the form.
          </FieldDescription>
        </Field>
        <Field>
          <FieldLabel htmlFor="set-code">Code</FieldLabel>
          <Input
            id="set-code"
            value={code}
            readOnly={existing !== ""}
            onChange={(event) => setCode(event.target.value.trim())}
          />
          <FieldDescription>
            A new set's code cannot be changed afterwards, and a code that
            already exists is refused.
          </FieldDescription>
        </Field>
        <Field>
          <FieldLabel htmlFor="set-name">Name</FieldLabel>
          <Input
            id="set-name"
            value={name}
            onChange={(event) => setName(event.target.value)}
          />
        </Field>
        <Field>
          <FieldLabel htmlFor="set-release-date">Release date</FieldLabel>
          <Input
            id="set-release-date"
            type="date"
            value={releaseDate}
            onChange={(event) => setReleaseDate(event.target.value)}
          />
        </Field>
        <Field>
          <FieldLabel htmlFor="set-series">Series</FieldLabel>
          <select
            id="set-series"
            className={SELECT_CLASS}
            value={seriesId}
            onChange={(event) => setSeriesId(event.target.value)}
          >
            <option value="">Choose a Series</option>
            {series.map((s) => (
              <option key={s.id} value={s.id}>
                {s.name}
              </option>
            ))}
          </select>
        </Field>
        {error && <FieldError>{error}</FieldError>}
        {saved && <p role="status">{saved}</p>}
        <Field>
          <Button type="submit" disabled={pending}>
            {pending ? "Saving..." : "Save set"}
          </Button>
        </Field>
      </FieldGroup>
    </form>
  )
}
