import { useInfiniteQuery } from "@tanstack/react-query"
import { DropdownMenuItem } from "@/components/ui/dropdown-menu"
import {
  getListScanReprintsQueryKey,
  listScanReprints,
} from "@/generated/endpoints/scan/scan"
import type { CardSummary } from "@/generated/models"
import { CardThumb } from "./card-thumb"

// The API pages the Reprints five at a time, and offsets count from the first.
const PAGE_SIZE = 5

/**
 * The Reprints of the Cards a Scan holds, five at a time, as menu items. It
 * fetches only when mounted, so the host mounts it on open. Picking a Reprint
 * passes the Card and its position across all pages. The host must be a
 * dropdown menu: the rows and Load more are its items.
 */
export function OtherPrints({
  cardIds,
  onPick,
}: {
  /** The Card ids the Scan holds, in the order the matcher offered them. */
  cardIds: string[]
  onPick: (card: CardSummary, position: number) => void
}) {
  const query = useInfiniteQuery({
    queryKey: ["infinite", ...getListScanReprintsQueryKey({ cardIds })],
    queryFn: async ({ pageParam, signal }) => {
      const response = await listScanReprints(
        { cardIds, offset: pageParam },
        { signal }
      )
      if (response.status !== 200) throw new Error("reprints failed")
      return response.data.data
    },
    initialPageParam: 0,
    getNextPageParam: (last, pages) =>
      last.hasMore ? pages.length * PAGE_SIZE : undefined,
  })

  // The Scan's own Cards never show, and a Card is listed once even if two pages send it.
  const sent = new Set(cardIds)
  const listed = new Set<string>()
  const cards = (query.data?.pages ?? [])
    .flatMap((page) => page.cards ?? [])
    .filter((card) => {
      if (sent.has(card.id) || listed.has(card.id)) return false
      listed.add(card.id)
      return true
    })

  return (
    <>
      {query.isPending && (
        <p className="px-3 py-2 text-sm">Loading other prints…</p>
      )}
      {cards.map((card, position) => (
        <DropdownMenuItem
          key={card.id}
          className="py-1.5"
          onClick={() => onPick(card, position)}
        >
          <CardThumb card={card} />
          <span className="flex min-w-0 flex-1 flex-col">
            <span className="truncate text-sm font-medium">{card.name}</span>
            <span className="text-xs">
              {card.expansionSet.code} · No. {card.localId}
            </span>
          </span>
        </DropdownMenuItem>
      ))}
      {query.isSuccess && cards.length === 0 && !query.hasNextPage && (
        <p className="px-3 py-2 text-sm">No other prints</p>
      )}
      {query.isError && (
        <>
          <p role="alert" className="px-3 py-2 text-sm text-destructive">
            Could not load other prints.
          </p>
          <DropdownMenuItem
            closeOnClick={false}
            onClick={() => void query.refetch()}
          >
            Retry
          </DropdownMenuItem>
        </>
      )}
      {!query.isError && query.hasNextPage && (
        <DropdownMenuItem
          closeOnClick={false}
          disabled={query.isFetchingNextPage}
          onClick={() => void query.fetchNextPage()}
        >
          Load more
        </DropdownMenuItem>
      )}
    </>
  )
}
