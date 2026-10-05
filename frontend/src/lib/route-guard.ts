import { redirect } from "@tanstack/react-router"
import { z } from "zod"
import { LOGIN_PATH, REGISTER_PATH } from "./auth-paths"
import type { AuthGate } from "./clerk-auth"

interface GuardContext {
  context: { auth: AuthGate }
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
    !path.startsWith("/\\") &&
    // URL parsers strip tab/CR/LF, so `/<TAB>/host` would become `//host`.
    !/\p{Cc}/u.test(path)
  )
}

// `.catch`: a rejected target is dropped, so login and register still work.
// `redirect` only ever needs to point back into this app (`requireAuth` sets it
// from the router's own location), so it is restricted to a same-origin path.
// Left unvalidated, a crafted `/auth/login?redirect=` link could send a
// successful sign-in to an attacker controlled destination (open redirect).
const redirectSchema = z
  .string()
  .refine(isSameOriginPath)
  .optional()
  .catch(undefined)

/** `validateSearch` for login and register. */
export const redirectSearchSchema = z.object({ redirect: redirectSchema })

/**
 * Clerk moves between its steps (`client-trust`, `factor-one`, ...) by pushing
 * a bare path, which drops the `redirect` the sign-in started with. Re-adds the
 * current one, if it is a safe same-origin path, to a step beneath login or
 * register that does not name its own.
 */
export function carryRedirect(to: string, currentSearch: string) {
  const [path = "", query = ""] = to.split("?")
  const isStep = [LOGIN_PATH, REGISTER_PATH].some((base) =>
    path.startsWith(`${base}/`)
  )
  const target = new URLSearchParams(currentSearch).get("redirect") ?? undefined
  if (
    !isStep ||
    new URLSearchParams(query).has("redirect") ||
    !isSameOriginPath(target)
  ) {
    return to
  }
  return `${to}${query ? "&" : "?"}${new URLSearchParams({ redirect: target })}`
}

/** Carries a validated `redirect` target across the login/register switch link. */
export function withRedirect(path: string, target: string | undefined) {
  return target ? `${path}?${new URLSearchParams({ redirect: target })}` : path
}

/**
 * Reusable `beforeLoad` guard for protected routes. Waits for Clerk to load
 * (through the router context's `auth`) so a reload on a protected page never
 * bounces a signed-in user, then redirects to `/auth/login` when nobody is
 * signed in, preserving the attempted path and query string as a relative
 * `redirect` search param (`location.href` can be absolute, which the login
 * page's same-origin check rejects).
 *
 * Attach directly to a protected route's `beforeLoad`, or to a shared
 * pathless layout route (e.g. `_authenticated.tsx`) that groups several.
 */
export async function requireAuth({ context, location }: RequireAuthArgs) {
  if (!(await context.auth.isSignedIn())) {
    throw redirect({
      to: LOGIN_PATH,
      search: { redirect: location.pathname + location.searchStr },
    })
  }
}

/**
 * `beforeLoad` guard for login and register: signed-in users are sent to `/`.
 * Like `requireAuth` it waits for Clerk to load; a guest is never locked out.
 */
export async function requireGuest({ context }: GuardContext) {
  if (await context.auth.isSignedIn()) {
    throw redirect({ to: "/" })
  }
}
