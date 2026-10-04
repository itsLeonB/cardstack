import type { QueryClient } from "@tanstack/react-query"
import { toast } from "sonner"
import { resetCache } from "./session"

export const SESSION_EXPIRED_MESSAGE =
  "Your session has expired. Please log in again."

/**
 * Builds the handler `http.ts` calls when a request that carried a token still
 * got a 401 after a fresh one (see `setOnAuthLost`). It needs the query cache
 * and the router, neither of which `http.ts` has, so `router.tsx` assembles it
 * here and injects the navigation, which also keeps it testable without a router.
 */
export function createAuthLostHandler(
  queryClient: QueryClient,
  redirectToLogin: () => void
) {
  return () => {
    resetCache(queryClient)
    toast.error(SESSION_EXPIRED_MESSAGE)
    redirectToLogin()
  }
}
