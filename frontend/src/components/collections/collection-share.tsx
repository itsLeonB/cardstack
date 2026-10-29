import { useState } from "react"
import { RiShareLine } from "@remixicon/react"
import { Button } from "@/components/ui/button"
import { ShareDialog } from "@/components/share/share-dialog"
import { useGetCollection } from "@/generated/endpoints/collections/collections"
import { useGetProfile } from "@/generated/endpoints/profile/profile"
import {
  NETWORK_ERROR,
  errorDetail,
  useShareCollectionMutation,
  useUnshareCollectionMutation,
} from "@/lib/collections"

/**
 * The Share button in a Collection's header. The link is built from the
 * caller's profile Slug and the Collection's own Slug, so the button stays
 * disabled until the profile is known: it never offers a link it cannot build.
 */
export function CollectionShare({ collectionId }: { collectionId: string }) {
  const [error, setError] = useState<string>()
  const query = useGetCollection(collectionId)
  const profile = useGetProfile()
  const share = useShareCollectionMutation()
  const unshare = useUnshareCollectionMutation()

  const collection =
    query.data?.status === 200 ? query.data.data.data : undefined
  const profileSlug =
    profile.data?.status === 200 ? profile.data.data.data.slug : undefined
  // A rejected request or an error status both count as failed, and a retry in
  // flight is loading again, not failed.
  const profileFailed =
    !profileSlug &&
    !profile.isFetching &&
    (profile.isError || profile.data !== undefined)
  if (!collection) return null
  if (!profileSlug)
    return (
      <div className="flex flex-wrap items-center gap-2">
        <Button variant="outline" size="sm" disabled>
          <RiShareLine data-icon="inline-start" />
          {collection.slug ? "Shared" : "Share"}
        </Button>
        {profileFailed && (
          <div role="alert" className="flex flex-wrap items-center gap-2">
            <p className="text-sm text-destructive">Could not load sharing</p>
            <Button
              variant="outline"
              size="sm"
              onClick={() => void profile.refetch()}
            >
              Retry
            </Button>
          </div>
        )}
      </div>
    )

  const link = collection.slug
    ? `${window.location.origin}/s/${profileSlug}/${collection.slug}`
    : ""

  function outcome(fallback: string) {
    return {
      onSuccess: (response: Parameters<typeof errorDetail>[0]) =>
        setError(errorDetail(response, fallback)),
      onError: () => setError(NETWORK_ERROR),
    }
  }

  function change(next: boolean) {
    setError(undefined)
    if (next)
      share.mutate(
        { id: collectionId },
        outcome("Could not share this Collection.")
      )
    else
      unshare.mutate(
        { id: collectionId },
        outcome("Could not stop sharing this Collection.")
      )
  }

  return (
    <ShareDialog
      shared={Boolean(collection.slug)}
      link={link}
      pending={share.isPending || unshare.isPending}
      error={error}
      onShare={() => change(true)}
      onUnshare={() => change(false)}
    />
  )
}
