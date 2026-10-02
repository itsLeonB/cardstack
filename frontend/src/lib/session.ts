import { useQueryClient } from "@tanstack/react-query"
import type { QueryClient } from "@tanstack/react-query"
import {
  getGetCurrentUserQueryKey,
  useGetCurrentUser,
  useLogin,
  useLogout,
  useRegister,
} from "@/generated/endpoints/auth/auth"
import type { MeResponse } from "@/generated/models"
import { setCsrfToken } from "./http"

// GET /auth/me never throws on a 401 (the fetch mutator resolves for every
// HTTP status), so an unauthenticated visitor reads as an ordinary
// `status: 401` response rather than a thrown query error/toast. A 5-minute
// staleTime avoids re-probing on every render, and retry is off because
// retrying a 401 can't turn it into a 200.
const SESSION_STALE_TIME = 5 * 60 * 1000

/**
 * Wraps `GET /auth/me` as the single source of truth for client-side auth
 * state, since the real session cookies are HttpOnly and unreadable by JS.
 */
export function useSession() {
  const query = useGetCurrentUser({
    query: {
      staleTime: SESSION_STALE_TIME,
      retry: false,
    },
  })

  const response = query.data
  const isAuthenticated = response?.status === 200
  const user: MeResponse | null =
    response && response.status === 200 ? response.data.data : null

  return {
    user,
    isAuthenticated,
    isLoading: query.isPending,
    query,
  }
}

// Routes read via ensureQueryData, so any user-scoped entry left in the cache
// would be served to the next user who signs in without a page reload. Drop
// everything except the session query, which is refetched instead.
//
// Returned so a mutation's `onSuccess` can await it: the caller's own
// `onSuccess` then runs against a settled session, and a `requireAuth` guard
// triggered by an immediate redirect (login -> the page the user wanted) can't
// read the stale pre-login 401 from the cache. Exported too, because a failed
// refresh has to drop the same dead session before sending the user to login.
export function resetCache(queryClient: QueryClient) {
  const sessionKey = getGetCurrentUserQueryKey()
  queryClient.removeQueries({
    predicate: (query) => query.queryKey[0] !== sessionKey[0],
  })
  // resetQueries (not invalidate) drops the old data, so a failed refetch
  // can't leave a stale 200/401 for the route guards to trust.
  return queryClient.resetQueries({ queryKey: sessionKey })
}

/** Login mutation that refreshes the session query once cookies are set. */
export function useLoginMutation() {
  const queryClient = useQueryClient()

  return useLogin({
    mutation: {
      onSuccess: (response) => {
        if (response.status === 200) {
          // The csrf_token cookie is on the backend's origin, not readable
          // by this page's JS once frontend/backend are cross-site (Vercel/
          // Railway) — the response body is the only place this page can
          // actually get it from. See http.ts's setCsrfToken doc comment.
          setCsrfToken(response.data.data.csrfToken ?? null)
          return resetCache(queryClient)
        }
      },
    },
  })
}

/** Register mutation. Registration alone doesn't establish a session. */
export function useRegisterMutation() {
  return useRegister()
}

/** Logout mutation that clears the session query once the backend confirms. */
export function useLogoutMutation() {
  const queryClient = useQueryClient()

  return useLogout({
    mutation: {
      onSuccess: (response) => {
        if (response.status === 204) {
          setCsrfToken(null)
          return resetCache(queryClient)
        }
      },
    },
  })
}
