import { afterEach, describe, expect, it, vi } from "vitest"
import { QueryClient } from "@tanstack/react-query"
import { toast } from "sonner"
import { SESSION_EXPIRED_MESSAGE, createAuthLostHandler } from "./auth-lost"

const COLLECTIONS_KEY = ["/collections"]

describe("createAuthLostHandler", () => {
  afterEach(() => {
    vi.restoreAllMocks()
  })

  it("drops the cached user data, toasts and sends the user to login", () => {
    const queryClient = new QueryClient()
    queryClient.setQueryData(COLLECTIONS_KEY, { status: 200 })
    const errorToast = vi.spyOn(toast, "error")
    const redirectToLogin = vi.fn(() => {
      // The stale data must be gone before the login page renders.
      expect(queryClient.getQueryData(COLLECTIONS_KEY)).toBeUndefined()
    })

    createAuthLostHandler(queryClient, redirectToLogin)()

    expect(queryClient.getQueryData(COLLECTIONS_KEY)).toBeUndefined()
    expect(errorToast).toHaveBeenCalledWith(SESSION_EXPIRED_MESSAGE)
    expect(redirectToLogin).toHaveBeenCalledTimes(1)
  })
})
