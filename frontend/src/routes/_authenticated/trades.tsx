import { useRef, useState } from "react"
import { toast } from "sonner"
import { useQueryClient } from "@tanstack/react-query"
import { createFileRoute, Link, useNavigate } from "@tanstack/react-router"

import { PageContainer } from "@/components/layout/page-container"
import { PageHeader } from "@/components/layout/page-header"
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
  AlertDialogTrigger,
} from "@/components/ui/alert-dialog"
import { Button } from "@/components/ui/button"
import { Skeleton } from "@/components/ui/skeleton"
import {
  getListTradesQueryKey,
  useCancelTrade,
  useCreateTrade,
  useDiscardTrade,
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

function DiscardTradeDialog({ slug }: { slug: string }) {
  const queryClient = useQueryClient()
  const [open, setOpen] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const cancelRef = useRef<HTMLButtonElement>(null)
  const discard = useDiscardTrade()

  function confirm() {
    setError(null)
    discard.mutate(
      { slug },
      {
        onSuccess: async (response) => {
          if (response.status !== 204) {
            setError(response.data.detail ?? "Could not discard this Trade.")
            return
          }
          setOpen(false)
          await queryClient.invalidateQueries({
            queryKey: getListTradesQueryKey(),
          })
        },
        onError: () => setError(NETWORK_ERROR),
      }
    )
  }

  return (
    <AlertDialog
      open={open}
      onOpenChange={(nextOpen) => {
        setOpen(nextOpen)
        if (nextOpen) setError(null)
      }}
    >
      <AlertDialogTrigger
        render={
          <Button
            variant="outline"
            size="sm"
            className="min-h-11 text-destructive"
          />
        }
      >
        Discard
      </AlertDialogTrigger>
      <AlertDialogContent initialFocus={cancelRef}>
        <AlertDialogHeader>
          <AlertDialogTitle>Discard this Trade?</AlertDialogTitle>
          <AlertDialogDescription>
            This permanently deletes the Trade and its Cards. This cannot be
            undone.
          </AlertDialogDescription>
        </AlertDialogHeader>
        {error && (
          <p role="alert" className="text-sm text-destructive">
            {error}
          </p>
        )}
        <AlertDialogFooter>
          <AlertDialogCancel ref={cancelRef}>Cancel</AlertDialogCancel>
          <AlertDialogAction
            variant="destructive"
            disabled={discard.isPending}
            onClick={confirm}
          >
            {discard.isPending ? "Discarding..." : "Discard"}
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  )
}

function TradeRow({ trade }: { trade: TradeSummary }) {
  const queryClient = useQueryClient()
  const cancelTrade = useCancelTrade()
  const slug = trade.slug
  const discardable = slug !== undefined && trade.status === "open"
  const cancellable =
    slug !== undefined && CANCELLABLE_STATUSES.has(trade.status)
  const age = formatAge(trade.createdAt)
  const label = trade.otherName
    ? `With ${trade.otherName}`
    : "Waiting for someone to join"

  return (
    <li className="relative flex flex-wrap items-center gap-x-3 gap-y-1 rounded-lg border p-3 text-sm">
      {slug ? (
        <Link
          to="/trades/$slug"
          params={{ slug }}
          className="inline-flex min-h-11 min-w-11 items-center text-foreground underline after:absolute after:inset-0"
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
      {(discardable || cancellable) && (
        <div className="relative z-10 flex w-full justify-end border-t pt-2">
          {discardable ? (
            <DiscardTradeDialog slug={slug} />
          ) : (
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
        </div>
      )}
    </li>
  )
}
