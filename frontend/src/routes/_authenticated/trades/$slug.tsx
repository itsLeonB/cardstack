import { useState } from "react"
import { createFileRoute, useRouter } from "@tanstack/react-router"

import { NotFoundView } from "@/components/layout/not-found"
import { PageContainer } from "@/components/layout/page-container"
import { PageHeader } from "@/components/layout/page-header"
import { pageHead } from "@/lib/site"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import type { TradeItemView, TradeView } from "@/generated/models"
import {
  getGetTradeQueryKey,
  getGetTradeQueryOptions,
  useJoinTrade,
} from "@/generated/endpoints/trades/trades"

/** A Trade is readable by anyone signed in through its link; reading never claims the second seat, joining does. */
export const Route = createFileRoute("/_authenticated/trades/$slug")({
  // Read fresh on every visit: a cached view would keep offering a seat
  // someone took since.
  loader: async ({ context: { queryClient }, params }) => {
    const response = await queryClient.fetchQuery({
      ...getGetTradeQueryOptions(params.slug),
      staleTime: 0,
    })
    if (response.status === 404) {
      queryClient.removeQueries({ queryKey: getGetTradeQueryKey(params.slug) })
      return "missing" as const
    }
    return response.status === 200 ? response.data.data : null
  },
  head: () => pageHead("Trade"),
  component: TradePage,
})

// Trade pages inherit noindex from the _authenticated layout.
const STATUS_LABELS = {
  open: "Open",
  negotiating: "Negotiating",
  locked: "Locked",
  executed: "Executed",
  cancelled: "Cancelled",
} satisfies Record<string, string>

function TradePage() {
  const trade = Route.useLoaderData()
  const router = useRouter()

  if (trade === "missing") return <NotFoundView resource="Trade" />

  function retry() {
    void router.invalidate()
  }

  if (!trade)
    return (
      <PageContainer>
        <p role="alert" className="text-sm text-destructive">
          Could not load this Trade.
        </p>
        <div>
          <Button variant="outline" onClick={retry}>
            Retry
          </Button>
        </div>
      </PageContainer>
    )

  return (
    <PageContainer>
      <PageHeader
        title="Trade"
        // SAFETY: a status outside the map (a newer state) reads as itself.
        description={
          STATUS_LABELS[trade.status as keyof typeof STATUS_LABELS] ??
          trade.status
        }
      />
      <TradeSide name={trade.ownerName} items={trade.ownerItems ?? []} />
      <TradeSide
        name={trade.otherName ?? null}
        items={trade.otherItems ?? []}
      />
      {trade.isOwner && <OwnerLink slug={trade.slug} />}
      <JoinSeat trade={trade} />
    </PageContainer>
  )
}

function TradeSide({
  name,
  items,
}: {
  name: string | null
  items: TradeItemView[]
}) {
  return (
    <Card>
      <CardHeader>
        <CardTitle>
          <h2 className="text-lg">{name ?? "Waiting for someone to join"}</h2>
        </CardTitle>
      </CardHeader>
      <CardContent>
        {items.length === 0 ? (
          <p className="text-sm text-muted-foreground">No Cards offered yet.</p>
        ) : (
          <ul className="flex flex-col gap-1 text-sm">
            {items.map((item) => (
              <li key={`${item.card.id}-${item.collection}`}>
                {item.card.name} × {item.quantity} — from {item.collection}
              </li>
            ))}
          </ul>
        )}
      </CardContent>
    </Card>
  )
}

/** The owner's view of the link to send; the address bar carries it too. */
function OwnerLink({ slug }: { slug: string }) {
  const [copied, setCopied] = useState(false)
  const url = `${window.location.origin}/trades/${slug}`

  return (
    <div className="flex flex-col items-start gap-2">
      <p className="text-sm break-all">{url}</p>
      <Button
        variant="outline"
        onClick={() => {
          void navigator.clipboard.writeText(url)
          setCopied(true)
          setTimeout(() => setCopied(false), 2000)
        }}
      >
        {copied ? "Copied!" : "Copy link"}
      </Button>
    </div>
  )
}

function JoinSeat({ trade }: { trade: TradeView }) {
  const [errorMessage, setErrorMessage] = useState<string | null>(null)
  const router = useRouter()
  const joinMutation = useJoinTrade()

  if (trade.isParticipant || trade.status !== "open") return null

  return (
    <div className="flex flex-col items-start gap-3">
      {errorMessage && (
        <p role="alert" className="text-sm text-destructive">
          {errorMessage}
        </p>
      )}
      <Button
        disabled={joinMutation.isPending}
        onClick={() => {
          setErrorMessage(null)
          joinMutation.mutate(
            { slug: trade.slug },
            {
              onSuccess: (response) => {
                if (response.status === 200) {
                  void router.invalidate()
                  return
                }
                setErrorMessage(
                  response.data.code === "trade_seat_taken"
                    ? "This Trade is already full."
                    : (response.data.detail ?? "Could not join this Trade.")
                )
              },
              onError: () =>
                setErrorMessage(
                  "Could not reach the server. Please try again."
                ),
            }
          )
        }}
      >
        {joinMutation.isPending ? "Joining..." : "Join this Trade"}
      </Button>
    </div>
  )
}
