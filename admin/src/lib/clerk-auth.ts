// The Clerk-facing surface for code outside React: the session token (for
// `http.ts`), whether anyone is signed in (for the route guards, which run in
// `beforeLoad`), and ending a session. A React component (`ClerkBridge`)
// publishes the loaded Clerk instance here; everything waits for it, but only
// up to LOAD_TIMEOUT_MS. If clerk-js never loads (a content blocker, being
// offline) that is remembered, so every later call answers "nobody is signed
// in" at once instead of stalling. A Clerk that loads after all is picked up.
import type { SessionToken, TokenOptions } from "./http"

export interface ClerkSession {
  id: string
  getToken: (options?: TokenOptions) => Promise<string | null>
}

/** The part of the loaded Clerk instance this module uses. */
export interface ClerkLike {
  session?: ClerkSession | null
  signOut: () => Promise<void>
}

/** What a route guard asks about the visitor. Injected through router context. */
export interface AuthGate {
  /** Waits for Clerk to load, then says whether a session is active. */
  isSignedIn: () => Promise<boolean>
}

const LOAD_TIMEOUT_MS = 5000

export function createClerkAuth(loadTimeoutMs = LOAD_TIMEOUT_MS) {
  let clerk: ClerkLike | null = null
  let loadFailed = false
  let waiters: Array<(loaded: ClerkLike | null) => void> = []
  let timer: ReturnType<typeof setTimeout> | undefined

  function settle(loaded: ClerkLike | null) {
    clearTimeout(timer)
    timer = undefined
    const pending = waiters
    waiters = []
    for (const resolve of pending) resolve(loaded)
  }

  function whenLoaded(): Promise<ClerkLike | null> {
    if (clerk) return Promise.resolve(clerk)
    if (loadFailed) return Promise.resolve(null)
    return new Promise((resolve) => {
      waiters.push(resolve)
      timer ??= setTimeout(() => {
        loadFailed = true
        settle(null)
      }, loadTimeoutMs)
    })
  }

  return {
    /** Called by `ClerkBridge` with the loaded instance, or null when it goes away. */
    publish(next: ClerkLike | null) {
      clerk = next
      if (next) {
        loadFailed = false
        settle(next)
      }
    },

    /**
     * Resolves a fresh or cached token with the id of the session it was
     * issued for (read before the fetch, so it is that session's even if the
     * user switches meanwhile); null when nobody is signed in. Rejects when
     * Clerk cannot fetch one (offline).
     */
    async getToken(options?: TokenOptions): Promise<SessionToken | null> {
      const session = (await whenLoaded())?.session
      if (!session) return null
      const token = await session.getToken(options)
      return token ? { token, sessionId: session.id } : null
    },

    currentSessionId(): string | null {
      return clerk?.session?.id ?? null
    },

    // Reads the session Clerk holds, never a token: fetching one fails
    // offline, which must not make a signed-in user look signed out.
    async isSignedIn(): Promise<boolean> {
      const loaded = await whenLoaded()
      return Boolean(loaded?.session)
    },

    /** Ends the Clerk session if there is one, then calls `then` either way. */
    async endSession(then: () => void) {
      try {
        const loaded = await whenLoaded()
        if (loaded?.session) await loaded.signOut()
      } catch {
        // Clerk could not sign out; the user still has to be moved on.
      }
      then()
    },
  }
}

export const clerkAuth = createClerkAuth()
