import type { QueryClient } from "@tanstack/react-query"
import { toast } from "sonner"
import { resetCache } from "./session"

export const SESSION_EXPIRED_MESSAGE =
  "Your session has expired. Please log in again."

interface AuthLostActions {
  /** Signs out of Clerk if needed, then calls the callback either way. */
  endSession: (then: () => void) => Promise<void>
  /** The path and query the user is on, read before anything navigates. */
  attemptedPath: () => string
  redirectToLogin: (attemptedPath: string) => void
}

/**
 * Builds the handler `http.ts` calls when the session was lost (see
 * `setOnAuthLost`). It needs the query cache and the router, neither of which
 * `http.ts` has, so `router.tsx` assembles it here and injects the navigation
 * and the Clerk sign-out, which also keeps it testable without either.
 *
 * Clerk is signed out before the redirect: the API refused a session Clerk
 * still holds, and with that session alive `requireGuest` would bounce the
 * user from the login page straight back, leaving a signed-in header. The
 * attempted path is read first because Clerk's sign-out navigates away.
 */
export function createAuthLostHandler(
  queryClient: QueryClient,
  { endSession, attemptedPath, redirectToLogin }: AuthLostActions
) {
  return () => {
    const attempted = attemptedPath()
    resetCache(queryClient)
    toast.error(SESSION_EXPIRED_MESSAGE)
    void endSession(() => redirectToLogin(attempted))
  }
}
