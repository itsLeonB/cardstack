import { createRouter as createTanStackRouter } from "@tanstack/react-router"
import { QueryClient } from "@tanstack/react-query"
import { CrashFallback } from "@/components/layout/crash-fallback"
import { clerkAuth } from "@/lib/clerk-auth"
import { setTokenGetter } from "@/lib/http"
import { routeTree } from "./routeTree.gen"

export function getRouter() {
  const queryClient = new QueryClient()

  const router = createTanStackRouter({
    routeTree,
    context: { queryClient, auth: clerkAuth },
    defaultErrorComponent: CrashFallback,
  })

  // `http.ts` is orval's mutator, so it has no React context: hand it the
  // Clerk token getter.
  setTokenGetter(clerkAuth.getToken, clerkAuth.currentSessionId)

  return router
}

declare module "@tanstack/react-router" {
  interface Register {
    router: ReturnType<typeof getRouter>
  }
}
