import { afterEach, describe, expect, it, vi } from "vitest"
import { toast } from "sonner"
import { clerkAuth } from "@/lib/clerk-auth"
import type { ClerkLike } from "@/lib/clerk-auth"
import {
  customFetch,
  rearmAuthLost,
  setOnAuthLost,
  setTokenGetter,
} from "@/lib/http"
import { SESSION_EXPIRED_MESSAGE } from "@/lib/auth-lost"
import { getRouter } from "./router"

// The real router, with its real wiring of Clerk into the request wrapper, the
// guards, auth-lost and session changes. Only Clerk's instance is faked.

type Listener = Parameters<ClerkLike["addListener"]>[0]

interface FakeClerk extends ClerkLike {
  signOut: ReturnType<typeof vi.fn<() => Promise<void>>>
}

function signedInClerk() {
  const listeners = new Set<Listener>()
  const clerk: FakeClerk = {
    session: { id: "sess_1", getToken: async () => "jwt" },
    addListener: (callback) => {
      listeners.add(callback)
      callback({ session: clerk.session })
      return () => listeners.delete(callback)
    },
    signOut: vi.fn(async () => {
      drop()
    }),
  }
  function drop() {
    clerk.session = null
    for (const callback of listeners) callback({ session: null })
  }
  return { clerk, drop }
}

afterEach(() => {
  clerkAuth.publish(null)
  clerkAuth.onSessionChange(null)
  setTokenGetter(null)
  setOnAuthLost(null)
  rearmAuthLost()
  vi.restoreAllMocks()
  vi.unstubAllGlobals()
})

async function openAccountPage() {
  window.history.pushState({}, "", "/account")
  const router = getRouter()
  const fake = signedInClerk()
  clerkAuth.publish(fake.clerk)
  await router.load()
  expect(router.state.location.pathname).toBe("/account")
  return { router, ...fake }
}

describe("a lost session, wired through the real router", () => {
  it("sends the user to sign-in, remembering the page, when Clerk drops the session under a private page", async () => {
    const error = vi.spyOn(toast, "error")
    const { router, drop } = await openAccountPage()

    drop()

    await vi.waitFor(() =>
      expect(router.state.location.pathname).toBe("/auth/login")
    )
    expect(router.state.location.search).toEqual({ redirect: "/account" })
    expect(error).toHaveBeenCalledTimes(1)
    expect(error).toHaveBeenCalledWith(SESSION_EXPIRED_MESSAGE)
  })

  it("signs the user out of Clerk, then lands on sign-in, when the API refuses a valid session twice", async () => {
    const error = vi.spyOn(toast, "error")
    vi.stubGlobal(
      "fetch",
      vi.fn(async () => new Response("{}", { status: 401 }))
    )
    const { router, clerk } = await openAccountPage()

    await customFetch("https://api.example.com/collections", { method: "GET" })

    await vi.waitFor(() =>
      expect(router.state.location.pathname).toBe("/auth/login")
    )
    expect(clerk.signOut).toHaveBeenCalledTimes(1)
    // Settled for good: requireGuest did not bounce a still-signed-in user to /.
    await new Promise((resolve) => setTimeout(resolve, 50))
    expect(router.state.location.pathname).toBe("/auth/login")
    expect(router.state.location.search).toEqual({ redirect: "/account" })
    expect(error).toHaveBeenCalledTimes(1)
  })

  it("does nothing special when the session drops on a public page", async () => {
    const error = vi.spyOn(toast, "error")
    window.history.pushState({}, "", "/catalog")
    const router = getRouter()
    const fake = signedInClerk()
    clerkAuth.publish(fake.clerk)
    await router.load()

    fake.drop()

    await new Promise((resolve) => setTimeout(resolve, 50))
    expect(router.state.location.pathname).toBe("/catalog")
    expect(error).not.toHaveBeenCalled()
  })
})
