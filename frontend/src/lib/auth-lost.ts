import type { QueryClient } from "@tanstack/react-query"
import { toast } from "sonner"
import { setCsrfToken } from "./http"
import { resetCache } from "./session"

export const SESSION_EXPIRED_MESSAGE =
  "Your session has expired. Please log in again."

/**
 * Builds the handler `http.ts` calls when a refresh fails on a request that
 * needed the session (see `setOnAuthLost`). It needs the query cache and the
 * router, neither of which `http.ts` has, so `router.tsx` assembles it here
 * and injects the navigation — which also keeps it testable without a router.
 */
export function createAuthLostHandler(
  queryClient: QueryClient,
  redirectToLogin: () => void
) {
  return () => {
    // Not awaited: `resetCache` drops the cached queries synchronously, and
    // its session refetch is not worth holding the toast and the redirect
    // behind a second round trip to a backend that just said no.
    void resetCache(queryClient).catch(() => {})
    setCsrfToken(null)
    toast.error(SESSION_EXPIRED_MESSAGE)
    redirectToLogin()
  }
}
