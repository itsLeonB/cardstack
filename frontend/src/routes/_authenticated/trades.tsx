import { toast } from "sonner"
import { useQueryClient } from "@tanstack/react-query"
import { createFileRoute, Link, useNavigate } from "@tanstack/react-router"

import { PageContainer } from "@/components/layout/page-container"
import { PageHeader } from "@/components/layout/page-header"
import { Button } from "@/components/ui/button"
import { Skeleton } from "@/components/ui/skeleton"
import {
  getListTradesQueryKey,
  useCancelTrade,
  useCreateTrade,
  useListTrades,
} from "@/generated/endpoints/trades/trades"
import type { TradeSummary } from "@/generated/models"
import { NETWORK_ERROR } from "@/lib/collections"
import { formatAge } from "@/lib/date"
import { pageHead } from "@/lib/site"
import { CANCELLABLE_STATUSES, tradeStatusLabel } from "@/lib/trade-status"

export const Route = createFileRoute("/_authenticated/trades")({
  head: () => pageHead("Trades"),
  component: TradesPage,
})

function TradesPage() {
  const navigate = useNavigate()
  const createTrade = useCreateTrade()

  return (
    <PageContainer>
      <PageHeader
        title="Trades"
        description="Start a Trade to swap Cards with someone, and share its link."
        actions={
          <Button
            className="min-h-11"
            disabled={createTrade.isPending}
            onClick={() =>
              createTrade.mutate(undefined, {
                onSuccess: (response) => {
                  if (response.status === 201) {
                    void navigate({
                      to: "/trades/$slug",
                      params: { slug: response.data.data.slug },
                    })
                    return
                  }
                  toast.error(
                    response.data.detail ?? "Could not start this Trade."
                  )
                },
                onError: () => toast.error(NETWORK_ERROR),
              })
            }
          >
            {createTrade.isPending ? "Starting..." : "Start a Trade"}
          </Button>
        }
      />
      <TradesList />
    </PageContainer>
  )
}

function TradesList() {
  const tradesQuery = useListTrades()

  if (tradesQuery.isPending) {
    return (
      <div role="status" className="flex flex-col gap-2">
        <span>Loading your Trades...</span>
        <Skeleton className="h-10 w-full" />
        <Skeleton className="h-10 w-full" />
        <Skeleton className="h-10 w-full" />
      </div>
    )
  }

  const trades =
    tradesQuery.data?.status === 200 ? (tradesQuery.data.data.data ?? []) : null

  return (
    <section className="flex flex-col gap-2">
      <h2 className="text-sm font-medium">Your Trades</h2>
      {trades === null ? (
        <div
          role="alert"
          className="flex flex-col items-start gap-2 text-sm text-destructive"
        >
          <p>Could not load your Trades.</p>
          <Button
            type="button"
            className="min-h-11"
            variant="outline"
            onClick={() => void tradesQuery.refetch()}
          >
            Retry
          </Button>
        </div>
      ) : trades.length === 0 ? (
        <p className="text-sm text-muted-foreground">No Trades yet.</p>
      ) : (
        <ul className="flex flex-col gap-2">
          {trades.map((trade) => (
            <TradeRow key={trade.id} trade={trade} />
          ))}
        </ul>
      )}
    </section>
  )
}

function TradeRow({ trade }: { trade: TradeSummary }) {
  const queryClient = useQueryClient()
  const cancelTrade = useCancelTrade()
  const slug = trade.slug
  const cancellable =
    slug !== undefined && CANCELLABLE_STATUSES.has(trade.status)
  const age = formatAge(trade.createdAt)
  const label = trade.otherName ?? "Waiting for someone to join"

  return (
    <li className="flex flex-wrap items-center gap-x-3 gap-y-1 text-sm">
      {slug ? (
        <Link
          to="/trades/$slug"
          params={{ slug }}
          className="inline-flex min-h-11 min-w-11 items-center text-foreground underline"
        >
          {label}
        </Link>
      ) : (
        <span>{label}</span>
      )}
      <span className="text-muted-foreground">
        {tradeStatusLabel(trade.status)}
        {age ? ` · started ${age}` : ""}
      </span>
      {cancellable && (
        <Button
          type="button"
          className="min-h-11"
          size="sm"
          variant="outline"
          disabled={cancelTrade.isPending}
          onClick={() => {
            if (!slug) return
            cancelTrade.mutate(
              { slug },
              {
                onSuccess: (response) => {
                  void queryClient.invalidateQueries({
                    queryKey: getListTradesQueryKey(),
                  })
                  if (response.status === 204) return
                  toast.error(
                    response.data.detail ?? "Could not cancel this Trade."
                  )
                },
                onError: () => toast.error(NETWORK_ERROR),
              }
            )
          }}
        >
          {cancelTrade.isPending ? "Cancelling..." : "Cancel"}
        </Button>
      )}
    </li>
  )
}
