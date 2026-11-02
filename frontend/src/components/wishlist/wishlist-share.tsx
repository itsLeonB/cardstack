import { useState } from "react"
import { useQueryClient } from "@tanstack/react-query"
import { ShareDialog } from "@/components/share/share-dialog"
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
 * GET /wishlist/meta and offers the Share dialog to toggle it.
 */
export function WishlistShare() {
  const queryClient = useQueryClient()
  const [error, setError] = useState<string>()
  const profile = useGetProfile()
  const profileSlug =
    profile.data?.status === 200 ? profile.data.data.data.slug : undefined
  const wishlistMeta = useGetWishlist({
    query: { refetchOnMount: "always" },
  })
  const share = useShareWishlist()
  const unshare = useUnshareWishlist()

  // Only consider the wishlist shared if we have both the profile and the meta flag.
  // Without a profile slug we cannot build the link, so the dialog cannot show it.
  const isShared =
    profileSlug !== undefined &&
    wishlistMeta.data?.status === 200 &&
    wishlistMeta.data.data.data.shared
  const pending =
    share.isPending ||
    unshare.isPending ||
    wishlistMeta.isFetching ||
    profile.isFetching

  async function refresh() {
    await wishlistMeta.refetch()
    if (profileSlug) void invalidateSharedWishlist(queryClient, profileSlug)
  }

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
      shared={isShared ?? false}
      link={`${window.location.origin}/s/${profileSlug}/wishlist`}
      pending={pending}
      error={error}
      onShare={() => change(true)}
      onUnshare={() => change(false)}
    />
  )
}
