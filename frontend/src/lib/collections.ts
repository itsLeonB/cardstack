import { useQueryClient } from "@tanstack/react-query"
import {
  getListCollectionsQueryKey,
  useCreateCollection,
  useDeleteCollection,
  useUpdateCollection,
} from "@/generated/endpoints/collections/collections"

/** Create mutation that refreshes the Collections list once the Collection exists. */
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

/** Update mutation that refreshes the Collections list once the edit is saved. */
export function useUpdateCollectionMutation() {
  const queryClient = useQueryClient()

  return useUpdateCollection({
    mutation: {
      onSuccess: (response) => {
        if (response.status === 200) {
          queryClient.invalidateQueries({
            queryKey: getListCollectionsQueryKey(),
          })
        }
      },
    },
  })
}

/** Delete mutation that refreshes the Collections list once the delete is confirmed by the backend. */
export function useDeleteCollectionMutation() {
  const queryClient = useQueryClient()

  return useDeleteCollection({
    mutation: {
      onSuccess: (response) => {
        if (response.status === 204) {
          queryClient.invalidateQueries({
            queryKey: getListCollectionsQueryKey(),
          })
        }
      },
    },
  })
}
