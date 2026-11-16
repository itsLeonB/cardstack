import { toast } from "sonner"
import { useQueryClient } from "@tanstack/react-query"
import { createFileRoute, Link, useNavigate } from "@tanstack/react-router"

import { pageHead } from "@/lib/site"
import { NETWORK_ERROR } from "@/lib/collections"
import { formatAge } from "@/lib/date"
import { CANCELLABLE_STATUSES, tradeStatusLabel } from "@/lib/trade-status"
import {
  getListTradesQueryKey,
  useCancelTrade,
  useCreateTrade,
  useListTrades,
} from "@/generated/endpoints/trades/trades"
import type { TradeSummary } from "@/generated/models"
import { PageContainer } from "@/components/layout/page-container"
import { PageHeader } from "@/components/layout/page-header"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { useSession, useSignOut } from "@/lib/session"

/** Settings-style page, reached from the user menu. Navigation lives in the shell, not here. */
export const Route = createFileRoute("/_authenticated/account")({
  head: () => pageHead("Account"),
  component: AccountPage,
})

function AccountPage() {
  const { user } = useSession()
  const { signOut, isPending } = useSignOut()
  const navigate = useNavigate()
  const createTrade = useCreateTrade()

  return (
    <PageContainer>
      <PageHeader
        title="Account"
        description="Your sign-in details and session."
      />

      <Card>
        <CardHeader>
          <CardTitle>
            <h2 className="text-lg">Profile</h2>
          </CardTitle>
        </CardHeader>
        <CardContent>
          <dl className="flex flex-col gap-1 text-sm">
            {user?.name && (
              <>
                <dt className="font-medium">Name</dt>
                <dd className="break-words">{user.name}</dd>
              </>
            )}
            <dt className="font-medium">Email</dt>
            <dd className="break-all">{user?.email}</dd>
          </dl>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>
            <h2 className="text-lg">Trades</h2>
          </CardTitle>
        </CardHeader>
        <CardContent className="flex flex-col items-start gap-3">
          <p className="text-sm">
            Start a Trade to swap Cards with someone, and share its link.
          </p>
          <Button
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
          <TradesList />
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>
            <h2 className="text-lg">Session</h2>
          </CardTitle>
        </CardHeader>
        <CardContent className="flex flex-col items-start gap-3">
          <p className="text-sm">Sign out of Cardstack on this device.</p>
          <Button
            variant="outline"
            disabled={isPending}
            onClick={() => void signOut()}
          >
            {isPending ? "Logging out..." : "Log out"}
          </Button>
        </CardContent>
      </Card>
    </PageContainer>
  )
}

/** The caller's Trades, newest first, with state, age and cancel for live ones. */
function TradesList() {
  const tradesQuery = useListTrades()

  if (tradesQuery.isPending) return null
  const trades =
    tradesQuery.data?.status === 200 ? (tradesQuery.data.data.data ?? []) : null

  return (
    <div className="flex w-full flex-col gap-2 border-t pt-3">
      <h3 className="text-sm font-medium">Your Trades</h3>
      {trades === null ? (
        <div
          role="alert"
          className="flex flex-col items-start gap-2 text-sm text-destructive"
        >
          <p>Could not load your Trades.</p>
          <Button
            type="button"
            size="sm"
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
    </div>
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
          className="text-foreground underline"
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
          size="sm"
          variant="outline"
          disabled={cancelTrade.isPending}
          onClick={() => {
            if (!slug) return
            cancelTrade.mutate(
              { slug },
              {
                onSuccess: (response) => {
                  // The row's state may have moved on (executed refuses with
                  // trade_executed): refresh either way.
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
