import { useQueryClient } from "@tanstack/react-query"
import {
  getGetCurrentUserQueryKey,
  useGetCurrentUser,
  useLogin,
  useLogout,
  useRegister,
} from "@/generated/endpoints/auth/auth"
import type { MeResponse } from "@/generated/models"

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

/** Login mutation that refreshes the session query once cookies are set. */
export function useLoginMutation() {
  const queryClient = useQueryClient()

  return useLogin({
    mutation: {
      onSuccess: (response) => {
        if (response.status === 200) {
          queryClient.invalidateQueries({
            queryKey: getGetCurrentUserQueryKey(),
          })
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
          queryClient.invalidateQueries({
            queryKey: getGetCurrentUserQueryKey(),
          })
        }
      },
    },
  })
}
