import { useCallback } from "react"
import type { ReactNode } from "react"
import { CardTile } from "@/components/catalog/card-tile"
import { VirtualGrid } from "@/components/catalog/virtual-grid"
import { Button } from "@/components/ui/button"
import { Skeleton } from "@/components/ui/skeleton"
import type { CardSummary } from "@/generated/models"

export interface InfiniteCardResultsProps {
  cards: CardSummary[]
  total: number
  isPending: boolean
  isError: boolean
  errorMessage?: string
  emptyMessage: string
  hasNextPage: boolean
  isFetching: boolean
  /** Starts the next page; the caller must ignore it while a fetch is in flight. */
  onLoadMore: () => void
  renderControl?: (card: CardSummary) => ReactNode
}

const getCardKey = (card: CardSummary) => card.id

/**
 * Results for an infinite card query: loading, error and empty states, a
 * virtualized grid that loads the next page near the end, and a "Load more"
 * button that is both the fallback control and a trigger.
 */
export function InfiniteCardResults({
  cards,
  total,
  isPending,
  isError,
  errorMessage,
  emptyMessage,
  hasNextPage,
  isFetching,
  onLoadMore,
  renderControl,
}: InfiniteCardResultsProps) {
  const renderCard = useCallback(
    (card: CardSummary) => (
      <CardTile card={card} control={renderControl?.(card)} />
    ),
    [renderControl]
  )

  const canLoadMore = hasNextPage && !isFetching
  const showList = !isPending && cards.length > 0

  let body: ReactNode
  if (isPending) {
    body = (
      <div
        className="grid grid-cols-2 gap-4 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-5"
        aria-busy="true"
        aria-label="Loading cards"
      >
        {Array.from({ length: 12 }).map((_, index) => (
          // oxlint-disable-next-line no-array-index-key -- fixed-size skeleton placeholders, never reordered
          <Skeleton key={index} className="aspect-[5/7] w-full rounded-4xl" />
        ))}
      </div>
    )
  } else if (isError && cards.length === 0) {
    body = (
      <p role="alert" className="text-sm text-destructive">
        {errorMessage ?? "Could not load cards. Please try again."}
      </p>
    )
  } else if (cards.length === 0) {
    body = <p className="text-sm text-muted-foreground">{emptyMessage}</p>
  } else {
    body = (
      <VirtualGrid
        items={cards}
        getKey={getCardKey}
        renderItem={renderCard}
        // A failed page is retried by the button, never by scrolling alone.
        canLoadMore={canLoadMore && !isError}
        onLoadMore={onLoadMore}
      />
    )
  }

  return (
    <div className="flex flex-col gap-6">
      {body}
      {showList && isError && (
        <p role="alert" className="text-center text-sm text-destructive">
          {errorMessage ?? "Could not load more cards."}
        </p>
      )}
      {/* Always mounted: a live region that appears together with its text is often not announced. */}
      <p
        className="text-center text-sm text-muted-foreground empty:sr-only"
        aria-live="polite"
      >
        {showList ? `${cards.length} of ${total} cards loaded` : ""}
      </p>
      {showList && (
        <Button
          type="button"
          variant="outline"
          size="sm"
          aria-disabled={!canLoadMore}
          className="self-center aria-disabled:cursor-not-allowed aria-disabled:opacity-50"
          onClick={() => {
            if (canLoadMore) onLoadMore()
          }}
        >
          Load more
        </Button>
      )}
    </div>
  )
}
