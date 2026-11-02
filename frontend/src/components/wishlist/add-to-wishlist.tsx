import { useEffect, useState } from "react"
import { useQueryClient } from "@tanstack/react-query"
import { RiAddLine, RiSubtractLine } from "@remixicon/react"
import { SignInLink } from "@/components/auth/sign-in-link"
import { Button } from "@/components/ui/button"
import { Skeleton } from "@/components/ui/skeleton"
import { useSetWishlistItem } from "@/generated/endpoints/wishlist/wishlist"
import type { CardSummary } from "@/generated/models"
import { NETWORK_ERROR } from "@/lib/collections"
import { useSession } from "@/lib/session"
import { invalidateWishlist, useWishlistWanted } from "@/lib/wishlist"

const MAX_WANTED = 2147483647

/**
 * Wanted quantity stepper and the action that saves it. The API replaces the
 * wanted quantity of a Card already on the Wishlist, so this never adds a
 * second item, and it shows the quantity already wanted.
 */
export function AddToWishlist({
  cardId,
  cardName,
  wanted,
  onAnnounce,
}: {
  cardId: string
  cardName: string
  /** The quantity the Wishlist already wants, absent when it has no such Card. */
  wanted?: number
  onAnnounce: (message: string) => void
}) {
  const queryClient = useQueryClient()
  const save = useSetWishlistItem()
  const [quantity, setQuantity] = useState(wanted ?? 1)
  const [error, setError] = useState("")

  // An answer from the server wins over a quantity picked before it arrived.
  useEffect(() => {
    setQuantity(wanted ?? 1)
  }, [wanted])

  function addItem() {
    if (save.isPending) return
    setError("")
    save.mutate(
      { cardId, data: { quantity } },
      {
        onSuccess: (response) => {
          if (response.status !== 204) {
            setError(
              response.data.detail ??
                "Could not add this Card to your Wishlist."
            )
            return
          }
          onAnnounce(`Want ${quantity} of ${cardName}.`)
          invalidateWishlist(queryClient)
        },
        onError: () => setError(NETWORK_ERROR),
      }
    )
  }

  const onWishlist = wanted !== undefined

  return (
    <div className="flex flex-col gap-1.5">
      {onWishlist && <p className="text-xs font-medium">Wanted ×{wanted}</p>}
      <div className="flex flex-wrap items-center gap-1">
        <Button
          type="button"
          variant="outline"
          size="icon-sm"
          aria-label={`Decrease wanted quantity of ${cardName}`}
          disabled={quantity <= 1 || save.isPending}
          onClick={() => setQuantity(quantity - 1)}
        >
          <RiSubtractLine />
        </Button>
        <span className="min-w-6 text-center text-sm">
          <span className="sr-only">Wanted quantity </span>
          {quantity}
        </span>
        <Button
          type="button"
          variant="outline"
          size="icon-sm"
          aria-label={`Increase wanted quantity of ${cardName}`}
          disabled={quantity >= MAX_WANTED || save.isPending}
          onClick={() => setQuantity(quantity + 1)}
        >
          <RiAddLine />
        </Button>
        <Button
          type="button"
          variant="outline"
          size="sm"
          aria-label={
            onWishlist
              ? `Update the wanted quantity of ${cardName}`
              : `Add ${cardName} to your Wishlist`
          }
          disabled={save.isPending}
          onClick={addItem}
        >
          {onWishlist ? "Update" : "Add to Wishlist"}
        </Button>
      </div>
      {error && (
        <p role="alert" className="text-xs text-destructive">
          {error}
        </p>
      )}
    </div>
  )
}

/**
 * The Wishlist action for the one Card a catalog page shows, with its wanted
 * quantity looked up first so an add never replaces one blindly. A Guest gets
 * the sign-in prompt instead.
 */
export function CardWishlistAction({
  card,
  loginRedirect,
}: {
  card: CardSummary
  /** Path (with search) the sign-in prompt returns to. */
  loginRedirect?: string
}) {
  const { isAuthenticated, isLoading } = useSession()
  const lookup = useWishlistWanted([card.id], isAuthenticated)
  const [announcement, setAnnouncement] = useState("")

  let body
  if (
    isLoading ||
    (isAuthenticated && !lookup.failed && !lookup.loaded[card.id])
  ) {
    body = (
      <Skeleton className="h-10 w-full" aria-label="Loading your Wishlist" />
    )
  } else if (!isAuthenticated) {
    body = (
      <p className="text-sm text-muted-foreground">
        <SignInLink redirect={loginRedirect}>Sign in</SignInLink> to add this
        Card to your Wishlist.
      </p>
    )
  } else if (lookup.failed) {
    body = (
      <p role="alert" className="text-sm text-destructive">
        Could not load your Wishlist. Please try again.
      </p>
    )
  } else {
    body = (
      <AddToWishlist
        cardId={card.id}
        cardName={card.name}
        wanted={lookup.wanted[card.id]}
        onAnnounce={setAnnouncement}
      />
    )
  }

  return (
    <section aria-label="Wishlist" className="flex flex-col gap-3">
      <h2 className="font-heading text-lg font-medium">Wishlist</h2>
      {body}
      {/* Always mounted: a live region that appears together with its text is often not announced. */}
      <p role="status" className="text-sm empty:sr-only">
        {announcement}
      </p>
    </section>
  )
}
