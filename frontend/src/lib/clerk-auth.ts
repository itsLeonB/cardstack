// The two things code outside React needs from Clerk: the session token (for
// `http.ts`) and whether anyone is signed in (for the route guards, which run
// in `beforeLoad`). Both go through Clerk's standalone `getToken`, which waits
// for Clerk to finish loading (up to 10s, then throws) and works anywhere in
// the browser, so neither depends on a component having mounted yet.
import { getToken } from "@clerk/react"
import type { TokenGetter } from "./http"

export const getSessionToken: TokenGetter = (options) => getToken(options)

/** What a route guard asks about the visitor. Injected through router context. */
export interface AuthGate {
  /** Waits for Clerk to load, then says whether a session is active. */
  isSignedIn: () => Promise<boolean>
}

export const clerkAuth: AuthGate = {
  // `getToken` is null exactly when nobody is signed in. A Clerk that cannot
  // load or reach its API reads as signed out: the visitor can still reach
  // the sign-in page rather than hanging on a blank route.
  isSignedIn: async () => {
    try {
      return (await getToken()) !== null
    } catch {
      return false
    }
  },
}
