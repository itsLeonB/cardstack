import { useState } from "react"
import { useClerk, useUser } from "@clerk/react"
import type { QueryClient } from "@tanstack/react-query"
import { toast } from "sonner"
import { LOGIN_PATH } from "./auth-paths"
import { rearmAuthLost, reportAuthLost } from "./http"

export interface SessionUser {
  id: string
  /** The account's full name, absent for plain email sign-ups. */
  name: string | null
  email: string | null
}

/**
 * Clerk-derived auth state: the single source of truth for the shell and for
 * public pages gating features on `isAuthenticated`. `isLoading` is true until
 * Clerk has loaded, which is when "signed out" cannot be told from "not known yet".
 */
export function useSession() {
  const { isLoaded, isSignedIn, user } = useUser()
  const isAuthenticated = isLoaded && isSignedIn === true

  const sessionUser: SessionUser | null =
    isAuthenticated && user
      ? {
          id: user.id,
          name: user.fullName,
          email: user.primaryEmailAddress?.emailAddress ?? null,
        }
      : null

  return { user: sessionUser, isAuthenticated, isLoading: !isLoaded }
}

// Routes read via ensureQueryData, so any user-scoped entry left in the cache
// would be served to the next user who signs in (or to the guest after a
// sign-out) without a page reload. Dropping everything is right: the app has
// no query that outlives a session.
export function resetCache(queryClient: QueryClient) {
  queryClient.removeQueries()
}

// Set by `useSignOut` so the sign-out it asked for is not mistaken for a session
// Clerk lost; the next sign-out transition consumes it. Cleared if Clerk fails.
let deliberateSignOut = false

/**
 * What a change of Clerk session (sign-in, sign-out, another account) does to
 * the app, wherever it came from, including another tab: drop the old
 * session's data, and re-arm the lost-session notice once a new session
 * exists. A session that vanished without our asking (ended elsewhere,
 * revoked) while a private page is mounted is announced like any lost
 * session, which moves the user to sign-in; a guest-only change is not.
 */
export function createSessionChangeHandler(
  queryClient: QueryClient,
  isOnPrivatePage: () => boolean
) {
  return (current: string | null, previous: string | null) => {
    resetCache(queryClient)
    if (current) {
      rearmAuthLost()
      return
    }
    const asked = deliberateSignOut
    deliberateSignOut = false
    if (!asked && previous && isOnPrivatePage()) reportAuthLost()
  }
}

export const LOGOUT_FAILED = "Could not log out. You are still signed in."

/**
 * Ends the Clerk session and leaves for the login page. A failure toasts
 * instead of navigating, since the user is still signed in. The query cache is
 * dropped by the session-change handler rather than here.
 */
export function useSignOut() {
  const { signOut } = useClerk()
  const [isPending, setIsPending] = useState(false)

  async function handleSignOut() {
    setIsPending(true)
    deliberateSignOut = true
    try {
      await signOut({ redirectUrl: LOGIN_PATH })
    } catch {
      deliberateSignOut = false
      toast.error(LOGOUT_FAILED)
    } finally {
      setIsPending(false)
    }
  }

  return { signOut: handleSignOut, isPending }
}
