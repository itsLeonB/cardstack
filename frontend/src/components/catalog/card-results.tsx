import type { ReactNode } from "react"
import { CardTile } from "@/components/catalog/card-tile"
import { Skeleton } from "@/components/ui/skeleton"
import { Button } from "@/components/ui/button"
import { RiArrowLeftSLine, RiArrowRightSLine } from "@remixicon/react"
import type { CardSummary } from "@/generated/models"

export interface CardResultsProps {
  cards: CardSummary[]
  total: number
  page: number
  limit: number
  isPending: boolean
  isError: boolean
  errorMessage?: string
  emptyMessage: string
  onPageChange: (page: number) => void
  renderControl?: (card: CardSummary) => ReactNode
}

/**
 * Shared results grid for both the Expansion Set browse page and the catalog
 * search page: handles the loading/error/empty states and Previous/Next
 * pagination so the two routes only need to supply the query state.
 */
export function CardResults({
  cards,
  total,
  page,
  limit,
  isPending,
  isError,
  errorMessage,
  emptyMessage,
  onPageChange,
  renderControl,
}: CardResultsProps) {
  const totalPages = Math.max(1, Math.ceil(total / limit))

  if (isPending) {
    return (
      <div
        className="grid grid-cols-2 gap-4 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-5"
        aria-busy="true"
        aria-label="Loading cards"
      >
        {Array.from({ length: limit }).map((_, index) => (
          // oxlint-disable-next-line no-array-index-key -- fixed-size skeleton placeholders, never reordered
          <Skeleton key={index} className="aspect-[5/7] w-full rounded-4xl" />
        ))}
      </div>
    )
  }

  if (isError) {
    return (
      <p role="alert" className="text-sm text-destructive">
        {errorMessage ?? "Could not load cards. Please try again."}
      </p>
    )
  }

  if (cards.length === 0) {
    return <p className="text-sm text-muted-foreground">{emptyMessage}</p>
  }

  return (
    <div className="flex flex-col gap-6">
      <p className="text-sm text-muted-foreground" aria-live="polite">
        {total} {total === 1 ? "card" : "cards"}
      </p>
      <ul className="grid grid-cols-2 gap-4 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-5">
        {cards.map((card) => (
          <li key={card.id}>
            <CardTile card={card} control={renderControl?.(card)} />
          </li>
        ))}
      </ul>
      {totalPages > 1 && (
        <nav
          aria-label="Pagination"
          className="flex items-center justify-center gap-3"
        >
          <Button
            type="button"
            variant="outline"
            size="sm"
            disabled={page <= 1}
            onClick={() => onPageChange(page - 1)}
          >
            <RiArrowLeftSLine data-icon="inline-start" />
            Previous
          </Button>
          <span className="text-sm text-muted-foreground" aria-live="polite">
            Page {page} of {totalPages}
          </span>
          <Button
            type="button"
            variant="outline"
            size="sm"
            disabled={page >= totalPages}
            onClick={() => onPageChange(page + 1)}
          >
            Next
            <RiArrowRightSLine data-icon="inline-end" />
          </Button>
        </nav>
      )}
    </div>
  )
}
