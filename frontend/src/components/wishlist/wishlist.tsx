import { useEffect, useState } from "react"
import { keepPreviousData, useQueryClient } from "@tanstack/react-query"
import { RiAddLine, RiSubtractLine } from "@remixicon/react"
import { CatalogFilterPanel } from "@/components/catalog/filter-panel"
import { InfiniteCardResults } from "@/components/catalog/infinite-card-results"
import { Button } from "@/components/ui/button"
import { PresetSheet } from "@/components/wishlist/preset-sheet"
import {
  useListCatalogFacets,
  useListCatalogSeries,
} from "@/generated/endpoints/catalog/catalog"
import {
  useClearFulfilledWishlist,
  useListWishlistFacets,
  useRemoveWishlistItem,
  useSetWishlistItem,
} from "@/generated/endpoints/wishlist/wishlist"
import type { WishlistItem } from "@/generated/models"
import { NETWORK_ERROR } from "@/lib/collections"
import { useInfiniteCardResultsProps } from "@/lib/infinite-catalog-cards"
import {
  hasActiveWishlistFilters,
  invalidateWishlist,
  isWishlistStatus,
  useInfiniteWishlist,
  wishlistStatus,
} from "@/lib/wishlist"
import type { WishlistFilters } from "@/lib/wishlist"

const STATUS_OPTIONS = [
  { value: "all", label: "All" },
  { value: "needed", label: "Still needed" },
  { value: "fulfilled", label: "Fulfilled" },
]

interface WishlistProps {
  search: WishlistFilters
  /** Receives the next filters; the route writes them to the URL. */
  onSearchChange: (next: WishlistFilters) => void
}

/**
 * The Wishlist: the Cards the user wants, each with how many they own of how
 * many they want, the derived status, and controls to change or remove it.
 */
export function Wishlist({ search, onSearchChange }: WishlistProps) {
  const queryClient = useQueryClient()
  const query = useInfiniteWishlist(search)
  const facetsQuery = useListWishlistFacets(
    { ...search, status: wishlistStatus(search) },
    { query: { placeholderData: keepPreviousData } }
  )
  const seriesQuery = useListCatalogSeries()
  // The Preset adds Cards that are NOT on the Wishlist yet, so its options come
  // from the whole catalog: a Wishlist facet is computed from the Cards already
  // on it, which the Preset excludes anyway.
  const presetFacetsQuery = useListCatalogFacets(undefined, {
    query: { placeholderData: keepPreviousData },
  })
  const clearFulfilled = useClearFulfilledWishlist()

  const results = useInfiniteCardResultsProps(query)
  const items = query.data?.items
  const facets =
    facetsQuery.data?.status === 200 ? facetsQuery.data.data.data : undefined
  const series =
    seriesQuery.data?.status === 200
      ? (seriesQuery.data.data.data?.series ?? [])
      : []
  const presetFacets =
    presetFacetsQuery.data?.status === 200
      ? presetFacetsQuery.data.data.data
      : undefined
  // Always mounted: a live region that appears together with its text is often not announced.
  const [announcement, setAnnouncement] = useState("")
  const [clearError, setClearError] = useState("")
  const [presetOpen, setPresetOpen] = useState(false)

  function clearFulfilledItems() {
    setClearError("")
    clearFulfilled.mutate(undefined, {
      onSuccess: (response) => {
        if (response.status !== 200) {
          setClearError(
            response.data.detail ?? "Could not clear the fulfilled Cards."
          )
          return
        }
        const removed = response.data.data?.removed ?? 0
        setAnnouncement(
          removed === 0
            ? "No fulfilled Cards to clear."
            : `Cleared ${removed} fulfilled ${removed === 1 ? "Card" : "Cards"}.`
        )
        invalidateWishlist(queryClient)
      },
      onError: () => setClearError(NETWORK_ERROR),
    })
  }

  return (
    <section className="flex flex-col gap-6" aria-label="Wishlist contents">
      <CatalogFilterPanel
        search={search}
        facets={facets}
        series={series}
        onChange={(patch) => onSearchChange({ ...search, ...patch })}
        onClear={() => onSearchChange({ status: wishlistStatus(search) })}
        statusFilter={{
          value: wishlistStatus(search),
          options: STATUS_OPTIONS,
          onChange: (status) => {
            if (!isWishlistStatus(status)) return
            onSearchChange({ ...search, status })
          },
        }}
      />

      <div className="flex flex-wrap items-center justify-between gap-3">
        <div className="flex flex-wrap gap-2">
          <Button
            type="button"
            variant="outline"
            size="sm"
            aria-disabled={clearFulfilled.isPending}
            className="aria-disabled:cursor-not-allowed aria-disabled:opacity-50"
            onClick={() => {
              if (!clearFulfilled.isPending) clearFulfilledItems()
            }}
          >
            Clear fulfilled
          </Button>
          <Button
            type="button"
            variant="outline"
            size="sm"
            onClick={() => setPresetOpen(true)}
          >
            Fill from sets
          </Button>
        </div>
        <p role="status" className="text-sm empty:sr-only">
          {announcement}
        </p>
        {clearError && (
          <p role="alert" className="text-sm text-destructive">
            {clearError}
          </p>
        )}
      </div>

      <InfiniteCardResults
        {...results}
        emptyMessage={
          hasActiveWishlistFilters(search)
            ? "No Cards on your Wishlist match these filters."
            : "Your Wishlist is empty. Add Cards from the catalog."
        }
        renderControl={(card) => {
          const item = items?.[card.id]
          if (!item) return null
          return (
            <WishlistItemControl item={item} onAnnounce={setAnnouncement} />
          )
        }}
      />

      <PresetSheet
        open={presetOpen}
        onOpenChange={setPresetOpen}
        expansionSets={presetFacets?.expansionSets ?? []}
        rarities={presetFacets?.rarities ?? []}
        categories={presetFacets?.categories ?? []}
        tags={presetFacets?.tags ?? []}
      />
    </section>
  )
}

/**
 * One item's progress and controls. The wanted quantity is optimistic until the
 * server answers; the status is not, because it is derived on read.
 */
function WishlistItemControl({
  item,
  onAnnounce,
}: {
  item: WishlistItem
  onAnnounce: (message: string) => void
}) {
  const queryClient = useQueryClient()
  const save = useSetWishlistItem()
  const remove = useRemoveWishlistItem()
  const [wanted, setWanted] = useState<number | null>(null)
  const [error, setError] = useState("")
  const name = item.card.name
  const shown = wanted ?? item.wantedQuantity
  const pending = save.isPending || remove.isPending

  // Fresh server data wins over the optimistic value once it arrives.
  useEffect(() => {
    setWanted(null)
  }, [item.wantedQuantity])

  function setQuantity(quantity: number) {
    if (quantity < 1 || pending) return
    setError("")
    setWanted(quantity)
    save.mutate(
      { cardId: item.card.id, data: { quantity } },
      {
        onSuccess: (response) => {
          if (response.status !== 204) {
            setWanted(null)
            setError(
              response.data.detail ?? "Could not change this wanted quantity."
            )
            return
          }
          onAnnounce(`Want ${quantity} of ${name}.`)
          invalidateWishlist(queryClient)
        },
        onError: () => {
          setWanted(null)
          setError(NETWORK_ERROR)
        },
      }
    )
  }

  function removeItem() {
    if (pending) return
    setError("")
    remove.mutate(
      { cardId: item.card.id },
      {
        onSuccess: (response) => {
          if (response.status !== 204) {
            setError(response.data.detail ?? "Could not remove this Card.")
            return
          }
          onAnnounce(`Removed ${name} from your Wishlist.`)
          invalidateWishlist(queryClient)
        },
        onError: () => setError(NETWORK_ERROR),
      }
    )
  }

  return (
    <div className="flex flex-col gap-1.5">
      <p className="text-sm font-medium">
        <span className="sr-only">
          {item.ownedQuantity} owned of {shown} wanted
        </span>
        <span aria-hidden="true">
          {item.ownedQuantity} / {shown}
        </span>
      </p>
      {item.fulfilled && <p className="text-xs font-medium">Fulfilled</p>}
      <div className="flex flex-wrap items-center gap-1">
        <Button
          type="button"
          variant="outline"
          size="icon-sm"
          aria-label={`Decrease wanted quantity of ${name}`}
          disabled={shown <= 1 || pending}
          onClick={() => setQuantity(shown - 1)}
        >
          <RiSubtractLine />
        </Button>
        <Button
          type="button"
          variant="outline"
          size="icon-sm"
          aria-label={`Increase wanted quantity of ${name}`}
          disabled={pending}
          onClick={() => setQuantity(shown + 1)}
        >
          <RiAddLine />
        </Button>
        <Button
          type="button"
          variant="ghost"
          size="sm"
          aria-label={`Remove ${name} from your Wishlist`}
          disabled={pending}
          onClick={removeItem}
        >
          Remove
        </Button>
      </div>
      {error && (
        <p role="alert" className="text-xs text-destructive">
          {error}
        </p>
      )}
    </div>
  )
}
