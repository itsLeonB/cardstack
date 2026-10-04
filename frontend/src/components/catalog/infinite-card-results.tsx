import { useCallback } from "react"
import type { ReactNode } from "react"
import { SignInLink } from "@/components/auth/sign-in-link"
import { CardTile } from "@/components/catalog/card-tile"
import { VirtualGrid } from "@/components/catalog/virtual-grid"
import { Button, buttonVariants } from "@/components/ui/button"
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
  /** A Guest sees the first page only: no next-page load, a sign-in prompt where it would be. */
  guest?: boolean
  /** The API refused with `login_required`: show a sign-in prompt instead of an error. */
  loginRequired?: boolean
  /** Path (with search) the sign-in prompts return to. */
  signInRedirect?: string
}

const getCardKey = (card: CardSummary) => card.id

/**
 * Results for an infinite card query: loading, error and empty states, a
 * virtualized grid that loads the next page near the end, and a "Load more"
 * button that is both the fallback control and a trigger. For a Guest the
 * button is a "Sign in to see more" link and nothing loads past page 1.
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
  guest = false,
  loginRequired = false,
  signInRedirect,
}: InfiniteCardResultsProps) {
  const renderCard = useCallback(
    (card: CardSummary) => (
      <CardTile card={card} control={renderControl?.(card)} />
    ),
    [renderControl]
  )

  const canLoadMore = hasNextPage && !isFetching && !guest
  const showList = !isPending && cards.length > 0
  const signInPrompt = (
    <p role="status" className="text-sm">
      <SignInLink redirect={signInRedirect}>Sign in</SignInLink> to see these
      results.
    </p>
  )

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
    body = loginRequired ? (
      signInPrompt
    ) : (
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
        totalCount={total}
        // A failed page is retried by the button, never by scrolling alone.
        canLoadMore={canLoadMore && !isError}
        onLoadMore={onLoadMore}
      />
    )
  }

  return (
    <div className="flex flex-col gap-6">
      {body}
      {showList &&
        isError &&
        (loginRequired ? (
          signInPrompt
        ) : (
          <p role="alert" className="text-center text-sm text-destructive">
            {errorMessage ?? "Could not load more cards."}
          </p>
        ))}
      {/* Always mounted: a live region that appears together with its text is often not announced. */}
      <p
        className="text-center text-sm text-muted-foreground empty:sr-only"
        aria-live="polite"
      >
        {showList ? `${cards.length} of ${total} cards loaded` : ""}
      </p>
      {/* Hidden once the last page is loaded; while fetching it stays, inert, so focus isn't lost. */}
      {showList && hasNextPage && !guest && (
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
      {/* Always mounted, like the status line above: a live region that appears together with its text is often not announced. */}
      <div aria-live="polite" className="self-center empty:hidden">
        {showList && hasNextPage && guest && (
          <SignInLink
            redirect={signInRedirect}
            className={buttonVariants({ variant: "outline", size: "sm" })}
          >
            Sign in to see more
          </SignInLink>
        )}
      </div>
    </div>
  )
}
