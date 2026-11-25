import { useState } from "react"
import { createFileRoute, Link } from "@tanstack/react-router"

import { TradeShare } from "@/components/share/trade-share"
import { useTradeSessionGate } from "@/components/trade/use-trade-session-gate"
import { Breadcrumbs } from "@/components/layout/breadcrumbs"
import { PageContainer } from "@/components/layout/page-container"
import { PageHeader } from "@/components/layout/page-header"
import { pageHead, NOINDEX_META } from "@/lib/site"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import type { TradeItemView, TradeView } from "@/generated/models"
import { clerkAuth } from "@/lib/clerk-auth"
import {
  loadTradeForSession,
  useTradePolling,
  useRemoveTradeItemAction,
  useJoinTradeAction,
  useAcceptTradeAction,
  useConfirmTradeAction,
} from "@/lib/trade-loader"
import { tradeStatusLabel } from "@/lib/trade-status"

/** The Trade's name follows the counterparty once one joins; an open seat has no name yet. */
function tradeTitle(trade: TradeView): string {
  if (trade.status === "open") return "New trade"
  const counterparty = trade.isOwner ? trade.otherName : trade.ownerName
  return counterparty ?? "Trade"
}

/**
 * Outside `_authenticated/`: the API answers a Guest's read with the same
 * not-found as an unknown slug, so the link leaks nothing and no guard
 * redirects before that answer can render.
 */
export const Route = createFileRoute("/trades/$slug")({
  // The loader's request carries the Clerk token, which exists only in the
  // browser (same reason as `_authenticated.tsx`).
  ssr: false,
  // Read fresh on every visit: a cached view would keep offering a seat
  // someone took since.
  loader: ({ context: { queryClient }, params }) =>
    loadTradeForSession(queryClient, params.slug),
  head: ({ loaderData }) => {
    const trade = loaderData?.trade
    // The body hides stale session data; the document title needs the same gate.
    const staleSession =
      loaderData?.sessionVersion !== clerkAuth.currentSessionVersion()
    return {
      meta: [
        ...pageHead(
          staleSession || !trade || trade === "missing"
            ? "Trade"
            : tradeTitle(trade)
        ).meta,
        NOINDEX_META,
      ],
    }
  },
  component: TradePage,
})

function TradePage() {
  const { trade, sessionVersion } = Route.useLoaderData()
  useTradePolling({ trade, sessionVersion })

  const gate = useTradeSessionGate({ trade, sessionVersion, title: "Trade" })
  if (gate || !trade || trade === "missing") return gate

  const negotiating = trade.isParticipant && trade.status === "negotiating"
  const ownerAccepted = trade.isOwner ? trade.accepted : trade.otherAccepted
  const seatAccepted = trade.isOwner ? trade.otherAccepted : trade.accepted
  function acceptLabel(accepted: boolean, own: boolean, name: string) {
    if (accepted) return "Accepted the deal."
    if (!negotiating) return null
    return own ? "Waiting on you to accept." : `Waiting on ${name} to accept.`
  }
  return (
    <PageContainer>
      <Breadcrumbs
        crumbs={[
          {
            label: "Trades",
            link: trade.isParticipant ? { to: "/trades" } : undefined,
          },
          { label: "Trade" },
        ]}
      />
      <PageHeader
        title={tradeTitle(trade)}
        description={tradeStatusLabel(trade.status)}
      />
      <TradeSide
        key={`${sessionVersion}:${trade.slug}:owner`}
        name={trade.isOwner ? "You" : trade.ownerName}
        items={trade.ownerItems ?? []}
        slug={trade.slug}
        editable={negotiating && trade.isOwner}
        acceptLabel={acceptLabel(ownerAccepted, trade.isOwner, trade.ownerName)}
      />
      <TradeSide
        key={`${sessionVersion}:${trade.slug}:other`}
        name={
          trade.isParticipant && !trade.isOwner
            ? "You"
            : (trade.otherName ?? null)
        }
        items={trade.otherItems ?? []}
        slug={trade.slug}
        editable={trade.isParticipant && negotiating && !trade.isOwner}
        acceptLabel={acceptLabel(
          seatAccepted,
          !trade.isOwner,
          trade.otherName ?? "the other participant"
        )}
      />
      {trade.isParticipant && negotiating && (
        <AcceptTrade
          key={`${sessionVersion}:${trade.slug}:accept`}
          trade={trade}
        />
      )}
      {trade.isParticipant && trade.status === "locked" && (
        <ConfirmTrade
          key={`${sessionVersion}:${trade.slug}:confirm`}
          trade={trade}
        />
      )}
      {trade.status === "executed" && (
        <>
          <p role="status">The exchange is complete.</p>
          {trade.inboxId && (
            <Link
              to="/trade-inboxes/$collectionId"
              params={{ collectionId: trade.inboxId }}
              className="text-foreground underline"
            >
              View your Trade Inbox
            </Link>
          )}
        </>
      )}
      {trade.isOwner && trade.status === "open" && (
        <TradeShareLink slug={trade.slug} />
      )}
      <JoinSeat key={`${sessionVersion}:${trade.slug}`} trade={trade} />
    </PageContainer>
  )
}

function TradeSide({
  name,
  items,
  slug,
  editable,
  acceptLabel,
}: {
  name: string | null
  items: TradeItemView[]
  slug: string
  editable: boolean
  acceptLabel: string | null
}) {
  const [error, setError] = useState("")
  const action = useRemoveTradeItemAction(slug, setError)

  function removeItem(itemId: string) {
    setError("")
    action.remove(itemId)
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle>
          <h2 className="text-lg">{name ?? "Waiting for someone to join"}</h2>
        </CardTitle>
        {acceptLabel && (
          <p className="text-sm text-muted-foreground">{acceptLabel}</p>
        )}
      </CardHeader>
      <CardContent className="flex flex-col gap-4">
        {items.length === 0 ? (
          <p className="text-sm text-muted-foreground">No Cards offered yet.</p>
        ) : (
          <ul className="flex flex-col gap-3 text-sm">
            {items.map((item, index) => (
              <li
                key={item.id}
                className="flex items-center gap-3 rounded-lg border p-3"
              >
                <img
                  src={item.card.imageUrl}
                  alt=""
                  className="h-16 w-12 shrink-0 rounded object-contain"
                  loading="lazy"
                />
                <div className="min-w-0 flex-1">
                  <p className="font-medium">
                    {item.card.name} × {item.quantity}
                  </p>
                  <p className="text-muted-foreground">
                    {item.card.expansionSet.name} · {item.card.rarity.name}
                  </p>
                  <p className="text-muted-foreground">
                    {item.collection
                      ? `From ${item.collection}`
                      : "From a deleted Collection"}
                  </p>
                  {item.short && (
                    <p className="mt-1">
                      <Badge variant="destructive">
                        <span className="sr-only">Short: </span>
                        {item.available === 0
                          ? "none"
                          : `only ${item.available}`}{" "}
                        of {item.quantity} available
                      </Badge>
                    </p>
                  )}
                </div>
                {editable && (
                  <Button
                    type="button"
                    size="sm"
                    variant="outline"
                    className="min-h-11"
                    aria-label={`Remove ${item.card.name} from ${item.collection ?? "a deleted Collection"}, item ${index + 1}`}
                    disabled={action.pending}
                    onClick={() => removeItem(item.id)}
                  >
                    Remove
                  </Button>
                )}
              </li>
            ))}
          </ul>
        )}
        {editable && (
          <Button
            variant="outline"
            size="sm"
            render={<Link to="/trades/$slug/add-item" params={{ slug }} />}
          >
            Add Card
          </Button>
        )}
        {error && (
          <p role="alert" className="text-sm text-destructive">
            {error}
          </p>
        )}
      </CardContent>
    </Card>
  )
}

/** The owner's view of the link to send; the address bar carries it too. */
function TradeShareLink({ slug }: { slug: string }) {
  return <TradeShare link={`${window.location.origin}/trades/${slug}`} />
}

function JoinSeat({ trade }: { trade: TradeView }) {
  const action = useJoinTradeAction(trade.slug)
  if (trade.isParticipant || trade.status !== "open") return null
  if (action.seatTaken)
    return (
      <p role="status" className="text-sm">
        This Trade is already full.
      </p>
    )
  return (
    <Button disabled={action.pending} onClick={action.join}>
      {action.pending ? "Joining..." : "Join this Trade"}
    </Button>
  )
}

/** The caller's one acceptance of the deal as it stands; the second
 * acceptance locks the Trade server-side. */
function AcceptTrade({ trade }: { trade: TradeView }) {
  const action = useAcceptTradeAction(trade)
  if (trade.accepted)
    return (
      <p role="status" className="text-sm">
        You accepted the deal as it stands.
      </p>
    )
  return (
    <div className="flex flex-col items-start gap-2">
      <Button disabled={action.pending} onClick={action.accept}>
        {action.pending ? "Accepting..." : "Accept the Trade"}
      </Button>
      <p className="text-sm text-muted-foreground">
        Accept the deal as it stands. The Trade locks when both sides accept.
      </p>
    </div>
  )
}

function ConfirmTrade({ trade }: { trade: TradeView }) {
  const action = useConfirmTradeAction(trade.slug)
  if (trade.confirmed)
    return (
      <p role="status">
        You confirmed the handover. Waiting on the other side.
      </p>
    )
  return (
    <div className="flex flex-col items-start gap-2">
      {trade.otherConfirmed && (
        <p role="status">
          The other side confirmed the handover. Confirm to complete the
          exchange.
        </p>
      )}
      {action.error && (
        <p role="alert" className="text-sm text-destructive">
          {action.error}
        </p>
      )}
      <Button disabled={action.pending} onClick={action.confirm}>
        {action.pending ? "Confirming..." : "Confirm handover"}
      </Button>
      <p className="text-sm">The exchange happens after both sides confirm.</p>
    </div>
  )
}
