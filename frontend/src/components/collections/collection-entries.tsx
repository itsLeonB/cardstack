import { useEffect, useRef, useState } from "react"
import { keepPreviousData, useQueryClient } from "@tanstack/react-query"
import { useListCatalogSeries } from "@/generated/endpoints/catalog/catalog"
import { useListCollectionFacets } from "@/generated/endpoints/inventory/inventory"
import { CardPositions } from "@/components/collections/card-positions"
import { QuantityControl } from "@/components/collections/quantity-control"
import type { BinderSlotTarget } from "@/generated/models"
import { InfiniteCardResults } from "@/components/catalog/infinite-card-results"
import { CatalogFilterPanel } from "@/components/catalog/filter-panel"
import type { CatalogFilters } from "@/lib/catalog-search"
import { hasActiveFilters } from "@/lib/catalog-search"
import {
  invalidateCollectionCounts,
  invalidateCollectionEntries,
  patchCollectionEntryQuantities,
} from "@/lib/collections"
import { useInfiniteCardResultsProps } from "@/lib/infinite-catalog-cards"
import { useInfiniteCollectionEntries } from "@/lib/infinite-inventory"
import { invalidateMasterInventory } from "@/lib/master-inventory"
import { useQuantityBatch } from "@/lib/use-quantity-batch"
import { Button } from "@/components/ui/button"
import { CollectionMoveBar } from "@/components/collections/collection-move"

// The Move request takes at most 100 Card ids.
const MOVE_LIMIT = 100

interface CollectionEntriesProps {
  collectionId: string
  search: CatalogFilters
  /** Receives the next filters; the route writes them to the URL. */
  onSearchChange: (next: CatalogFilters) => void
  /** Opens the binder on a position's Page. */
  onFindPosition: (position: BinderSlotTarget) => void
}

export function CollectionEntries({
  collectionId,
  search,
  onSearchChange,
  onFindPosition,
}: CollectionEntriesProps) {
  const queryClient = useQueryClient()
  // A save writes the confirmed quantities into the cached pages instead of
  // refetching them: a refetch costs a request per loaded page and would drop
  // a card saved at 0, whose tile stays until the next real refetch.
  const batch = useQuantityBatch(collectionId, async (results) => {
    invalidateMasterInventory(queryClient)
    invalidateCollectionCounts(queryClient, collectionId)
    // No results means the response was lost: the write may have committed, so
    // there is nothing to patch from and the list must be refetched.
    if (results.length === 0)
      invalidateCollectionEntries(queryClient, collectionId)
    else
      await patchCollectionEntryQuantities(queryClient, collectionId, results)
  })

  const query = useInfiniteCollectionEntries(
    collectionId,
    search,
    () => !batch.isBusy()
  )
  const facetsQuery = useListCollectionFacets(collectionId, search, {
    query: { placeholderData: keepPreviousData },
  })
  const seriesQuery = useListCatalogSeries()

  const results = useInfiniteCardResultsProps(query)
  const serverQuantities = query.data?.quantities
  const serverPositions = query.data?.positions
  const facets =
    facetsQuery.data?.status === 200 ? facetsQuery.data.data.data : undefined
  const series =
    seriesQuery.data?.status === 200
      ? (seriesQuery.data.data.data?.series ?? [])
      : []

  // Fresh server data replaces optimistic values. Appending a page or patching
  // a save changes `data` too, but prune keeps what the cache still agrees with.
  useEffect(
    () => batch.prune((cardId) => serverQuantities?.[cardId]),
    [query.data]
  )

  // Rapid filter clicks build on each other (pendingSearch is what the panel
  // shows meanwhile); the drain flushes edits until none are left, then writes
  // the final filters to the URL once.
  const [pendingSearch, setPendingSearch] = useState<CatalogFilters | null>(
    null
  )
  const target = useRef<CatalogFilters | null>(null)

  async function drain() {
    do {
      await batch.flush()
    } while (batch.hasPending())
    const next = target.current
    target.current = null
    setPendingSearch(null)
    if (next) onSearchChange(next)
  }

  function changeSearch(update: (current: CatalogFilters) => CatalogFilters) {
    const next = update(pendingSearch ?? search)
    const draining = target.current !== null
    target.current = next
    setPendingSearch(next)
    if (!draining) void drain()
  }

  // Selection mode. Each ticked Card keeps its copies, so the Move's count
  // holds even for a Card whose page is no longer loaded after a filter change.
  const [selecting, setSelecting] = useState(false)
  const [selected, setSelected] = useState<Map<string, number>>(new Map())
  const [announcement, setAnnouncement] = useState("")
  const atLimit = selected.size >= MOVE_LIMIT
  const copies = [...selected.values()].reduce((sum, n) => sum + n, 0)

  function toggleSelected(cardId: string, quantity: number) {
    setSelected((current) => {
      const next = new Map(current)
      if (next.has(cardId)) next.delete(cardId)
      else if (next.size < MOVE_LIMIT) next.set(cardId, quantity)
      return next
    })
  }

  function leaveSelection() {
    setSelecting(false)
    setSelected(new Map())
  }

  return (
    <section className="flex flex-col gap-6" aria-label="Collection contents">
      <CatalogFilterPanel
        search={pendingSearch ?? search}
        facets={facets}
        series={series}
        onChange={(patch) =>
          changeSearch((current) => ({ ...current, ...patch }))
        }
        onClear={() => changeSearch(() => ({}))}
      />

      <div className="flex flex-wrap items-center justify-between gap-3">
        {!selecting && (
          <Button
            variant="outline"
            size="sm"
            onClick={() => {
              setAnnouncement("")
              setSelecting(true)
            }}
          >
            Select
          </Button>
        )}
        {/* Always mounted: a live region that appears together with its text is often not announced. */}
        <p role="status" className="text-sm empty:sr-only">
          {atLimit ? `Up to ${MOVE_LIMIT} cards at a time` : ""}
        </p>
        <p role="status" className="text-sm empty:sr-only">
          {announcement}
        </p>
      </div>

      <InfiniteCardResults
        {...results}
        emptyMessage={
          hasActiveFilters(search)
            ? "No Cards in this Collection match these filters."
            : "This Collection has no Cards yet. Add some from the catalog."
        }
        renderControl={(card) => {
          if (selecting) {
            // The quantity shown in the list is the one the Move counts.
            const quantity =
              batch.quantities[card.id] ?? serverQuantities?.[card.id] ?? 0
            const checked = selected.has(card.id)
            return (
              <input
                type="checkbox"
                aria-label={`Select ${card.name} (${card.expansionSet.code} ${card.localId})`}
                className="size-5 accent-primary"
                checked={checked}
                disabled={!checked && atLimit}
                onChange={() => toggleSelected(card.id, quantity)}
              />
            )
          }
          // A binder's Cards carry positions and change only by placing Slots
          // (ADR-0020), so they show where they sit instead of a quantity.
          const positions = serverPositions?.[card.id]
          if (positions)
            return (
              <CardPositions positions={positions} onFind={onFindPosition} />
            )
          const server = serverQuantities?.[card.id] ?? 0
          return (
            <QuantityControl
              cardName={card.name}
              value={batch.quantities[card.id] ?? server}
              error={batch.errors[card.id]}
              onChange={(quantity) =>
                batch.setQuantity(card.id, quantity, server)
              }
            />
          )
        }}
      />

      {selecting && (
        <CollectionMoveBar
          collectionId={collectionId}
          cardIds={[...selected.keys()]}
          copies={copies}
          onCancel={leaveSelection}
          onMoved={(message) => {
            leaveSelection()
            setAnnouncement(message)
          }}
        />
      )}
    </section>
  )
}
