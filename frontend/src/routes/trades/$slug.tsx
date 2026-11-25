import { useState } from "react"
import { toast } from "sonner"
import { createFileRoute, Link, useRouter } from "@tanstack/react-router"

import { TradeShare } from "@/components/share/trade-share"
import { useTradeSessionGate } from "@/components/trade/use-trade-session-gate"
import { Breadcrumbs } from "@/components/layout/breadcrumbs"
import { PageContainer } from "@/components/layout/page-container"
import { PageHeader } from "@/components/layout/page-header"
import { pageHead, NOINDEX_META } from "@/lib/site"
import { NETWORK_ERROR } from "@/lib/collections"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import type { TradeItemView, TradeView } from "@/generated/models"
import { clerkAuth } from "@/lib/clerk-auth"
import {
  loadTradeForSession,
  useTradePolling,
  useTradeItemSync,
} from "@/lib/trade-loader"
import { tradeStatusLabel } from "@/lib/trade-status"
import {
  getTrade,
  useJoinTrade,
  useRemoveTradeItem,
  useAcceptTrade,
  useConfirmTrade,
} from "@/generated/endpoints/trades/trades"

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
      {trade.isParticipant && negotiating && <AcceptTrade trade={trade} />}
      {trade.isParticipant && trade.status === "locked" && (
        <ConfirmTrade trade={trade} />
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
  const startSync = useTradeItemSync()
  const [error, setError] = useState("")
  const removeMutation = useRemoveTradeItem()

  function removeItem(itemId: string) {
    setError("")
    const sync = startSync()
    removeMutation.mutate(
      { slug, itemId },
      {
        onSuccess: (response) => {
          if (!sync.current()) return
          if (response.status === 200) void sync.refresh(slug, response)
          else
            setError(
              "Could not remove this Card. Check the Trade and try again."
            )
        },
        onError: () => {
          if (sync.current()) setError(NETWORK_ERROR)
        },
      }
    )
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
                    disabled={removeMutation.isPending}
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
  const router = useRouter()
  const joinMutation = useJoinTrade()
  const sessionVersion = clerkAuth.currentSessionVersion()
  // Keep the offer if another participant's seat makes the fresh read a 404.
  const [seatTaken, setSeatTaken] = useState(false)

  if (trade.isParticipant || trade.status !== "open") return null

  if (seatTaken)
    return (
      <p role="status" className="text-sm">
        This Trade is already full.
      </p>
    )

  return (
    <Button
      disabled={joinMutation.isPending}
      onClick={() =>
        joinMutation.mutate(
          { slug: trade.slug },
          {
            onSuccess: async (response) => {
              if (clerkAuth.currentSessionVersion() !== sessionVersion) return
              if (response.status === 200) {
                void router.invalidate()
                return
              }
              if (response.data.code === "trade_seat_taken") {
                setSeatTaken(true)
                try {
                  const current = await getTrade(trade.slug)
                  if (clerkAuth.currentSessionVersion() !== sessionVersion)
                    return
                  if (current.status === 200) {
                    void router.invalidate()
                  } else if (current.status !== 404) {
                    toast.error(
                      current.data.detail ?? "Could not load this Trade."
                    )
                  }
                } catch {
                  if (clerkAuth.currentSessionVersion() === sessionVersion)
                    toast.error(NETWORK_ERROR)
                }
                return
              }
              toast.error(response.data.detail ?? "Could not join this Trade.")
            },
            onError: () => {
              if (clerkAuth.currentSessionVersion() === sessionVersion)
                toast.error(NETWORK_ERROR)
            },
          }
        )
      }
    >
      {joinMutation.isPending ? "Joining..." : "Join this Trade"}
    </Button>
  )
}

/** The caller's one acceptance of the deal as it stands; the second
 * acceptance locks the Trade server-side. */
function AcceptTrade({ trade }: { trade: TradeView }) {
  const router = useRouter()
  const acceptMutation = useAcceptTrade()
  const sessionVersion = clerkAuth.currentSessionVersion()

  if (trade.accepted)
    return (
      <p role="status" className="text-sm">
        You accepted the deal as it stands.
      </p>
    )

  return (
    <div className="flex flex-col items-start gap-2">
      <Button
        disabled={acceptMutation.isPending}
        onClick={() =>
          acceptMutation.mutate(
            { slug: trade.slug, data: { revision: trade.revision } },
            {
              onSuccess: (response) => {
                if (clerkAuth.currentSessionVersion() !== sessionVersion) return
                if (response.status === 200) {
                  void router.invalidate()
                  return
                }
                toast.error(
                  response.data.detail ?? "Could not accept the Trade."
                )
                // The Trade may have moved on (locked, cancelled, or its
                // offer changed under this view): resync.
                void router.invalidate()
              },
              onError: () => {
                if (clerkAuth.currentSessionVersion() === sessionVersion)
                  toast.error(NETWORK_ERROR)
              },
            }
          )
        }
      >
        {acceptMutation.isPending ? "Accepting..." : "Accept the Trade"}
      </Button>
      <p className="text-sm text-muted-foreground">
        Accept the deal as it stands. The Trade locks when both sides accept.
      </p>
    </div>
  )
}

function ConfirmTrade({ trade }: { trade: TradeView }) {
  const router = useRouter()
  const mutation = useConfirmTrade()
  const sessionVersion = clerkAuth.currentSessionVersion()
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
      {mutation.data?.status && mutation.data.status !== 200 && (
        <p role="alert" className="text-sm text-destructive">
          {mutation.data.data.detail ?? "Could not confirm the handover."}
        </p>
      )}
      <Button
        disabled={mutation.isPending}
        onClick={() =>
          mutation.mutate(
            { slug: trade.slug },
            {
              onSuccess: (response) => {
                if (clerkAuth.currentSessionVersion() !== sessionVersion) return
                if (response.status !== 200)
                  toast.error(
                    response.data.detail ?? "Could not confirm the handover."
                  )
                void router.invalidate()
              },
              onError: () => {
                if (clerkAuth.currentSessionVersion() === sessionVersion)
                  toast.error(NETWORK_ERROR)
              },
            }
          )
        }
      >
        {mutation.isPending ? "Confirming..." : "Confirm handover"}
      </Button>
      <p className="text-sm">The exchange happens after both sides confirm.</p>
    </div>
  )
}
