import { useQueryClient } from "@tanstack/react-query"
import type { InfiniteData, QueryClient } from "@tanstack/react-query"
import { invalidateMasterInventory } from "@/lib/master-inventory"
import {
  getGetCollectionQueryKey,
  getListCollectionsQueryKey,
  useCreateCollection,
  useDeleteCollection,
  useUpdateCollection,
} from "@/generated/endpoints/collections/collections"
import {
  getListCollectionEntriesInfiniteQueryKey,
  getListCollectionEntriesQueryKey,
} from "@/generated/endpoints/inventory/inventory"
import type { listCollectionEntries } from "@/generated/endpoints/inventory/inventory"
import type {
  InventoryChangeResult,
  ListCollectionEntriesParams,
} from "@/generated/models"

/**
 * Refreshes a Collection's entries wherever they are cached. Two key roots, so
 * both are needed: the infinite list on the Collection page (keys start with
 * `'infinite'`) and the plain quantity lookups of the catalog search page.
 */
export function invalidateCollectionEntries(
  queryClient: QueryClient,
  collectionId: string,
  /** Only refresh the plain lookups that asked for one of these cards (one request each, not one per loaded page). */
  cardIds?: string[]
) {
  void queryClient.invalidateQueries({
    queryKey: getListCollectionEntriesInfiniteQueryKey(collectionId),
  })
  const [root] = getListCollectionEntriesQueryKey(collectionId)
  void queryClient.invalidateQueries({
    queryKey: getListCollectionEntriesQueryKey(collectionId),
    predicate: ({ queryKey }) => {
      if (!cardIds) return true
      // SAFETY: keys under this root are built by getListCollectionEntriesQueryKey, [url, params?].
      const [, params] = queryKey as [string, ListCollectionEntriesParams?]
      return (
        queryKey[0] === root &&
        !!params?.cardId?.some((id) => cardIds.includes(id))
      )
    },
  })
}

type EntriesPage = Awaited<ReturnType<typeof listCollectionEntries>>

/**
 * Writes saved quantities into every cached page of a Collection's infinite
 * entries list, so the cache stays the source of truth without a refetch. A
 * card saved at 0 stays in the cache at 0 (its tile stays on screen) and goes
 * away with the next real refetch. Pages with no change keep their identity,
 * and so does the whole cache entry when nothing changed.
 *
 * Cancels fetches in flight first: a page append (or refetch) that started
 * before the save writes back the pages it saw when it started, which would
 * undo the patch. The grid asks for the cancelled page again.
 */
export async function patchCollectionEntryQuantities(
  queryClient: QueryClient,
  collectionId: string,
  results: InventoryChangeResult[]
) {
  const queryKey = getListCollectionEntriesInfiniteQueryKey(collectionId)
  // Only lists holding pages: cancelling a first fetch (a filter just changed)
  // would leave it idle, and nothing would start it again.
  await queryClient.cancelQueries({
    queryKey,
    predicate: (query) => query.state.data !== undefined,
  })
  const saved = new Map(results.map((r) => [r.cardId, r.quantity]))
  const changed = (page: EntriesPage) =>
    page.status === 200 &&
    page.data.data?.some(
      (item) =>
        saved.has(item.card.id) && saved.get(item.card.id) !== item.quantity
    )
  queryClient.setQueriesData<InfiniteData<EntriesPage, number | undefined>>(
    { queryKey },
    (cached) => {
      if (!cached?.pages.some(changed)) return cached
      return {
        ...cached,
        pages: cached.pages.map((page) => {
          if (page.status !== 200 || !changed(page)) return page
          return {
            ...page,
            data: {
              ...page.data,
              data:
                page.data.data?.map((item) =>
                  saved.has(item.card.id)
                    ? { ...item, quantity: saved.get(item.card.id)! }
                    : item
                ) ?? null,
            },
          }
        }),
      }
    }
  )
}

/**
 * Inventory Entry changes move a Collection's `cardCount`. The list route's
 * loader serves cached data, so refetch inactive queries now instead of
 * letting the next visit flash the old count.
 */
export function invalidateCollectionCounts(
  queryClient: QueryClient,
  collectionId: string
) {
  void queryClient.invalidateQueries({
    queryKey: getListCollectionsQueryKey(),
    refetchType: "all",
  })
  void queryClient.invalidateQueries({
    queryKey: getGetCollectionQueryKey(collectionId),
    refetchType: "all",
  })
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
