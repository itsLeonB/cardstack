import { redirect } from "@tanstack/react-router"
import type { QueryClient } from "@tanstack/react-query"
import { getGetCurrentUserQueryOptions } from "@/generated/endpoints/auth/auth"

interface RequireAuthArgs {
  context: { queryClient: QueryClient }
  location: { href: string }
}

/**
 * Reusable `beforeLoad` guard for protected routes. Probes `GET /auth/me`
 * through the router's queryClient (so it shares the cache with
 * `useSession()`) and redirects to `/login` when the session isn't
 * authenticated, preserving the attempted URL as a `redirect` search param.
 *
 * Attach directly to a protected route's `beforeLoad`, or to a shared
 * pathless layout route (e.g. `_authenticated.tsx`) that groups several.
 */
export async function requireAuth({ context, location }: RequireAuthArgs) {
  const response = await context.queryClient.ensureQueryData(
    getGetCurrentUserQueryOptions()
  )

  if (response.status !== 200) {
    throw redirect({
      to: "/login",
      search: { redirect: location.href },
    })
  }

  return { user: response.data.data }
}
