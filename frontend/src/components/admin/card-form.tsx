import { useEffect, useRef, useState } from "react"
import { useMutation, useQuery } from "@tanstack/react-query"
import { SELECT_CLASS, flattenSets } from "@/components/admin/form-shared"
import { Button } from "@/components/ui/button"
import {
  Field,
  FieldDescription,
  FieldError,
  FieldGroup,
  FieldLabel,
} from "@/components/ui/field"
import { Input } from "@/components/ui/input"
import { upsertAdminCard } from "@/generated/endpoints/admin/admin"
import {
  getListCatalogRaritiesQueryOptions,
  getListCatalogSeriesQueryOptions,
} from "@/generated/endpoints/catalog/catalog"
import type { UpsertCardInputBody } from "@/generated/models"
import {
  CardUploadError,
  uploadCardImage,
  validateCardImage,
} from "@/lib/admin-card-upload"

// What the admin picks as a card's type. Item, Supporter and Stadium are
// Trainer cards told apart by their tag; Pokémon and Energi carry no tag.
const KINDS = ["Pokémon", "Item", "Supporter", "Stadium", "Energi"] as const
type Kind = (typeof KINDS)[number]

function fieldsOf(kind: Kind): Pick<UpsertCardInputBody, "category" | "tag"> {
  if (kind === "Item" || kind === "Supporter" || kind === "Stadium") {
    return { category: "Trainer", tag: kind }
  }
  return { category: kind }
}

/** "009" -> "010", "99" -> "100": the next number, keeping the entered padding. */
function nextCollectorNumber(localId: string): string {
  const match = /^(.*?)(\d+)$/.exec(localId)
  if (!match) return ""
  return match[1] + String(Number(match[2]) + 1).padStart(match[2].length, "0")
}

interface Saved {
  setCode: string
  localId: string
}

/**
 * Enters or corrects one Card with its photographed image (docs/adr/0019):
 * saves the Card, then uploads the image straight to the bucket and confirms.
 * Saving the same set and number again updates that Card. A Card whose upload
 * failed stays out of the catalog until a later save uploads its image.
 */
export function CardForm() {
  const series = useQuery(getListCatalogSeriesQueryOptions())
  const rarities = useQuery(getListCatalogRaritiesQueryOptions())

  const browse = series.data?.status === 200 ? series.data.data.data : null
  const sets = flattenSets(browse)
  const rarityOptions =
    rarities.data?.status === 200 ? (rarities.data.data.data ?? []) : []

  const [setId, setSetId] = useState("")
  const [localId, setLocalId] = useState("")
  const [name, setName] = useState("")
  const [kind, setKind] = useState<Kind>("Pokémon")
  const [rarityId, setRarityId] = useState("")
  const [illustrator, setIllustrator] = useState("")
  const [file, setFile] = useState<File | null>(null)
  const [preview, setPreview] = useState<string | null>(null)
  const fileInput = useRef<HTMLInputElement>(null)
  const picked = useRef(0)
  const [error, setError] = useState<string | null>(null)
  const [saved, setSaved] = useState<Saved | null>(null)

  const save = useMutation({
    mutationFn: async (body: UpsertCardInputBody) => {
      const card = await upsertAdminCard(body)
      if (card.status !== 200) {
        throw new CardUploadError(
          card.data.detail ?? `Could not save the card (${card.status}).`
        )
      }
      await uploadCardImage(card.data.data.id, file!)
    },
  })

  useEffect(() => {
    if (!file) return
    const url = URL.createObjectURL(file)
    setPreview(url)
    return () => {
      URL.revokeObjectURL(url)
      setPreview(null)
    }
  }, [file])

  // Checks the file before it can be saved, so a mistake shows at once and
  // nothing is sent for it. Only the latest pick counts.
  async function handlePick(chosen: File | undefined) {
    const pick = ++picked.current
    setFile(null)
    setError(null)
    if (!chosen) return
    try {
      await validateCardImage(chosen)
    } catch (err) {
      if (pick !== picked.current) return
      setError(
        err instanceof CardUploadError
          ? err.message
          : "Could not read the image."
      )
      return
    }
    if (pick === picked.current) setFile(chosen)
  }

  function handleSubmit(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault()
    submit(false)
  }

  /** Saves the card; with `advance`, keeps the set and readies the next number. */
  function submit(advance: boolean) {
    setError(null)
    setSaved(null)

    if (!setId || !localId.trim() || !name.trim()) {
      setError("Choose an expansion set and enter a collector number and name.")
      return
    }
    if (!file) {
      setError("Choose an image: a card cannot be saved without one.")
      return
    }

    const setCode = sets.find((s) => s.id === setId)?.code ?? ""
    const body: UpsertCardInputBody = {
      expansionSetId: setId,
      localId: localId.trim(),
      name: name.trim(),
      ...fieldsOf(kind),
    }
    if (rarityId) body.rarityId = rarityId
    if (illustrator.trim()) body.illustrator = illustrator.trim()

    save.mutate(body, {
      onSuccess: () => {
        setSaved({ setCode, localId: localId.trim() })
        if (!advance) return
        setLocalId(nextCollectorNumber(localId.trim()))
        setName("")
        picked.current++
        setFile(null)
        if (fileInput.current) fileInput.current.value = ""
      },
      onError: (err) =>
        setError(
          err instanceof CardUploadError
            ? err.message
            : "Could not save the card."
        ),
    })
  }

  return (
    <form onSubmit={handleSubmit} noValidate aria-label="Card">
      <FieldGroup>
        <Field>
          <FieldLabel htmlFor="card-set">Expansion set</FieldLabel>
          <select
            id="card-set"
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
          <FieldLabel htmlFor="card-number">Collector number</FieldLabel>
          <Input
            id="card-number"
            value={localId}
            onChange={(e) => setLocalId(e.target.value)}
          />
          <FieldDescription>
            An existing number in the set updates that card.
          </FieldDescription>
        </Field>
        <Field>
          <FieldLabel htmlFor="card-name">Name</FieldLabel>
          <Input
            id="card-name"
            value={name}
            onChange={(e) => setName(e.target.value)}
          />
        </Field>
        <Field>
          <FieldLabel htmlFor="card-kind">Type</FieldLabel>
          <select
            id="card-kind"
            className={SELECT_CLASS}
            value={kind}
            onChange={(e) =>
              setKind(KINDS.find((k) => k === e.target.value) ?? "Pokémon")
            }
          >
            {KINDS.map((k) => (
              <option key={k} value={k}>
                {k}
              </option>
            ))}
          </select>
        </Field>
        <Field>
          <FieldLabel htmlFor="card-rarity">Rarity</FieldLabel>
          <select
            id="card-rarity"
            className={SELECT_CLASS}
            value={rarityId}
            onChange={(e) => setRarityId(e.target.value)}
          >
            <option value="">Default (Tanpa Tanda)</option>
            {rarityOptions.map((r) => (
              <option key={r.id} value={r.id}>
                {r.name}
              </option>
            ))}
          </select>
        </Field>
        <Field>
          <FieldLabel htmlFor="card-illustrator">Illustrator</FieldLabel>
          <Input
            id="card-illustrator"
            value={illustrator}
            onChange={(e) => setIllustrator(e.target.value)}
          />
        </Field>
        <Field>
          <FieldLabel htmlFor="card-image">Image</FieldLabel>
          <Input
            id="card-image"
            ref={fileInput}
            type="file"
            accept="image/jpeg,image/png,image/webp"
            onChange={(e) => void handlePick(e.target.files?.[0])}
          />
          <FieldDescription>
            JPEG, PNG or WebP, a portrait photo of the card, up to 10 MB.
            Required.
          </FieldDescription>
          {preview && (
            <img
              src={preview}
              alt="Preview of the chosen image"
              className="max-h-64 w-fit rounded-xl"
            />
          )}
        </Field>
        {error && <FieldError>{error}</FieldError>}
        {saved && (
          <p role="status">
            Saved {saved.setCode} {saved.localId}.
          </p>
        )}
        <Field>
          <Button type="submit" disabled={save.isPending}>
            {save.isPending ? "Saving..." : "Save"}
          </Button>
          <Button
            type="button"
            variant="outline"
            disabled={save.isPending}
            onClick={() => submit(true)}
          >
            Save and next
          </Button>
        </Field>
      </FieldGroup>
    </form>
  )
}
