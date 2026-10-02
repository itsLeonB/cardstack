import { redirect } from "@tanstack/react-router"
import { CancelledError } from "@tanstack/react-query"
import type { QueryClient } from "@tanstack/react-query"
import { getGetCurrentUserQueryOptions } from "@/generated/endpoints/auth/auth"

// In dev, React StrictMode mounts and unmounts the header's `useSession`
// observer while this probe is still in flight; with no observer left, query
// cancels the fetch. Retrying once starts a fresh fetch against the settled mount.
async function probeSession(queryClient: QueryClient) {
  const probe = () =>
    queryClient.ensureQueryData(getGetCurrentUserQueryOptions())
  try {
    return await probe()
  } catch (error) {
    if (error instanceof CancelledError) return probe()
    throw error
  }
}

interface GuardContext {
  context: { queryClient: QueryClient }
}

interface RequireAuthArgs extends GuardContext {
  location: { pathname: string; searchStr: string }
}

/**
 * True only for a path that stays on this origin: it must start with a single
 * `/`. `//host` and `/\host` are protocol-relative (browsers read `\` as `/`),
 * so an unchecked `?redirect=` could otherwise send a login off-site.
 */
export function isSameOriginPath(path: string | undefined): path is string {
  return (
    path !== undefined &&
    path.startsWith("/") &&
    !path.startsWith("//") &&
    !path.startsWith("/\\")
  )
}

/**
 * Reusable `beforeLoad` guard for protected routes. Probes `GET /auth/me`
 * through the router's queryClient (so it shares the cache with
 * `useSession()`) and redirects to `/login` when the session isn't
 * authenticated, preserving the attempted path and query string as a
 * relative `redirect` search param (`location.href` can be absolute, which
 * the login page's same-origin check rejects).
 *
 * Attach directly to a protected route's `beforeLoad`, or to a shared
 * pathless layout route (e.g. `_authenticated.tsx`) that groups several.
 */
export async function requireAuth({ context, location }: RequireAuthArgs) {
  const response = await probeSession(context.queryClient)

  if (response.status !== 200) {
    throw redirect({
      to: "/login",
      search: { redirect: location.pathname + location.searchStr },
    })
  }

  return { user: response.data.data }
}

/**
 * `beforeLoad` guard for login and register: signed-in users are sent to `/`.
 * Anything but a 200 (or a failed probe, e.g. backend unreachable) leaves the
 * forms reachable, since a guest must never be locked out of them.
 */
export async function requireGuest({ context }: GuardContext) {
  const response = await probeSession(context.queryClient).catch(() => null)

  if (response?.status === 200) {
    throw redirect({ to: "/" })
  }
}
