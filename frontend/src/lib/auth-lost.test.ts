import { afterEach, describe, expect, it, vi } from "vitest"
import { QueryClient } from "@tanstack/react-query"
import { toast } from "sonner"
import { SESSION_EXPIRED_MESSAGE, createAuthLostHandler } from "./auth-lost"

const COLLECTIONS_KEY = ["/collections"]

describe("createAuthLostHandler", () => {
  afterEach(() => {
    vi.restoreAllMocks()
  })

  it("drops the cached user data, toasts, signs out of Clerk and then sends the user to login", async () => {
    const queryClient = new QueryClient()
    queryClient.setQueryData(COLLECTIONS_KEY, { status: 200 })
    const errorToast = vi.spyOn(toast, "error")
    const order: string[] = []
    let path = "/collections?q=binder"
    const endSession = vi.fn(async (then: () => void) => {
      order.push("signed out")
      // Clerk's own sign-out navigates away before the callback runs.
      path = "/"
      then()
    })
    const redirectToLogin = vi.fn((attempted: string) => {
      order.push(`redirected from ${attempted}`)
      // The stale data must be gone before the login page renders.
      expect(queryClient.getQueryData(COLLECTIONS_KEY)).toBeUndefined()
    })

    createAuthLostHandler(queryClient, {
      endSession,
      attemptedPath: () => path,
      redirectToLogin,
    })()

    await vi.waitFor(() => expect(redirectToLogin).toHaveBeenCalledTimes(1))
    expect(errorToast).toHaveBeenCalledWith(SESSION_EXPIRED_MESSAGE)
    expect(order).toEqual([
      "signed out",
      "redirected from /collections?q=binder",
    ])
  })
})
