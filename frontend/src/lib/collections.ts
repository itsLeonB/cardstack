import { useQueryClient } from "@tanstack/react-query"
import {
  getGetCollectionQueryKey,
  getListCollectionsQueryKey,
  useCreateCollection,
  useDeleteCollection,
  useUpdateCollection,
} from "@/generated/endpoints/collections/collections"

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
        }
      },
    },
  })
}
