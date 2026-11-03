import { useState } from "react"
import { useQueryClient } from "@tanstack/react-query"
import { RiShareLine } from "@remixicon/react"
import { ShareDialog } from "@/components/share/share-dialog"
import { Button } from "@/components/ui/button"
import { useGetProfile } from "@/generated/endpoints/profile/profile"
import {
  useShareWishlist,
  useUnshareWishlist,
  useGetWishlist,
} from "@/generated/endpoints/wishlist/wishlist"
import { NETWORK_ERROR } from "@/lib/collections"
import { invalidateSharedWishlist } from "@/lib/wishlist"

/**
 * The Share button in the Wishlist's header. Reads the shared flag from
 * GET /wishlist/meta and offers the Share dialog to toggle it. The button stays
 * disabled until that flag is known: it never offers to share on a state it
 * has no answer for, and a failed read says so inline with a way to read again.
 */
export function WishlistShare() {
  const queryClient = useQueryClient()
  const [error, setError] = useState<string>()
  const profile = useGetProfile()
  const wishlistMeta = useGetWishlist({
    query: { refetchOnMount: "always" },
  })
  const share = useShareWishlist()
  const unshare = useUnshareWishlist()

  const metaFailed =
    !wishlistMeta.isFetching &&
    (wishlistMeta.isError ||
      (wishlistMeta.data !== undefined && wishlistMeta.data.status !== 200))
  const profileFailed =
    !profile.isFetching &&
    (profile.isError ||
      (profile.data !== undefined && profile.data.status !== 200))
  const loadFailed = metaFailed || profileFailed
  const loaded = wishlistMeta.data && profile.data
  const shared =
    loaded && wishlistMeta.data.status === 200 && profile.data.status === 200
      ? wishlistMeta.data.data.data.shared
      : undefined
  const profileSlug =
    loaded && profile.data.status === 200
      ? profile.data.data.data.slug
      : undefined
  // Only consider the wishlist shared if we have both the profile and the meta flag.
  // Without a profile slug we cannot build the link, so the dialog cannot show it.
  const isShared = shared === true && profileSlug !== undefined
  const pending =
    share.isPending ||
    unshare.isPending ||
    wishlistMeta.isFetching ||
    profile.isFetching

  async function refresh() {
    await wishlistMeta.refetch()
    await profile.refetch()
    if (profileSlug) void invalidateSharedWishlist(queryClient, profileSlug)
  }

  if (loadFailed || shared === undefined)
    return (
      <div className="flex flex-wrap items-center gap-2">
        <Button variant="outline" size="sm" disabled>
          <RiShareLine data-icon="inline-start" />
          Share
        </Button>
        {loadFailed && (
          <div role="alert" className="flex flex-wrap items-center gap-2">
            <p className="text-sm text-destructive">Could not load sharing</p>
            <Button variant="outline" size="sm" onClick={() => void refresh()}>
              Retry
            </Button>
          </div>
        )}
      </div>
    )

  // A share answers with the Slug (200), an unshare with no content (204);
  // either way the change shows once the flag is read again.
  function change(next: boolean) {
    setError(undefined)
    if (next)
      share.mutate(undefined, {
        onSuccess: async (response) => {
          if (response.status === 200) await refresh()
          else
            setError(response.data.detail ?? "Could not share your Wishlist.")
        },
        onError: () => setError(NETWORK_ERROR),
      })
    else
      unshare.mutate(undefined, {
        onSuccess: async (response) => {
          if (response.status === 204) await refresh()
          else
            setError(
              response.data.detail ?? "Could not stop sharing your Wishlist."
            )
        },
        onError: () => setError(NETWORK_ERROR),
      })
  }

  return (
    <ShareDialog
      shared={isShared}
      link={`${window.location.origin}/s/${profileSlug}/wishlist`}
      pending={pending}
      error={error}
      onShare={() => change(true)}
      onUnshare={() => change(false)}
    />
  )
}
