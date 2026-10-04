import { createRouter as createTanStackRouter } from "@tanstack/react-router"
import { QueryClient } from "@tanstack/react-query"
import { setupRouterSsrQueryIntegration } from "@tanstack/react-router-ssr-query"
import { CrashFallback } from "@/components/layout/crash-fallback"
import { createAuthLostHandler } from "@/lib/auth-lost"
import { LOGIN_PATH } from "@/lib/auth-paths"
import { clerkAuth } from "@/lib/clerk-auth"
import { setOnAuthLost, setTokenGetter } from "@/lib/http"
import { isSameOriginPath } from "@/lib/route-guard"
import { createSessionChangeHandler } from "@/lib/session"
import { routeTree } from "./routeTree.gen"

export function getRouter() {
  const queryClient = new QueryClient()

  const router = createTanStackRouter({
    routeTree,
    context: { queryClient, auth: clerkAuth },

    scrollRestoration: true,
    // Restored positions jump; never animate (reduced-motion users included).
    scrollRestorationBehavior: "instant",
    defaultErrorComponent: CrashFallback,
    defaultPreload: "intent",
    defaultPreloadStaleTime: 0,
  })

  setupRouterSsrQueryIntegration({ router, queryClient })

  // `http.ts` is orval's mutator, so it has no React context: hand it the
  // Clerk token getter, and the app-level reaction to a lost session. Same
  // redirect shape as `requireAuth`: the current path and query, only while
  // it stays on this origin.
  setTokenGetter(clerkAuth.getToken, clerkAuth.currentSessionId)
  setOnAuthLost(
    createAuthLostHandler(queryClient, {
      endSession: clerkAuth.endSession,
      attemptedPath: () => {
        const { pathname, searchStr } = router.state.location
        return pathname + searchStr
      },
      redirectToLogin: (attempted) => {
        // `navigate` rather than `history.push` (the convention for string-URL
        // redirects): the typed `search` object carries the redirect param.
        void router.navigate({
          to: LOGIN_PATH,
          search: {
            redirect: isSameOriginPath(attempted) ? attempted : undefined,
          },
        })
      },
    })
  )
  clerkAuth.onSessionChange(
    createSessionChangeHandler(queryClient, () =>
      // The pathless layout every private route lives under (`_authenticated.tsx`).
      router.state.matches.some((match) =>
        match.routeId.startsWith("/_authenticated")
      )
    )
  )

  return router
}

declare module "@tanstack/react-router" {
  interface Register {
    router: ReturnType<typeof getRouter>
  }
}
