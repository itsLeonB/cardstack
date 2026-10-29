import { useState } from "react"
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
 * caller's profile Slug and the Collection's own Slug, so nothing renders until
 * both are known: the button never offers a link it cannot build.
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
  if (!collection || !profileSlug) return null

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
