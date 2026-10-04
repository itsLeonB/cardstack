// The Clerk-facing surface for code outside React: the session token (for
// `http.ts`), whether anyone is signed in (for the route guards, which run in
// `beforeLoad`), ending a session, and session changes. A React component
// (`ClerkBridge`) publishes the loaded Clerk instance here; everything waits
// for it, but only up to LOAD_TIMEOUT_MS. If clerk-js never loads (a content
// blocker, being offline) that is remembered, so every later call answers
// "nobody is signed in" at once instead of stalling, and public pages keep
// working as a guest. A Clerk that loads after all is picked up: its session
// then counts as a change, because everything cached meanwhile was fetched
// as a guest.
import type { TokenOptions } from "./http"

export interface ClerkSession {
  id: string
  getToken: (options?: TokenOptions) => Promise<string | null>
}

interface SessionEmission {
  session?: { id: string } | null
}

/** The part of the loaded Clerk instance this module uses. */
export interface ClerkLike {
  session?: ClerkSession | null
  addListener: (callback: (emission: SessionEmission) => void) => () => void
  signOut: () => Promise<void>
}

/** What a route guard asks about the visitor. Injected through router context. */
export interface AuthGate {
  /** Waits for Clerk to load, then says whether a session is active. */
  isSignedIn: () => Promise<boolean>
}

export type SessionChangeHandler = (
  current: string | null,
  previous: string | null
) => void

const LOAD_TIMEOUT_MS = 5000

export function createClerkAuth(loadTimeoutMs = LOAD_TIMEOUT_MS) {
  let clerk: ClerkLike | null = null
  let loadFailed = false
  let loadedLate = false
  let waiters: Array<(loaded: ClerkLike | null) => void> = []
  let timer: ReturnType<typeof setTimeout> | undefined
  let handler: SessionChangeHandler | null = null
  let stopListening: (() => void) | null = null

  function settle(loaded: ClerkLike | null) {
    clearTimeout(timer)
    timer = undefined
    const pending = waiters
    waiters = []
    for (const resolve of pending) resolve(loaded)
  }

  // Clerk reports the current state when a listener attaches, which only
  // records the baseline; later reports that differ are the changes. They fire
  // inside Clerk's own state update, before it navigates anywhere (so a
  // cache reset cannot wipe what the next page loads).
  function listen() {
    stopListening?.()
    stopListening = null
    if (!clerk || !handler) return
    let known: string | null | undefined = loadedLate ? null : undefined
    stopListening = clerk.addListener(({ session }) => {
      const current = session?.id ?? null
      const previous = known
      known = current
      if (previous !== undefined && previous !== current) {
        handler?.(current, previous)
      }
    })
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
        loadedLate = loadFailed
        loadFailed = false
        settle(next)
      }
      listen()
    },

    /**
     * Browser-only, like `setOnAuthLost` (a server render must not leave one
     * request's state in this singleton). One handler; a new one replaces it.
     */
    onSessionChange(callback: SessionChangeHandler | null) {
      handler = "document" in globalThis ? callback : null
      listen()
    },

    /** Resolves a fresh or cached token; null when nobody is signed in. Rejects when Clerk cannot fetch one (offline). */
    async getToken(options?: TokenOptions): Promise<string | null> {
      const loaded = await whenLoaded()
      return loaded?.session ? loaded.session.getToken(options) : null
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
