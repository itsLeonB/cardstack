import { createRouter as createTanStackRouter } from "@tanstack/react-router"
import { QueryClient } from "@tanstack/react-query"
import { setupRouterSsrQueryIntegration } from "@tanstack/react-router-ssr-query"
import { CrashFallback } from "@/components/layout/crash-fallback"
import { createAuthLostHandler } from "@/lib/auth-lost"
import { setOnAuthLost } from "@/lib/http"
import { isSameOriginPath } from "@/lib/route-guard"
import { routeTree } from "./routeTree.gen"

export function getRouter() {
  const queryClient = new QueryClient()

  const router = createTanStackRouter({
    routeTree,
    context: { queryClient },

    scrollRestoration: true,
    // Restored positions jump; never animate (reduced-motion users included).
    scrollRestorationBehavior: "instant",
    defaultErrorComponent: CrashFallback,
    defaultPreload: "intent",
    defaultPreloadStaleTime: 0,
  })

  setupRouterSsrQueryIntegration({ router, queryClient })

  // `http.ts` is orval's mutator, so it owns the refresh but not the query
  // cache or the router; hand it the app-level reaction to a session that
  // could not be renewed. Same redirect shape as `requireAuth`: the current
  // path and query, only while it stays on this origin.
  setOnAuthLost(
    createAuthLostHandler(queryClient, () => {
      const { pathname, searchStr } = router.state.location
      const attempted = pathname + searchStr
      // `navigate` rather than `history.push` (the convention for string-URL
      // redirects): the typed `search` object carries the redirect param.
      void router.navigate({
        to: "/auth/login",
        search: {
          redirect: isSameOriginPath(attempted) ? attempted : undefined,
        },
      })
    })
  )

  return router
}

declare module "@tanstack/react-router" {
  interface Register {
    router: ReturnType<typeof getRouter>
  }
}
