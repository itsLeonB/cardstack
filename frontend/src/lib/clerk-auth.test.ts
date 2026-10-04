import { afterEach, describe, expect, it, vi } from "vitest"
import { createClerkAuth } from "./clerk-auth"
import type { ClerkLike, ClerkSession } from "./clerk-auth"

const TIMEOUT = 5000

afterEach(() => {
  vi.useRealTimers()
})

type Listener = Parameters<ClerkLike["addListener"]>[0]

function fakeClerk(
  sessionId: string | null,
  getToken: ClerkSession["getToken"] = async () => "jwt"
) {
  const listeners = new Set<Listener>()
  const clerk = {
    session: sessionId ? { id: sessionId, getToken } : null,
    addListener: vi.fn((callback: Listener) => {
      listeners.add(callback)
      // Clerk reports its current state to a new listener.
      callback({ session: clerk.session })
      return () => listeners.delete(callback)
    }),
    signOut: vi.fn(async () => {}),
  } satisfies ClerkLike
  return {
    clerk,
    emit(next: string | null) {
      for (const callback of listeners) {
        callback({ session: next ? { id: next } : null })
      }
    },
    listenerCount: () => listeners.size,
  }
}

describe("waiting for Clerk", () => {
  it("answers once Clerk is published, from the session it holds", async () => {
    const auth = createClerkAuth()
    const signedIn = auth.isSignedIn()
    const token = auth.getToken({ skipCache: true })

    const getJwt = vi.fn(async () => "jwt")
    auth.publish(fakeClerk("sess_1", getJwt).clerk)

    await expect(signedIn).resolves.toBe(true)
    await expect(token).resolves.toBe("jwt")
    expect(getJwt).toHaveBeenCalledWith({ skipCache: true })
  })

  it("reports a guest, and no token, when Clerk has no session", async () => {
    const auth = createClerkAuth()
    auth.publish(fakeClerk(null).clerk)

    await expect(auth.isSignedIn()).resolves.toBe(false)
    await expect(auth.getToken()).resolves.toBeNull()
  })

  it("waits for a Clerk that is slow to load, within the bound", async () => {
    vi.useFakeTimers()
    const auth = createClerkAuth(TIMEOUT)
    const signedIn = auth.isSignedIn()

    await vi.advanceTimersByTimeAsync(TIMEOUT - 1)
    auth.publish(fakeClerk("sess_1").clerk)

    await expect(signedIn).resolves.toBe(true)
  })

  it("gives up on a Clerk that never loads, and answers at once from then on", async () => {
    vi.useFakeTimers()
    const auth = createClerkAuth(TIMEOUT)
    const first = auth.getToken()
    const guard = auth.isSignedIn()

    await vi.advanceTimersByTimeAsync(TIMEOUT)

    await expect(first).resolves.toBeNull()
    await expect(guard).resolves.toBe(false)
    // Remembered: no new wait, so no timer needs to run for these to settle.
    await expect(auth.getToken()).resolves.toBeNull()
    await expect(auth.isSignedIn()).resolves.toBe(false)
  })

  it("starts answering from Clerk once it loads after all", async () => {
    vi.useFakeTimers()
    const auth = createClerkAuth(TIMEOUT)
    const gaveUp = auth.isSignedIn()
    await vi.advanceTimersByTimeAsync(TIMEOUT)
    await gaveUp

    auth.publish(fakeClerk("sess_1").clerk)

    await expect(auth.isSignedIn()).resolves.toBe(true)
    await expect(auth.getToken()).resolves.toBe("jwt")
  })
})

describe("a signed-in user while Clerk cannot fetch a token (offline)", () => {
  it("is still signed in for the guards, which read the session and fetch no token", async () => {
    const auth = createClerkAuth()
    const getJwt = vi.fn(async () => {
      throw new Error("offline")
    })
    auth.publish(fakeClerk("sess_1", getJwt).clerk)

    await expect(auth.isSignedIn()).resolves.toBe(true)
    expect(getJwt).not.toHaveBeenCalled()
  })

  it("rejects the token request, so the request wrapper can tell offline from signed out", async () => {
    const auth = createClerkAuth()
    auth.publish(
      fakeClerk("sess_1", async () => {
        throw new Error("offline")
      }).clerk
    )

    await expect(auth.getToken()).rejects.toThrow("offline")
  })
})

describe("onSessionChange", () => {
  function setup(initial: string | null) {
    const auth = createClerkAuth()
    const handler = vi.fn()
    const fake = fakeClerk(initial)
    auth.onSessionChange(handler)
    auth.publish(fake.clerk)
    return { auth, handler, fake }
  }

  it("treats what Clerk reports on attaching as the baseline, not a change", () => {
    const { handler } = setup("sess_1")

    expect(handler).not.toHaveBeenCalled()
  })

  it.each([
    ["signs in", null, "sess_1"],
    ["signs out", "sess_1", null],
    ["switches account", "sess_1", "sess_2"],
  ])("reports a change when the user %s", (_name, before, after) => {
    const { handler, fake } = setup(before)

    fake.emit(after)

    expect(handler).toHaveBeenCalledExactlyOnceWith(after, before)
  })

  it("ignores a report of the same session", () => {
    const { handler, fake } = setup("sess_1")

    fake.emit("sess_1")

    expect(handler).not.toHaveBeenCalled()
  })

  it("reports a session that appears after Clerk loaded late, since the cache was filled as a guest", async () => {
    vi.useFakeTimers()
    const auth = createClerkAuth(TIMEOUT)
    const handler = vi.fn()
    auth.onSessionChange(handler)
    const gaveUp = auth.isSignedIn()
    await vi.advanceTimersByTimeAsync(TIMEOUT)
    await gaveUp

    auth.publish(fakeClerk("sess_1").clerk)

    expect(handler).toHaveBeenCalledExactlyOnceWith("sess_1", null)
  })

  it("stops listening when Clerk goes away or the handler is replaced", () => {
    const { auth, handler, fake } = setup(null)

    auth.publish(null)
    expect(fake.listenerCount()).toBe(0)

    auth.publish(fake.clerk)
    const next = vi.fn()
    auth.onSessionChange(next)
    fake.emit("sess_1")

    expect(handler).not.toHaveBeenCalled()
    expect(next).toHaveBeenCalledTimes(1)
  })

  it("ignores a handler registered outside the browser realm", () => {
    const savedDocument = globalThis.document
    const auth = createClerkAuth()
    const handler = vi.fn()
    const fake = fakeClerk(null)
    try {
      Reflect.deleteProperty(globalThis, "document")
      auth.onSessionChange(handler)
    } finally {
      Reflect.set(globalThis, "document", savedDocument)
    }
    auth.publish(fake.clerk)

    fake.emit("sess_1")

    expect(handler).not.toHaveBeenCalled()
  })
})

describe("endSession", () => {
  it("signs out of Clerk, then calls back", async () => {
    const auth = createClerkAuth()
    const fake = fakeClerk("sess_1")
    auth.publish(fake.clerk)
    const then = vi.fn()

    await auth.endSession(then)

    expect(fake.clerk.signOut).toHaveBeenCalledTimes(1)
    expect(then).toHaveBeenCalledTimes(1)
  })

  it("only calls back when there is no session to end", async () => {
    const auth = createClerkAuth()
    const fake = fakeClerk(null)
    auth.publish(fake.clerk)
    const then = vi.fn()

    await auth.endSession(then)

    expect(fake.clerk.signOut).not.toHaveBeenCalled()
    expect(then).toHaveBeenCalledTimes(1)
  })

  it("still calls back when Clerk fails to sign out", async () => {
    const auth = createClerkAuth()
    const fake = fakeClerk("sess_1")
    fake.clerk.signOut.mockRejectedValue(new Error("network"))
    auth.publish(fake.clerk)
    const then = vi.fn()

    await auth.endSession(then)

    expect(then).toHaveBeenCalledTimes(1)
  })
})
