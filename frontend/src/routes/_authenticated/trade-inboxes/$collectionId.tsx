import { useAuth } from "@clerk/react"
import { useQuery } from "@tanstack/react-query"
import { createFileRoute } from "@tanstack/react-router"

import { InfiniteCardResults } from "@/components/catalog/infinite-card-results"
import { PageContainer } from "@/components/layout/page-container"
import { PageHeader } from "@/components/layout/page-header"
import {
  getCollection,
  getGetCollectionQueryKey,
  getGetCollectionQueryOptions,
} from "@/generated/endpoints/collections/collections"
import { useInfiniteCardResultsProps } from "@/lib/infinite-catalog-cards"
import { useInfiniteCollectionEntries } from "@/lib/infinite-inventory"
import { clerkAuth } from "@/lib/clerk-auth"
import { SessionChangedError, readInSession } from "@/lib/session"
import { pageHead } from "@/lib/site"
import { Button } from "@/components/ui/button"

export const Route = createFileRoute(
  "/_authenticated/trade-inboxes/$collectionId"
)({
  ssr: false,
  head: () => pageHead("Trade Inbox"),
  component: TradeInboxPage,
})

function TradeInboxPage() {
  const { collectionId } = Route.useParams()
  const { sessionId: clerkSessionId } = useAuth()
  const sessionId = clerkSessionId ?? null
  const currentSessionId = () => clerkAuth.currentSessionId()
  const isCurrentSession = () =>
    sessionId !== null && currentSessionId() === sessionId
  const collectionQuery = useQuery(
    getGetCollectionQueryOptions(collectionId, {
      query: {
        queryKey: [...getGetCollectionQueryKey(collectionId), sessionId],
        enabled: isCurrentSession(),
        gcTime: 0,
        retry: (failureCount, error) =>
          !(error instanceof SessionChangedError) && failureCount < 3,
        queryFn: ({ signal }) => {
          if (!sessionId) throw new SessionChangedError()
          return readInSession(sessionId, currentSessionId, () =>
            getCollection(collectionId, { signal })
          )
        },
      },
    })
  )
  const entriesQuery = useInfiniteCollectionEntries(
    collectionId,
    {},
    () => true,
    { id: sessionId, currentSessionId }
  )
  const results = useInfiniteCardResultsProps(entriesQuery)
  const collection =
    isCurrentSession() && collectionQuery.data?.status === 200
      ? collectionQuery.data.data.data
      : undefined
  const collectionError = isCurrentSession()
    ? collectionQuery.data?.status !== undefined &&
      collectionQuery.data.status !== 200
      ? collectionQuery.data.data.detail
      : collectionQuery.error?.detail
    : undefined

  return (
    <PageContainer>
      <PageHeader
        title={collection?.title ?? "Trade Inbox"}
        description="Cards received in this Trade. This page is read-only."
      />
      {(collectionQuery.isError || collectionError) && (
        <div role="alert" className="space-y-2 text-sm text-destructive">
          <p>{collectionError ?? "Could not load this Trade Inbox."}</p>
          <Button
            variant="outline"
            onClick={() => void collectionQuery.refetch()}
          >
            Retry
          </Button>
        </div>
      )}
      <InfiniteCardResults
        {...results}
        cards={isCurrentSession() ? results.cards : []}
        total={isCurrentSession() ? results.total : 0}
        isPending={!isCurrentSession() || results.isPending}
        isError={isCurrentSession() && results.isError}
        errorMessage={isCurrentSession() ? results.errorMessage : undefined}
        hasNextPage={isCurrentSession() && results.hasNextPage}
        isFetching={!isCurrentSession() || results.isFetching}
        emptyMessage="No cards are in this Trade Inbox."
        renderControl={(card) => {
          const quantity = entriesQuery.data?.quantities[card.id] ?? 0
          return (
            <p className="text-sm" aria-label={`${quantity} copies`}>
              Quantity: {quantity}
            </p>
          )
        }}
      />
      {isCurrentSession() && entriesQuery.isError && (
        <Button
          variant="outline"
          disabled={entriesQuery.isFetching}
          onClick={() =>
            void (entriesQuery.isFetchNextPageError
              ? entriesQuery.fetchNextPage()
              : entriesQuery.refetch())
          }
        >
          Retry
        </Button>
      )}
    </PageContainer>
  )
}
