import { useState, useSyncExternalStore } from "react"
import { toast } from "sonner"
import { createFileRoute, useRouter } from "@tanstack/react-router"

import { SignInLink } from "@/components/auth/sign-in-link"
import { NotFoundView } from "@/components/layout/not-found"
import { PageContainer } from "@/components/layout/page-container"
import { PageHeader } from "@/components/layout/page-header"
import { pageHead, NOINDEX_META } from "@/lib/site"
import { NETWORK_ERROR } from "@/lib/collections"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import type { TradeItemView, TradeView } from "@/generated/models"
import { clerkAuth } from "@/lib/clerk-auth"
import { useIsGuest } from "@/lib/session"
import {
  getGetTradeQueryKey,
  getGetTradeQueryOptions,
  useJoinTrade,
} from "@/generated/endpoints/trades/trades"

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
  loader: async ({ context: { queryClient }, params }) => {
    const read = () =>
      queryClient.fetchQuery({
        ...getGetTradeQueryOptions(params.slug),
        staleTime: 0,
      })
    // Check both attempts: even the retry can cross another session change.
    for (let attempt = 0; attempt < 2; attempt += 1) {
      const sessionVersion = clerkAuth.currentSessionVersion()
      const response = await read()
      if (clerkAuth.currentSessionVersion() !== sessionVersion) {
        queryClient.removeQueries({
          queryKey: getGetTradeQueryKey(params.slug),
        })
        continue
      }
      if (response.status === 404) {
        queryClient.removeQueries({
          queryKey: getGetTradeQueryKey(params.slug),
        })
        return { sessionVersion, trade: "missing" as const }
      }
      return {
        sessionVersion,
        trade: response.status === 200 ? response.data.data : null,
      }
    }
    return { sessionVersion: clerkAuth.currentSessionVersion(), trade: null }
  },
  head: ({ params }) => ({
    meta: [...pageHead(params.slug).meta, NOINDEX_META],
  }),
  component: TradePage,
})

const STATUS_LABELS = {
  open: "Open",
  negotiating: "Negotiating",
  locked: "Locked",
  executed: "Executed",
  cancelled: "Cancelled",
} satisfies Record<string, string>

function TradePage() {
  const { trade, sessionVersion } = Route.useLoaderData()
  const currentVersion = useSyncExternalStore(
    clerkAuth.subscribeSession,
    clerkAuth.currentSessionVersion,
    clerkAuth.currentSessionVersion
  )
  const router = useRouter()
  const isGuest = useIsGuest()

  if (sessionVersion !== currentVersion)
    return (
      <PageContainer>
        <PageHeader title="Trade" />
        <p role="status">Loading Trade...</p>
      </PageContainer>
    )

  if (trade === "missing") {
    const { pathname, searchStr } = router.state.location
    return (
      <NotFoundView resource="Trade">
        {/* A Guest's read is the same 404 as an unknown slug; the link asks
            for an account without confirming the Trade exists. */}
        {isGuest && (
          <p className="text-sm">
            <SignInLink redirect={pathname + searchStr}>Sign in</SignInLink> to
            see this Trade.
          </p>
        )}
      </NotFoundView>
    )
  }

  function retry() {
    void router.invalidate()
  }

  if (!trade)
    return (
      <PageContainer>
        <PageHeader title="Trade" />
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
      <JoinSeat key={`${sessionVersion}:${trade.slug}`} trade={trade} />
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
            {items.map((item, index) => (
              <li key={`${item.card.id}-${index}`}>
                {item.card.name} × {item.quantity}
                {item.collection
                  ? ` — from ${item.collection}`
                  : " — from a deleted Collection"}
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
  const url = `${window.location.origin}/trades/${slug}`

  return (
    <div className="flex flex-col items-start gap-2">
      <p className="text-sm break-all">{url}</p>
    </div>
  )
}

function JoinSeat({ trade }: { trade: TradeView }) {
  const router = useRouter()
  const joinMutation = useJoinTrade()
  // Set when the seat goes between the read and the join: a refetch would
  // answer this caller the same 404 as an unknown slug, so the page keeps
  // the offer it already showed and says the Trade is full instead.
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
            onSuccess: (response) => {
              if (response.status === 200) {
                void router.invalidate()
                return
              }
              if (response.data.code === "trade_seat_taken") {
                setSeatTaken(true)
                return
              }
              toast.error(response.data.detail ?? "Could not join this Trade.")
            },
            onError: () => toast.error(NETWORK_ERROR),
          }
        )
      }
    >
      {joinMutation.isPending ? "Joining..." : "Join this Trade"}
    </Button>
  )
}
