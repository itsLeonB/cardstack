import type { FormEvent } from "react"
import { useEffect, useState, useSyncExternalStore } from "react"
import { useQuery, useQueryClient } from "@tanstack/react-query"
import { toast } from "sonner"
import { createFileRoute, Link, useRouter } from "@tanstack/react-router"

import { SignInLink } from "@/components/auth/sign-in-link"
import { NotFoundView } from "@/components/layout/not-found"
import { PageContainer } from "@/components/layout/page-container"
import { PageHeader } from "@/components/layout/page-header"
import { pageHead, NOINDEX_META } from "@/lib/site"
import { NETWORK_ERROR } from "@/lib/collections"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import type { TradeItemView, TradeView } from "@/generated/models"
import { clerkAuth } from "@/lib/clerk-auth"
import { tradeStatusLabel } from "@/lib/trade-status"
import { useIsGuest } from "@/lib/session"
import {
  getTrade,
  getGetTradeQueryKey,
  getGetTradeQueryOptions,
  useJoinTrade,
  useAddTradeItem,
  useRemoveTradeItem,
  useAcceptTrade,
  useConfirmTrade,
} from "@/generated/endpoints/trades/trades"
import { getListCollectionsQueryOptions } from "@/generated/endpoints/collections/collections"
import { getListCollectionEntriesQueryOptions } from "@/generated/endpoints/inventory/inventory"

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

function TradePage() {
  const { trade, sessionVersion } = Route.useLoaderData()
  const currentVersion = useSyncExternalStore(
    clerkAuth.subscribeSession,
    clerkAuth.currentSessionVersion,
    clerkAuth.currentSessionVersion
  )
  const router = useRouter()
  const queryClient = useQueryClient()
  const isGuest = useIsGuest()
  const tradeQuery = useQuery({
    ...getGetTradeQueryOptions(trade && trade !== "missing" ? trade.slug : ""),
    enabled:
      trade !== null && trade !== "missing" && trade.isParticipant === true,
    staleTime: Number.POSITIVE_INFINITY,
    refetchOnMount: false,
    refetchOnWindowFocus: false,
    refetchInterval: 10_000,
  })
  useEffect(() => {
    if (!trade || trade === "missing") return
    if (tradeQuery.data?.status === 404) {
      void router.invalidate()
      return
    }
    const polled =
      tradeQuery.data?.status === 200 ? tradeQuery.data.data.data : null
    if (!polled) return
    if (
      polled.revision !== trade.revision ||
      polled.status !== trade.status ||
      polled.accepted !== trade.accepted ||
      polled.otherAccepted !== trade.otherAccepted ||
      polled.confirmed !== trade.confirmed ||
      polled.otherConfirmed !== trade.otherConfirmed
    ) {
      void queryClient
        .invalidateQueries({ queryKey: getGetTradeQueryKey(trade.slug) })
        .then(() => router.invalidate())
    }
  }, [trade, tradeQuery.data, router, queryClient])

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
    if (clerkAuth.currentSessionVersion() === sessionVersion)
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
      <PageHeader title="Trade" description={tradeStatusLabel(trade.status)} />
      <TradeSide
        key={`${sessionVersion}:${trade.slug}:owner`}
        name={trade.ownerName}
        items={trade.ownerItems ?? []}
        slug={trade.slug}
        editable={negotiating && trade.isOwner}
        acceptLabel={acceptLabel(ownerAccepted, trade.isOwner, trade.ownerName)}
      />
      <TradeSide
        key={`${sessionVersion}:${trade.slug}:other`}
        name={trade.otherName ?? null}
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
      {trade.isOwner && <OwnerLink slug={trade.slug} />}
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
  const router = useRouter()
  const [collectionId, setCollectionId] = useState("")
  const [cardId, setCardId] = useState("")
  const [cardName, setCardName] = useState("")
  const [quantity, setQuantity] = useState("1")
  const [error, setError] = useState("")
  const collectionsQuery = useQuery({
    ...getListCollectionsQueryOptions(),
    enabled: editable,
  })
  const entriesQuery = useQuery({
    ...getListCollectionEntriesQueryOptions(collectionId, {
      name: cardName || undefined,
      limit: 100,
    }),
    enabled: editable && collectionId !== "",
  })
  const addMutation = useAddTradeItem()
  const removeMutation = useRemoveTradeItem()
  const collections =
    collectionsQuery.data?.status === 200
      ? (collectionsQuery.data.data.data ?? [])
      : []
  const entries =
    entriesQuery.data?.status === 200 ? (entriesQuery.data.data.data ?? []) : []

  function addItem(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    const count = Number(quantity)
    if (
      !collectionId ||
      !cardId ||
      !Number.isInteger(count) ||
      count < 1 ||
      count > 2147483647
    ) {
      setError(
        "Choose a Collection and Card, and enter a quantity of at least 1."
      )
      return
    }
    setError("")
    addMutation.mutate(
      {
        slug,
        data: { cardId, quantity: count, sourceCollectionId: collectionId },
      },
      {
        onSuccess: (response) => {
          if (response.status === 200) {
            setCardId("")
            setQuantity("1")
            void router.invalidate()
          } else {
            setError("Could not add this Card. Check the Trade and try again.")
          }
        },
        onError: () => setError(NETWORK_ERROR),
      }
    )
  }

  function removeItem(itemId: string) {
    setError("")
    removeMutation.mutate(
      { slug, itemId },
      {
        onSuccess: (response) => {
          if (response.status === 200) void router.invalidate()
          else
            setError(
              "Could not remove this Card. Check the Trade and try again."
            )
        },
        onError: () => setError(NETWORK_ERROR),
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
            {items.map((item) => (
              <li key={item.id} className="flex items-center gap-3">
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
                    disabled={removeMutation.isPending}
                    onClick={() => removeItem(item.id)}
                  >
                    Remove {item.card.name}
                  </Button>
                )}
              </li>
            ))}
          </ul>
        )}
        {error && (
          <p role="alert" className="text-sm text-destructive">
            {error}
          </p>
        )}
        {editable &&
          !collectionsQuery.isPending &&
          (collectionsQuery.isError ||
            collectionsQuery.data?.status !== 200) && (
            <div
              role="alert"
              className="flex flex-col items-start gap-2 text-sm text-destructive"
            >
              <p>Could not load your Collections.</p>
              <Button
                type="button"
                size="sm"
                variant="outline"
                onClick={() => void collectionsQuery.refetch()}
              >
                Retry Collections
              </Button>
            </div>
          )}
        {editable &&
          collectionsQuery.data?.status === 200 &&
          collections.length === 0 && (
            <p className="text-sm text-muted-foreground">
              Create a Collection before adding a Card.
            </p>
          )}
        {editable && (
          <form
            onSubmit={addItem}
            className="grid gap-3 border-t pt-4 sm:grid-cols-2"
          >
            <div className="grid gap-1">
              <label htmlFor={`collection-${slug}`}>Source Collection</label>
              <select
                id={`collection-${slug}`}
                value={collectionId}
                className="h-9 w-full rounded-3xl border border-transparent bg-input/50 px-3 text-sm focus-visible:border-ring focus-visible:ring-3 focus-visible:ring-ring/30"
                onChange={(event) => {
                  setCollectionId(event.target.value)
                  setCardId("")
                }}
                required
              >
                <option value="">Choose a Collection</option>
                {collections.map((collection) => (
                  <option key={collection.id} value={collection.id}>
                    {collection.title}
                  </option>
                ))}
              </select>
            </div>
            <div className="grid gap-1">
              <label htmlFor={`card-search-${slug}`}>Search Cards</label>
              <Input
                id={`card-search-${slug}`}
                value={cardName}
                onChange={(event) => {
                  setCardName(event.target.value)
                  setCardId("")
                }}
                placeholder="Type a Card name"
                disabled={!collectionId}
              />
            </div>
            <div className="grid gap-1">
              <label htmlFor={`card-${slug}`}>Card</label>
              <select
                id={`card-${slug}`}
                value={cardId}
                className="h-9 w-full rounded-3xl border border-transparent bg-input/50 px-3 text-sm focus-visible:border-ring focus-visible:ring-3 focus-visible:ring-ring/30 disabled:opacity-50"
                onChange={(event) => setCardId(event.target.value)}
                required
                disabled={!collectionId || entriesQuery.isPending}
              >
                <option value="">Choose a Card</option>
                {entries.map((entry) => (
                  <option key={entry.card.id} value={entry.card.id}>
                    {entry.card.name} — {entry.card.expansionSet.name} —{" "}
                    {entry.card.rarity.name}
                  </option>
                ))}
              </select>
            </div>
            <div className="grid gap-1">
              <label htmlFor={`quantity-${slug}`}>Quantity</label>
              <Input
                id={`quantity-${slug}`}
                type="number"
                min={1}
                max={2147483647}
                step={1}
                required
                value={quantity}
                onChange={(event) => setQuantity(event.target.value)}
              />
            </div>
            {!entriesQuery.isPending &&
              (entriesQuery.isError ||
                (entriesQuery.data !== undefined &&
                  entriesQuery.data.status !== 200)) &&
              collectionId !== "" && (
                <div
                  role="alert"
                  className="flex flex-col items-start gap-2 text-sm text-destructive sm:col-span-2"
                >
                  <p>Could not load Cards from this Collection.</p>
                  <Button
                    type="button"
                    size="sm"
                    variant="outline"
                    onClick={() => void entriesQuery.refetch()}
                  >
                    Retry Cards
                  </Button>
                </div>
              )}
            <Button
              type="submit"
              disabled={
                addMutation.isPending ||
                collectionsQuery.isPending ||
                !collections.length
              }
              className="sm:col-span-2 sm:justify-self-start"
            >
              {addMutation.isPending ? "Adding..." : "Add Card"}
            </Button>
          </form>
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
