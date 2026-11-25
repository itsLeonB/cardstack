import { useEffect, useRef, useState } from "react"
import { toast } from "sonner"
import { NETWORK_ERROR } from "@/lib/collections"
import { useQuery, useQueryClient } from "@tanstack/react-query"
import type { QueryClient } from "@tanstack/react-query"
import { useRouter } from "@tanstack/react-router"

import { clerkAuth } from "@/lib/clerk-auth"
import {
  getTrade,
  getListTradesQueryKey,
  getGetTradeQueryKey,
  getGetTradeQueryOptions,
  useJoinTrade,
  useAcceptTrade,
  useConfirmTrade,
  useAddTradeItem,
  useRemoveTradeItem,
} from "@/generated/endpoints/trades/trades"
import type { TradeItemInputBody, TradeView } from "@/generated/models"
import type {
  acceptTradeResponse,
  confirmTradeResponse,
} from "@/generated/endpoints/trades/trades"

type SessionTrade = {
  sessionVersion: number
  trade: TradeView | "missing" | null
}

// Read a Trade in the caller's session: a response that crosses a session
// change is dropped, and both attempts are checked for the same reason.
export async function loadTradeForSession(
  queryClient: QueryClient,
  slug: string
): Promise<SessionTrade> {
  const read = () =>
    queryClient.fetchQuery({
      ...getGetTradeQueryOptions(slug),
      staleTime: 0,
    })
  for (let attempt = 0; attempt < 2; attempt += 1) {
    const sessionVersion = clerkAuth.currentSessionVersion()
    const response = await read()
    if (clerkAuth.currentSessionVersion() !== sessionVersion) {
      queryClient.removeQueries({ queryKey: getGetTradeQueryKey(slug) })
      continue
    }
    if (response.status === 404) {
      queryClient.removeQueries({ queryKey: getGetTradeQueryKey(slug) })
      return { sessionVersion, trade: "missing" }
    }
    return {
      sessionVersion,
      trade: response.status === 200 ? response.data.data : null,
    }
  }
  return { sessionVersion: clerkAuth.currentSessionVersion(), trade: null }
}

export function useTradePolling({ trade, sessionVersion }: SessionTrade) {
  const router = useRouter()
  const queryClient = useQueryClient()
  const slug = trade && trade !== "missing" ? trade.slug : ""
  const current = () => clerkAuth.currentSessionVersion() === sessionVersion
  const poll = useQuery({
    queryKey: ["trade-poll", slug, sessionVersion],
    queryFn: async () => {
      if (!current()) throw new Error("Session changed")
      const response = await getTrade(slug)
      if (!current()) throw new Error("Session changed")
      return response
    },
    enabled: !!trade && trade !== "missing" && trade.isParticipant && current(),
    initialData: null,
    staleTime: Infinity,
    refetchOnMount: false,
    refetchOnWindowFocus: false,
    refetchInterval: 10_000,
    retry: false,
  })
  const handled = useRef<{
    slug: string
    sessionVersion: number
    updatedAt: number
  } | null>(null)
  useEffect(() => {
    if (!trade || trade === "missing" || !current() || !poll.data) return
    if (
      handled.current?.slug === slug &&
      handled.current.sessionVersion === sessionVersion &&
      handled.current.updatedAt === poll.dataUpdatedAt
    )
      return
    handled.current = { slug, sessionVersion, updatedAt: poll.dataUpdatedAt }
    if (poll.data.status === 404) {
      void router.invalidate()
      return
    }
    const polled = poll.data?.status === 200 ? poll.data.data.data : null
    if (!polled || JSON.stringify(polled) === JSON.stringify(trade)) return
    void queryClient
      .invalidateQueries({ queryKey: getGetTradeQueryKey(slug) })
      .then(() => {
        if (current()) void router.invalidate()
      })
  }, [
    trade,
    sessionVersion,
    poll.data,
    poll.dataUpdatedAt,
    queryClient,
    router,
    slug,
  ])
}

function useTradeItemSync() {
  const router = useRouter()
  const queryClient = useQueryClient()

  return () => {
    const version = clerkAuth.currentSessionVersion()
    const current = () => clerkAuth.currentSessionVersion() === version
    return {
      current,
      async refresh(
        slug: string,
        response: Awaited<ReturnType<typeof getTrade>>
      ) {
        if (!current()) return false
        await queryClient.cancelQueries({ queryKey: ["trade-poll", slug] })
        if (!current()) return false
        await queryClient.cancelQueries({ queryKey: getGetTradeQueryKey(slug) })
        if (!current()) return false
        queryClient.setQueryData(getGetTradeQueryKey(slug), response)
        await queryClient.invalidateQueries({
          queryKey: getListTradesQueryKey(),
        })
        if (!current()) return false
        await router.invalidate()
        return current()
      },
    }
  }
}

export function useAddTradeItemAction(
  slug: string,
  onError: (message: string) => void,
  onAdded: () => void
) {
  const startSync = useTradeItemSync()
  const mutation = useAddTradeItem()
  return {
    pending: mutation.isPending,
    add(data: TradeItemInputBody) {
      const sync = startSync()
      mutation.mutate(
        { slug, data },
        {
          onSuccess: async (response) => {
            if (!sync.current()) return
            if (response.status !== 200) {
              onError(
                response.data.detail ??
                  "Could not add this Card. Check the Trade and try again."
              )
              return
            }
            if ((await sync.refresh(slug, response)) && sync.current())
              onAdded()
          },
          onError: () => {
            if (sync.current()) onError(NETWORK_ERROR)
          },
        }
      )
    },
  }
}

export function useRemoveTradeItemAction(
  slug: string,
  onError: (message: string) => void
) {
  const startSync = useTradeItemSync()
  const mutation = useRemoveTradeItem()
  return {
    pending: mutation.isPending,
    remove(itemId: string) {
      const sync = startSync()
      mutation.mutate(
        { slug, itemId },
        {
          onSuccess: (response) => {
            if (!sync.current()) return
            if (response.status === 200) void sync.refresh(slug, response)
            else
              onError(
                "Could not remove this Card. Check the Trade and try again."
              )
          },
          onError: () => {
            if (sync.current()) onError(NETWORK_ERROR)
          },
        }
      )
    },
  }
}

export function useTradeParticipationSync() {
  const router = useRouter()
  const queryClient = useQueryClient()
  const version = clerkAuth.currentSessionVersion()
  const current = () => clerkAuth.currentSessionVersion() === version

  return {
    current,
    async refresh(slug: string) {
      if (!current()) return
      await queryClient.cancelQueries({ queryKey: ["trade-poll", slug] })
      if (!current()) return
      await queryClient.cancelQueries({ queryKey: getGetTradeQueryKey(slug) })
      if (!current()) return
      await queryClient.invalidateQueries({
        queryKey: getGetTradeQueryKey(slug),
      })
      if (!current()) return
      await queryClient.invalidateQueries({ queryKey: getListTradesQueryKey() })
      if (current()) await router.invalidate()
    },
  }
}

export function useJoinTradeAction(slug: string) {
  const sync = useTradeParticipationSync()
  const mutation = useJoinTrade()
  const [seatTaken, setSeatTaken] = useState(false)
  return {
    seatTaken,
    pending: mutation.isPending,
    join() {
      mutation.mutate(
        { slug },
        {
          onSuccess: async (response) => {
            if (!sync.current()) return
            if (response.status === 200) {
              void sync.refresh(slug)
            } else if (response.data.code === "trade_seat_taken") {
              setSeatTaken(true)
              try {
                const current = await getTrade(slug)
                if (!sync.current()) return
                if (current.status === 200) void sync.refresh(slug)
                else if (current.status !== 404)
                  toast.error(
                    current.data.detail ?? "Could not load this Trade."
                  )
              } catch {
                if (sync.current()) toast.error(NETWORK_ERROR)
              }
            } else {
              toast.error(response.data.detail ?? "Could not join this Trade.")
            }
          },
          onError: () => {
            if (sync.current()) toast.error(NETWORK_ERROR)
          },
        }
      )
    },
  }
}

function participationResultHandlers(
  sync: ReturnType<typeof useTradeParticipationSync>,
  slug: string,
  errorMessage: string
) {
  return {
    onSuccess: (response: acceptTradeResponse | confirmTradeResponse) => {
      if (!sync.current()) return
      if (response.status !== 200)
        toast.error(response.data.detail ?? errorMessage)
      void sync.refresh(slug)
    },
    onError: () => {
      if (sync.current()) toast.error(NETWORK_ERROR)
    },
  }
}

export function useAcceptTradeAction(trade: TradeView) {
  const sync = useTradeParticipationSync()
  const mutation = useAcceptTrade()
  return {
    pending: mutation.isPending,
    accept() {
      mutation.mutate(
        { slug: trade.slug, data: { revision: trade.revision } },
        participationResultHandlers(
          sync,
          trade.slug,
          "Could not accept the Trade."
        )
      )
    },
  }
}

export function useConfirmTradeAction(slug: string) {
  const sync = useTradeParticipationSync()
  const mutation = useConfirmTrade()
  return {
    pending: mutation.isPending,
    error:
      mutation.data?.status && mutation.data.status !== 200
        ? (mutation.data.data.detail ?? "Could not confirm the handover.")
        : null,
    confirm() {
      mutation.mutate(
        { slug },
        participationResultHandlers(
          sync,
          slug,
          "Could not confirm the handover."
        )
      )
    },
  }
}
