// Shared fetch wrapper wired in as orval's fetch-client `mutator`
// (see orval.config.ts's `override.mutator` under the `cardstack` project).
// Every generated API call goes through this, so it's the one place that
// needs to know: cookies must ride along cross-origin (frontend on Vercel,
// backend on Railway in prod; different ports in dev), mutating requests
// need the CSRF double-submit header the backend checks against the
// (non-HttpOnly) csrf_token cookie, and an expired access token is renewed
// transparently once instead of surfacing a 401 the caller can't act on.
import type { EnvelopeAuthMessage } from "@/generated/models"

const MUTATING_METHODS = new Set(["POST", "PUT", "PATCH", "DELETE"])

// Their own 401 is the answer the caller asked for (bad credentials, no
// session to log out of/rotate), so refreshing and retrying them would only
// mask the error — and refreshing `/auth/refresh` would recurse.
const NO_REFRESH_PATHS = new Set([
  "/auth/login",
  "/auth/register",
  "/auth/logout",
  "/auth/refresh",
])

// `GET /auth/me` *is* retried through a refresh: an expired access token
// with a live refresh cookie should still read as authenticated, and the
// route guards depend on that. Its *failed* refresh is the ordinary "this
// visitor was never logged in" case, which must stay silent — `requireAuth`
// owns that redirect and there is no lost session to announce.
const SESSION_PATH = "/auth/me"

// Built from the URL of the request that 401'd rather than from a second
// copy of VITE_API_BASE_URL: the generated client always builds absolute
// URLs from an origin-only base, so the refresh belongs on that same
// origin. Not the generated `refreshToken` client either — that module
// imports this one as its mutator, so importing it back would be a cycle.
function refreshUrlFor(url: string): string {
  return new URL("/auth/refresh", url).toString()
}

// Cross-origin (Vercel frontend, Railway backend), document.cookie can't
// see the backend-origin csrf_token cookie at all — the browser still sends
// it TO the backend automatically, but this page's JS has no read access to
// a cookie scoped to a different site. So the login/refresh response body
// (which already echoes csrfToken, see AuthHandler.cookieResponse) is the
// real source for the header; session.ts calls setCsrfToken() once it has
// that value. readCookie stays as the same-origin/local-dev fallback, where
// the cookie is readable.
//
// Also persisted to sessionStorage: the in-memory value alone is lost on a
// page reload or a new tab, and since it's the *only* source in the
// cross-origin deployment (the cookie fallback never applies there), losing
// it would silently break logout/refresh for the rest of that session.
const CSRF_STORAGE_KEY = "csrf_token"
let inMemoryCsrfToken: string | null = null

function storedCsrfToken(): string | null {
  if (!("sessionStorage" in globalThis)) return null
  try {
    return sessionStorage.getItem(CSRF_STORAGE_KEY)
  } catch {
    return null
  }
}

export function setCsrfToken(token: string | null): void {
  inMemoryCsrfToken = token
  // A token means the backend just issued a session (login or refresh), so
  // the next expiry deserves its own "session expired" notice.
  if (token) authLostNotified = false
  if (!("sessionStorage" in globalThis)) return
  try {
    if (token) sessionStorage.setItem(CSRF_STORAGE_KEY, token)
    else sessionStorage.removeItem(CSRF_STORAGE_KEY)
  } catch {
    // Private-browsing/storage-blocked: in-memory token still works for
    // the rest of this page's lifetime.
  }
}

function readCookie(name: string): string | null {
  // SSR guard: TanStack Start loaders can run this on the server, where
  // there is no `document`/cookie jar to read from.
  if (!("document" in globalThis)) return null

  const match = document.cookie
    .split("; ")
    .find((row) => row.startsWith(`${name}=`))

  return match ? decodeURIComponent(match.slice(name.length + 1)) : null
}

interface FetchResponse {
  data: object
  status: number
  headers: Headers
}

let onAuthLost: (() => void) | null = null

/**
 * Registers what the app should do when a refresh fails on a request that
 * needed the session. This module is orval's mutator and has no queryClient
 * or router, so the app owns that wiring (`router.tsx`).
 */
export function setOnAuthLost(callback: (() => void) | null): void {
  onAuthLost = callback
}

// One expiry, one notice: a burst of concurrent 401s all fail the same
// refresh, and the user should be toasted and redirected once, not per
// request. `setCsrfToken` re-arms this once a new session exists.
let authLostNotified = false

function notifyAuthLost(): void {
  if (authLostNotified) return
  authLostNotified = true
  onAuthLost?.()
}

// The backend rotates the refresh token on every call, so two parallel
// refreshes would invalidate each other's cookie: every 401 that arrives
// while one is in flight waits on the same promise.
let inFlightRefresh: Promise<boolean> | null = null

function refreshAccessToken(url: string): Promise<boolean> {
  inFlightRefresh ??= performRefresh(url).finally(() => {
    inFlightRefresh = null
  })
  return inFlightRefresh
}

async function performRefresh(url: string): Promise<boolean> {
  const headers = new Headers()
  applyCsrfToken(headers, "POST")

  const res = await fetch(refreshUrlFor(url), {
    method: "POST",
    headers,
    credentials: "include",
  })
  if (res.status !== 200) return false

  const body = await res.text()
  // The csrf_token cookie sits on the backend's origin, so in the
  // cross-origin deployment this body is the only place the rotated token
  // can be read from — and the retry in customFetch needs it (see
  // setCsrfToken).
  // SAFETY: a 200 from /auth/refresh is the backend's EnvelopeAuthMessage
  // (the same envelope a successful `login` returns).
  const { data } = (body ? JSON.parse(body) : {}) as EnvelopeAuthMessage
  setCsrfToken(data.csrfToken ?? null)
  return true
}

/**
 * The session's cookies live in the browser's jar, so only the browser can
 * renew them; a server-side render sends none and its 401 can't be fixed by
 * refreshing. Browser-only side effects (toast, navigation) also must not
 * run during a render.
 */
function canRefreshSession(): boolean {
  return "document" in globalThis
}

function applyCsrfToken(headers: Headers, method: string): void {
  if (!MUTATING_METHODS.has(method)) return

  const csrfToken =
    inMemoryCsrfToken ?? storedCsrfToken() ?? readCookie("csrf_token")
  if (csrfToken) headers.set("X-CSRF-Token", csrfToken)
}

/** The generated client always builds absolute URLs; the base only covers relative ones. */
function requestPathname(url: string): string {
  return new URL(url, "http://localhost").pathname
}

async function sendRequest(
  url: string,
  method: string,
  options: RequestInit
): Promise<FetchResponse> {
  const headers = new Headers(options.headers)
  applyCsrfToken(headers, method)

  const res = await fetch(url, {
    ...options,
    method,
    headers,
    credentials: "include",
  })

  const body = [204, 205, 304].includes(res.status) ? null : await res.text()
  const data: object = body ? JSON.parse(body) : {}

  return { data, status: res.status, headers: res.headers }
}

function asResponse<T>(response: FetchResponse): T {
  // SAFETY: `T` is always one of orval's generated `<operation>Response`
  // unions, which are shaped exactly `{ data, status, headers }` for every
  // status code declared in the OpenAPI spec — matching orval's own default
  // fetch-client response shape (see the generated `getHealth`/`login`/etc.
  // in src/generated/endpoints), just with credentials/CSRF handling added.
  return response as T
}

export const customFetch = async <T>(
  url: string,
  options: RequestInit
): Promise<T> => {
  const method = (options.method ?? "GET").toUpperCase()
  const response = await sendRequest(url, method, options)

  if (response.status !== 401) return asResponse<T>(response)

  const pathname = requestPathname(url)
  if (NO_REFRESH_PATHS.has(pathname) || !canRefreshSession()) {
    return asResponse<T>(response)
  }

  // A refresh that never completed (network/CORS failure) counts as a
  // failed one: the caller gets the 401 it would have gotten without this
  // wrapper, and a session nobody could renew is lost either way.
  if (!(await refreshAccessToken(url).catch(() => false))) {
    if (pathname !== SESSION_PATH) notifyAuthLost()
    return asResponse<T>(response)
  }

  // Exactly one retry, with the rotated cookie; a second 401 is returned
  // as-is rather than retried again.
  return asResponse<T>(await sendRequest(url, method, options))
}

export default customFetch
