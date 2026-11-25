import { useEffect, useRef } from "react"
import { useQuery, useQueryClient } from "@tanstack/react-query"
import type { QueryClient } from "@tanstack/react-query"
import { useRouter } from "@tanstack/react-router"

import { clerkAuth } from "@/lib/clerk-auth"
import {
  getTrade,
  getGetTradeQueryKey,
  getGetTradeQueryOptions,
} from "@/generated/endpoints/trades/trades"
import type { TradeView } from "@/generated/models"

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
