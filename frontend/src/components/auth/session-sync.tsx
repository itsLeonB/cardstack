import { useEffect, useRef } from "react"
import { useAuth } from "@clerk/react"
import { useQueryClient } from "@tanstack/react-query"
import { handleSessionChange } from "@/lib/session"

/**
 * Renders nothing. Reacts to the Clerk session changing after the first load
 * (sign-in, sign-out, another account, or either of those in another tab) by
 * dropping the old session's cached data. The first load is just recorded.
 */
export function SessionSync() {
  const { isLoaded, sessionId } = useAuth()
  const queryClient = useQueryClient()
  const known = useRef<string | null | undefined>(undefined)

  useEffect(() => {
    if (!isLoaded) return
    const current = sessionId ?? null
    if (known.current !== undefined && known.current !== current) {
      handleSessionChange(queryClient)
    }
    known.current = current
  }, [isLoaded, sessionId, queryClient])

  return null
}
