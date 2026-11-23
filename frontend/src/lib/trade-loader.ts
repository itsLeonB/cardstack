import type { QueryClient } from "@tanstack/react-query"

import { clerkAuth } from "@/lib/clerk-auth"
import {
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
