import { useState } from "react"
import { useQuery } from "@tanstack/react-query"
import { createFileRoute } from "@tanstack/react-router"

import { CardTile } from "@/components/catalog/card-tile"
import { PageContainer } from "@/components/layout/page-container"
import { PageHeader } from "@/components/layout/page-header"
import { getGetCollectionQueryOptions } from "@/generated/endpoints/collections/collections"
import { getListCollectionEntriesQueryOptions } from "@/generated/endpoints/inventory/inventory"
import { pageHead } from "@/lib/site"
import { Button } from "@/components/ui/button"

const PAGE_SIZE = 24

export const Route = createFileRoute(
  "/_authenticated/trade-inboxes/$collectionId"
)({
  ssr: false,
  loader: async ({ context: { queryClient }, params }) => {
    const response = await queryClient.ensureQueryData(
      getGetCollectionQueryOptions(params.collectionId)
    )
    if (response.status === 404) return null
    return response.status === 200 ? response.data.data.title : undefined
  },
  head: ({ loaderData }) => pageHead(loaderData ?? "Trade Inbox"),
  component: TradeInboxPage,
})

function TradeInboxPage() {
  const { collectionId } = Route.useParams()
  const [page, setPage] = useState(1)
  const collectionQuery = useQuery(getGetCollectionQueryOptions(collectionId))
  const entriesQuery = useQuery(
    getListCollectionEntriesQueryOptions(collectionId, {
      page,
      limit: PAGE_SIZE,
    })
  )
  const collection =
    collectionQuery.data?.status === 200
      ? collectionQuery.data.data.data
      : undefined
  const entries =
    entriesQuery.data?.status === 200 ? (entriesQuery.data.data.data ?? []) : []
  const meta =
    entriesQuery.data?.status === 200 ? entriesQuery.data.data.meta : undefined
  const error =
    collectionQuery.data?.status !== undefined &&
    collectionQuery.data.status !== 200
      ? collectionQuery.data.data.detail
      : entriesQuery.data?.status !== undefined &&
          entriesQuery.data.status !== 200
        ? entriesQuery.data.data.detail
        : null

  return (
    <PageContainer>
      <PageHeader
        title={collection?.title ?? "Trade Inbox"}
        description="Cards received in this Trade. This page is read-only."
      />
      {(collectionQuery.isError || entriesQuery.isError || error) && (
        <div role="alert" className="space-y-2 text-sm text-destructive">
          <p>{error ?? "Could not load this Trade Inbox."}</p>
          <Button
            variant="outline"
            onClick={() => {
              void collectionQuery.refetch()
              void entriesQuery.refetch()
            }}
          >
            Retry
          </Button>
        </div>
      )}
      {entries.length > 0 ? (
        <div className="grid grid-cols-2 gap-3 sm:grid-cols-3 lg:grid-cols-4">
          {entries.map(({ card, quantity }) => (
            <CardTile
              key={card.id}
              card={card}
              control={
                <p className="text-sm" aria-label={`${quantity} copies`}>
                  Quantity: {quantity}
                </p>
              }
            />
          ))}
        </div>
      ) : (
        !entriesQuery.isPending && <p>No cards are in this Trade Inbox.</p>
      )}
      {meta && meta.total > page * PAGE_SIZE && (
        <Button
          variant="outline"
          disabled={entriesQuery.isFetching}
          onClick={() => setPage((current) => current + 1)}
        >
          {entriesQuery.isFetching ? "Loading..." : "Load more cards"}
        </Button>
      )}
    </PageContainer>
  )
}
