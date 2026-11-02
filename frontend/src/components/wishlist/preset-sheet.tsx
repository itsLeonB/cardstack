import { useState } from "react"
import { useQueryClient } from "@tanstack/react-query"
import { RiAddLine, RiSubtractLine } from "@remixicon/react"
import { Button } from "@/components/ui/button"
import {
  useApplyWishlistPreset,
  usePreviewWishlistPreset,
} from "@/generated/endpoints/wishlist/wishlist"
import { invalidateWishlist } from "@/lib/wishlist"
import type {
  ExpansionSetFacetOption,
  RarityFacetOption,
  StringFacetOption,
} from "@/generated/models"

interface PresetSheetProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  /** Available Expansion Sets to choose from */
  expansionSets: ExpansionSetFacetOption[]
  /** Available rarities to narrow by */
  rarities: RarityFacetOption[]
  /** Available categories to narrow by */
  categories: StringFacetOption[]
  /** Available tags to narrow by */
  tags: StringFacetOption[]
}

/**
 * A sheet for applying a Wishlist Preset: choose Expansion Sets, optionally
 * narrow by rarity/category/tag, set a quantity, preview how many Cards would be
 * added, and apply.
 */
export function PresetSheet({
  open,
  onOpenChange,
  expansionSets,
  rarities,
  categories,
  tags,
}: PresetSheetProps) {
  const queryClient = useQueryClient()
  const preview = usePreviewWishlistPreset()
  const apply = useApplyWishlistPreset()

  const [selectedSets, setSelectedSets] = useState<string[]>([])
  const [selectedRarities, setSelectedRarities] = useState<string[]>([])
  const [selectedCategories, setSelectedCategories] = useState<string[]>([])
  const [selectedTags, setSelectedTags] = useState<string[]>([])
  const [quantity, setQuantity] = useState(1)
  const [previewCount, setPreviewCount] = useState<number | null>(null)
  const [error, setError] = useState("")
  const [announcement, setAnnouncement] = useState("")

  function toggleSet(id: string) {
    setSelectedSets((prev) =>
      prev.includes(id) ? prev.filter((x) => x !== id) : [...prev, id]
    )
    setPreviewCount(null)
  }

  function toggleRarity(id: string) {
    setSelectedRarities((prev) =>
      prev.includes(id) ? prev.filter((x) => x !== id) : [...prev, id]
    )
    setPreviewCount(null)
  }

  function toggleCategory(value: string) {
    setSelectedCategories((prev) =>
      prev.includes(value) ? prev.filter((x) => x !== value) : [...prev, value]
    )
    setPreviewCount(null)
  }

  function toggleTag(value: string) {
    setSelectedTags((prev) =>
      prev.includes(value) ? prev.filter((x) => x !== value) : [...prev, value]
    )
    setPreviewCount(null)
  }

  function previewPreset() {
    if (selectedSets.length === 0) {
      setError("Select at least one Expansion Set.")
      return
    }
    setError("")
    preview.mutate(
      {
        data: {
          expansionSetIds: selectedSets,
          quantity,
          rarityIds: selectedRarities,
          categories: selectedCategories,
          tags: selectedTags,
        },
      },
      {
        onSuccess: (response) => {
          if (response.status !== 200) {
            setError(response.data.detail ?? "Could not preview the preset.")
            return
          }
          setPreviewCount(response.data.data?.count ?? 0)
        },
        onError: () => setError("Network error."),
      }
    )
  }

  function applyPreset() {
    if (selectedSets.length === 0) {
      setError("Select at least one Expansion Set.")
      return
    }
    setError("")
    apply.mutate(
      {
        data: {
          expansionSetIds: selectedSets,
          quantity,
          rarityIds: selectedRarities,
          categories: selectedCategories,
          tags: selectedTags,
        },
      },
      {
        onSuccess: (response) => {
          if (response.status !== 200) {
            setError(response.data.detail ?? "Could not apply the preset.")
            return
          }
          const count = response.data.data?.count ?? 0
          setAnnouncement(
            count === 0
              ? "No Cards were added."
              : `Added ${count} ${count === 1 ? "Card" : "Cards"} to your Wishlist.`
          )
          invalidateWishlist(queryClient)
          onOpenChange(false)
          setSelectedSets([])
          setSelectedRarities([])
          setSelectedCategories([])
          setSelectedTags([])
          setPreviewCount(null)
        },
        onError: () => setError("Network error."),
      }
    )
  }

  return (
    <section
      aria-labelledby="preset-heading"
      className={`flex min-h-0 flex-col border-t bg-background ${open ? "flex-[4] rounded-none" : "hidden"}`}
    >
      <div className="flex shrink-0 flex-wrap items-center gap-2 px-3 py-2">
        <h2
          id="preset-heading"
          className="flex-1 font-heading text-base font-medium"
        >
          Fill from sets
        </h2>
        <Button variant="outline" size="sm" onClick={() => onOpenChange(false)}>
          Cancel
        </Button>
      </div>

      <div className="flex min-h-0 flex-1 flex-col gap-4 overflow-y-auto px-3 pb-3">
        <div>
          <p className="mb-2 text-sm font-medium">
            I own none yet of these sets
          </p>
          <div className="flex flex-wrap gap-2">
            {expansionSets.map((set) => (
              <button
                key={set.id}
                type="button"
                className={`rounded-full px-3 py-1 text-sm ${selectedSets.includes(set.id) ? "bg-primary text-primary-foreground" : "bg-muted"}`}
                onClick={() => toggleSet(set.id)}
              >
                {set.name}
              </button>
            ))}
          </div>
        </div>

        {rarities.length > 0 && (
          <div>
            <p className="mb-2 text-sm font-medium">Rarity (optional)</p>
            <div className="flex flex-wrap gap-2">
              {rarities.map((rarity) => (
                <button
                  key={rarity.id}
                  type="button"
                  className={`rounded-full px-3 py-1 text-sm ${selectedRarities.includes(rarity.id) ? "bg-primary text-primary-foreground" : "bg-muted"}`}
                  onClick={() => toggleRarity(rarity.id)}
                >
                  {rarity.name}
                </button>
              ))}
            </div>
          </div>
        )}

        {categories.length > 0 && (
          <div>
            <p className="mb-2 text-sm font-medium">Category (optional)</p>
            <div className="flex flex-wrap gap-2">
              {categories.map((cat) => (
                <button
                  key={cat.value}
                  type="button"
                  className={`rounded-full px-3 py-1 text-sm ${selectedCategories.includes(cat.value) ? "bg-primary text-primary-foreground" : "bg-muted"}`}
                  onClick={() => toggleCategory(cat.value)}
                >
                  {cat.value}
                </button>
              ))}
            </div>
          </div>
        )}

        {tags.length > 0 && (
          <div>
            <p className="mb-2 text-sm font-medium">Tag (optional)</p>
            <div className="flex flex-wrap gap-2">
              {tags.map((tag) => (
                <button
                  key={tag.value}
                  type="button"
                  className={`rounded-full px-3 py-1 text-sm ${selectedTags.includes(tag.value) ? "bg-primary text-primary-foreground" : "bg-muted"}`}
                  onClick={() => toggleTag(tag.value)}
                >
                  {tag.value}
                </button>
              ))}
            </div>
          </div>
        )}

        <div>
          <p className="mb-2 text-sm font-medium">Quantity per Card</p>
          <div className="flex items-center gap-2">
            <Button
              type="button"
              variant="outline"
              size="icon-sm"
              disabled={quantity <= 1}
              onClick={() => setQuantity(quantity - 1)}
            >
              <RiSubtractLine />
            </Button>
            <span className="w-8 text-center">{quantity}</span>
            <Button
              type="button"
              variant="outline"
              size="icon-sm"
              onClick={() => setQuantity(quantity + 1)}
            >
              <RiAddLine />
            </Button>
          </div>
        </div>

        <div className="flex flex-wrap gap-2">
          <Button
            size="sm"
            disabled={preview.isPending || selectedSets.length === 0}
            onClick={previewPreset}
          >
            Preview
          </Button>
          {previewCount !== null && (
            <p role="status" className="self-center text-sm">
              {previewCount} {previewCount === 1 ? "Card" : "Cards"} will be
              added
            </p>
          )}
        </div>

        <Button
          size="sm"
          disabled={
            apply.isPending || previewCount === null || previewCount === 0
          }
          onClick={applyPreset}
        >
          Add to Wishlist
        </Button>

        {error && (
          <p role="alert" className="text-sm text-destructive">
            {error}
          </p>
        )}
        {announcement && (
          <p role="status" className="text-sm">
            {announcement}
          </p>
        )}
      </div>
    </section>
  )
}
