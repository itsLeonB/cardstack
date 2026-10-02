import { afterEach, describe, expect, it, vi } from "vitest"
import { QueryClient } from "@tanstack/react-query"
import { toast } from "sonner"
import { getGetCurrentUserQueryKey } from "@/generated/endpoints/auth/auth"
import { SESSION_EXPIRED_MESSAGE, createAuthLostHandler } from "./auth-lost"
import { setCsrfToken } from "./http"

const COLLECTIONS_KEY = ["/collections"]
const CSRF_STORAGE_KEY = "csrf_token"

describe("createAuthLostHandler", () => {
  afterEach(() => {
    setCsrfToken(null)
    vi.restoreAllMocks()
  })

  it("drops the cached user data, toasts and sends the user to login", () => {
    const queryClient = new QueryClient()
    queryClient.setQueryData(COLLECTIONS_KEY, { status: 200 })
    setCsrfToken("csrf-token")
    const errorToast = vi.spyOn(toast, "error")
    const resetSpy = vi.spyOn(queryClient, "resetQueries")
    const redirectToLogin = vi.fn(() => {
      // The token must be gone before the login page renders, so a stale
      // one can't ride along on the next mutating request.
      expect(sessionStorage.getItem(CSRF_STORAGE_KEY)).toBeNull()
    })

    createAuthLostHandler(queryClient, redirectToLogin)()

    expect(queryClient.getQueryData(COLLECTIONS_KEY)).toBeUndefined()
    expect(resetSpy).toHaveBeenCalledWith({
      queryKey: getGetCurrentUserQueryKey(),
    })
    expect(sessionStorage.getItem(CSRF_STORAGE_KEY)).toBeNull()
    expect(errorToast).toHaveBeenCalledWith(SESSION_EXPIRED_MESSAGE)
    expect(redirectToLogin).toHaveBeenCalledTimes(1)
  })
})
