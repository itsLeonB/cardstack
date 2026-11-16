import { useEffect, useState } from "react"
import { useAuth } from "@clerk/react"
import { useQuery, useQueryClient } from "@tanstack/react-query"
import { createFileRoute, useNavigate } from "@tanstack/react-router"

import { InfiniteCardResults } from "@/components/catalog/infinite-card-results"
import { PageContainer } from "@/components/layout/page-container"
import { PageHeader } from "@/components/layout/page-header"
import {
  getCollection,
  getGetCollectionQueryKey,
  getGetCollectionQueryOptions,
  useListCollections,
} from "@/generated/endpoints/collections/collections"
import { useMoveCollectionEntries } from "@/generated/endpoints/inventory/inventory"
import { useInfiniteCardResultsProps } from "@/lib/infinite-catalog-cards"
import { useInfiniteCollectionEntries } from "@/lib/infinite-inventory"
import { clerkAuth } from "@/lib/clerk-auth"
import { SessionChangedError, readInSession } from "@/lib/session"
import { pageHead } from "@/lib/site"
import { Button } from "@/components/ui/button"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import {
  invalidateCollectionEntries,
  invalidateCollectionCounts,
} from "@/lib/collections"

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
  const queryClient = useQueryClient()
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

  const collectionsQuery = useListCollections({
    query: { enabled: isCurrentSession() },
  })
  const collections =
    collectionsQuery.data?.status === 200
      ? (collectionsQuery.data.data.data ?? []).filter(
          (c) => c.id !== collectionId && !c.binder
        )
      : []
  const move = useMoveCollectionEntries()
  const [filingCard, setFilingCard] = useState<string | null>(null)
  const [destinationId, setDestinationId] = useState<string>("")
  const [filing, setFiling] = useState(false)
  const [fileError, setFileError] = useState<string | null>(null)
  const [completed, setCompleted] = useState(false)
  const navigate = useNavigate()

  useEffect(() => {
    if (filingCard && !destinationId && collections.length > 0) {
      setDestinationId(collections[0].id)
    }
  }, [filingCard, destinationId, collections])

  async function fileCard() {
    if (!filingCard || !destinationId) return
    setFiling(true)
    setFileError(null)
    try {
      const response = await move.mutateAsync({
        id: collectionId,
        data: { cardIds: [filingCard], destinationId },
      })
      if (response.status === 200) {
        invalidateCollectionEntries(queryClient, collectionId)
        invalidateCollectionCounts(queryClient, collectionId)
        invalidateCollectionEntries(queryClient, destinationId)
        invalidateCollectionCounts(queryClient, destinationId)
        setFilingCard(null)
        setDestinationId("")
        const refreshed = await collectionQuery.refetch()
        if (refreshed.data?.status !== 200) {
          setCompleted(true)
          return
        }
        void entriesQuery.refetch()
        return
      }
      setFileError("Could not file this card.")
    } catch {
      setFileError("Could not file this card.")
    } finally {
      setFiling(false)
    }
  }

  if (completed) {
    return (
      <PageContainer>
        <PageHeader title="Trade Inbox filed" />
        <p className="text-sm">
          All cards have been filed. This Trade Inbox is now empty.
        </p>
        <Button onClick={() => void navigate({ to: "/collections" })}>
          Back to Collections
        </Button>
      </PageContainer>
    )
  }

  return (
    <PageContainer>
      <PageHeader
        title={collection?.title ?? "Trade Inbox"}
        description="Cards received in this Trade. File them into one of your Collections."
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
            <div className="flex items-center gap-2">
              <p className="text-sm" aria-label={`${quantity} copies`}>
                Quantity: {quantity}
              </p>
              <Button
                size="sm"
                variant="outline"
                onClick={() => {
                  setFilingCard(card.id)
                  setDestinationId(collections[0]?.id ?? "")
                  setFileError(null)
                }}
              >
                File
              </Button>
            </div>
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

      <Dialog
        open={filingCard !== null}
        onOpenChange={(open) => {
          if (!open) setFilingCard(null)
        }}
      >
        <DialogContent>
          <DialogHeader>
            <DialogTitle>File card</DialogTitle>
            <DialogDescription>
              Choose the Collection to file this card into.
            </DialogDescription>
          </DialogHeader>
          <select
            className="w-full rounded border px-2 py-1"
            value={destinationId}
            onChange={(e) => setDestinationId(e.target.value)}
          >
            {collections.map((c) => (
              <option key={c.id} value={c.id}>
                {c.title}
              </option>
            ))}
          </select>
          {fileError && (
            <p role="alert" className="text-sm text-destructive">
              {fileError}
            </p>
          )}
          <Button
            disabled={filing || !destinationId}
            onClick={() => void fileCard()}
          >
            {filing ? "Filing…" : "File"}
          </Button>
        </DialogContent>
      </Dialog>
    </PageContainer>
  )
}
