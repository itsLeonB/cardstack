import { useQueryClient } from "@tanstack/react-query"
import type { QueryClient } from "@tanstack/react-query"
import { invalidateMasterInventory } from "@/lib/master-inventory"
import {
  getGetCollectionQueryKey,
  getListCollectionsQueryKey,
  useCreateCollection,
  useDeleteCollection,
  useUpdateCollection,
} from "@/generated/endpoints/collections/collections"

/**
 * Inventory Entry changes move a Collection's `cardCount`. The list route's
 * loader serves cached data, so refetch inactive queries now instead of
 * letting the next visit flash the old count.
 */
export function invalidateCollectionCounts(queryClient: QueryClient, collectionId: string) {
  void queryClient.invalidateQueries({ queryKey: getListCollectionsQueryKey(), refetchType: "all" })
  void queryClient.invalidateQueries({ queryKey: getGetCollectionQueryKey(collectionId), refetchType: "all" })
}

export function useCreateCollectionMutation() {
  const queryClient = useQueryClient()

  return useCreateCollection({
    mutation: {
      onSuccess: (response) => {
        if (response.status === 201) {
          queryClient.invalidateQueries({
            queryKey: getListCollectionsQueryKey(),
          })
        }
      },
    },
  })
}

export function useUpdateCollectionMutation() {
  const queryClient = useQueryClient()

  return useUpdateCollection({
    mutation: {
      onSuccess: (response, { id }) => {
        if (response.status === 200) {
          queryClient.invalidateQueries({
            queryKey: getListCollectionsQueryKey(),
          })
          // The edit loader reads this entry via ensureQueryData, and PUT is a
          // full replace, so a stale entry would overwrite the saved edit.
          queryClient.invalidateQueries({
            queryKey: getGetCollectionQueryKey(id),
          })
        }
      },
    },
  })
}

export function useDeleteCollectionMutation() {
  const queryClient = useQueryClient()

  return useDeleteCollection({
    mutation: {
      onSuccess: (response, { id }) => {
        if (response.status === 204) {
          queryClient.invalidateQueries({
            queryKey: getListCollectionsQueryKey(),
          })
          queryClient.removeQueries({ queryKey: getGetCollectionQueryKey(id) })
          // Deleting a Collection drops its entries from the Master Inventory totals.
          invalidateMasterInventory(queryClient)
        }
      },
    },
  })
}

// Success bodies are `{ data }` envelopes and error bodies carry `detail`; both
// fields are listed so every generated response variant is assignable.
type StatusResponse = {
  status: number
  data: { detail?: string; data?: unknown }
}

export function errorDetail(
  response: StatusResponse | undefined,
  fallback: string
) {
  if (!response || response.status === 200) return undefined
  return response.data.detail ?? fallback
}

export function collectionSubmitCallbacks({
  successStatus,
  failureMessage,
  onDone,
  setErrorMessage,
}: {
  successStatus: number
  failureMessage: string
  onDone: () => void
  setErrorMessage: (message: string) => void
}) {
  return {
    onSuccess: (response: StatusResponse) => {
      if (response.status === successStatus) {
        onDone()
        return
      }
      setErrorMessage(response.data.detail ?? failureMessage)
    },
    onError: () => {
      setErrorMessage("Could not reach the server. Please try again.")
    },
  }
}

export const NETWORK_ERROR = "Could not reach the server. Please try again."
