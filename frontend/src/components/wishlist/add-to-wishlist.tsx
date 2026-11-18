import { useState } from "react"
import { useQueryClient } from "@tanstack/react-query"
import { RiAddLine, RiSubtractLine } from "@remixicon/react"
import { SignInLink } from "@/components/auth/sign-in-link"
import { Button } from "@/components/ui/button"
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/components/ui/popover"
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
  const [savedWanted, setSavedWanted] = useState<{
    original: number | undefined
    quantity: number
  } | null>(null)
  const currentWanted =
    savedWanted && savedWanted.original === wanted
      ? savedWanted.quantity
      : wanted
  const [quantity, setQuantity] = useState(wanted ?? 1)
  const [error, setError] = useState("")
  const [open, setOpen] = useState(false)

  function changeOpen(nextOpen: boolean) {
    setOpen(nextOpen)
    if (nextOpen) {
      setQuantity(currentWanted ?? 1)
      setError("")
    }
  }

  function saveItem() {
    if (save.isPending || quantity < 1) return
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
          setSavedWanted({ original: wanted, quantity })
          invalidateWishlist(queryClient)
          setOpen(false)
        },
        onError: () => setError(NETWORK_ERROR),
      }
    )
  }

  return (
    <Popover open={open} onOpenChange={changeOpen}>
      <PopoverTrigger
        render={
          <Button variant="outline" size="sm" className="min-h-11 gap-2" />
        }
        aria-label={`Edit Wishlist quantity for ${cardName}${currentWanted === undefined ? "" : `, wanted ${currentWanted}`}`}
      >
        Wishlist
        {currentWanted !== undefined && (
          <span className="font-medium">Wanted ×{currentWanted}</span>
        )}
      </PopoverTrigger>
      <PopoverContent
        aria-label={`Wishlist for ${cardName}`}
        className="motion-reduce:duration-0 motion-reduce:data-open:animate-none motion-reduce:data-closed:animate-none"
      >
        <div className="flex flex-col gap-3">
          <h2 className="font-heading font-medium">Wishlist for {cardName}</h2>
          <p className="text-sm">Wanted quantity: {quantity}</p>
          <div className="flex items-center gap-2">
            <Button
              type="button"
              variant="outline"
              size="icon"
              className="size-11"
              aria-label={`Decrease wanted quantity of ${cardName}`}
              disabled={quantity <= 1 || save.isPending}
              onClick={() => setQuantity(quantity - 1)}
            >
              <RiSubtractLine aria-hidden="true" />
            </Button>
            <Button
              type="button"
              variant="outline"
              size="icon"
              className="size-11"
              aria-label={`Increase wanted quantity of ${cardName}`}
              disabled={quantity >= MAX_WANTED || save.isPending}
              onClick={() => setQuantity(quantity + 1)}
            >
              <RiAddLine aria-hidden="true" />
            </Button>
          </div>
          {error && (
            <p role="alert" className="text-sm text-destructive">
              {error}
            </p>
          )}
          <div className="flex justify-end gap-2">
            <Button
              type="button"
              variant="outline"
              className="min-h-11"
              disabled={save.isPending}
              onClick={() => changeOpen(false)}
            >
              Cancel
            </Button>
            <Button
              type="button"
              className="min-h-11"
              disabled={save.isPending || quantity < 1}
              onClick={saveItem}
            >
              {save.isPending ? "Saving…" : "Save"}
            </Button>
          </div>
        </div>
      </PopoverContent>
    </Popover>
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
      {body}
      {/* Always mounted: a live region that appears together with its text is often not announced. */}
      <p role="status" className="text-sm empty:sr-only">
        {announcement}
      </p>
    </section>
  )
}
