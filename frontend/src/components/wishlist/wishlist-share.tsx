import { useState } from "react"
import { RiShareLine } from "@remixicon/react"
import { useQueryClient } from "@tanstack/react-query"
import { Button } from "@/components/ui/button"
import { ShareDialog } from "@/components/share/share-dialog"
import { useGetProfile } from "@/generated/endpoints/profile/profile"
import { useGetSharedWishlist } from "@/generated/endpoints/shared/shared"
import {
  useShareWishlist,
  useUnshareWishlist,
} from "@/generated/endpoints/wishlist/wishlist"
import { NETWORK_ERROR } from "@/lib/collections"
import { invalidateSharedWishlist } from "@/lib/wishlist"

/**
 * The Share button in the Wishlist's header. The API has no flag to read, so
 * whether it is Shared is its own Shared page's answer for the owner's Slug:
 * 200 is Shared, 404 is not (ADR-0025). The button stays disabled until that
 * answer is in: it never offers a link it cannot build.
 */
export function WishlistShare() {
  const queryClient = useQueryClient()
  const [error, setError] = useState<string>()
  const profile = useGetProfile()
  const profileSlug =
    profile.data?.status === 200 ? profile.data.data.data.slug : undefined
  const shared = useGetSharedWishlist(profileSlug ?? "", {
    // Fresh on every visit: sharing may have been stopped elsewhere since.
    query: { enabled: Boolean(profileSlug), staleTime: 0 },
  })
  const share = useShareWishlist()
  const unshare = useUnshareWishlist()

  const isShared = shared.data?.status === 200
  // A rejected request or an error status both count as failed, and a retry in
  // flight is loading again, not failed. A 404 is the answer "not Shared".
  const profileFailed =
    !profileSlug &&
    !profile.isFetching &&
    (profile.isError || profile.data !== undefined)
  const sharedFailed =
    shared.data !== undefined &&
    shared.data.status !== 200 &&
    shared.data.status !== 404
  const failed = profileFailed || sharedFailed
  const pending =
    share.isPending ||
    unshare.isPending ||
    shared.isFetching ||
    profile.isFetching

  function refresh() {
    if (!profileSlug) return
    void invalidateSharedWishlist(queryClient, profileSlug)
  }

  // Sharing is off by default, so a missing answer shows the off label rather
  // than a link: the switch is never offered before the flag is known.
  if (failed || shared.data === undefined)
    return (
      <div className="flex flex-wrap items-center gap-2">
        <Button variant="outline" size="sm" disabled>
          <RiShareLine data-icon="inline-start" />
          Share
        </Button>
        {failed && (
          <div role="alert" className="flex flex-wrap items-center gap-2">
            <p className="text-sm text-destructive">Could not load sharing</p>
            <Button
              variant="outline"
              size="sm"
              onClick={() => {
                void profile.refetch()
                refresh()
              }}
            >
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
        onSuccess: (response) => {
          if (response.status === 200) return refresh()
          setError(response.data.detail ?? "Could not share your Wishlist.")
        },
        onError: () => setError(NETWORK_ERROR),
      })
    else
      unshare.mutate(undefined, {
        onSuccess: (response) => {
          if (response.status === 204) return refresh()
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
