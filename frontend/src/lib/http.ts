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

// The csrf_token cookie is the value the backend's CSRFGuard compares the
// header against, so whenever this page can read it, it is the only
// trustworthy source: every login and every refresh mints a *new* token, and
// a rotation in one tab leaves the copies held by the others stale — sending
// one of those is a 403 a reload can't clear, because the reload re-reads
// the same stale copy. Cross-origin (Vercel frontend, Railway backend) the
// cookie is scoped to a different site, so document.cookie can't see it at
// all: the browser still sends it TO the backend, but this page's JS has no
// read access. There the login/refresh response body (which echoes
// csrfToken, see AuthHandler.cookieResponse) is the only source, which is
// why session.ts hands it to setCsrfToken().
//
// sessionStorage persists those body-issued tokens: the in-memory value alone
// is lost on a page reload or a new tab, and in the cross-origin deployment
// (where the cookie can't be read at all) losing it would silently break
// logout and refresh for the rest of that session.
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
  // A server render builds a fresh router and queryClient per request, so a
  // registration made there would leave this process-wide singleton holding
  // one request's state for the next. Only the browser realm may register.
  onAuthLost = canRefreshSession() ? callback : null
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
let inFlightRefresh: Promise<RefreshOutcome> | null = null

// `expired` is the backend saying the refresh token itself is gone. Anything
// else means the refresh *request* never got an answer, which is not the same
// thing as a dead session.
type RefreshOutcome = "refreshed" | "expired" | "unavailable"

function refreshAccessToken(requestUrl: string): Promise<RefreshOutcome> {
  inFlightRefresh ??= performRefresh(requestUrl).finally(() => {
    inFlightRefresh = null
  })
  return inFlightRefresh
}

async function performRefresh(requestUrl: string): Promise<RefreshOutcome> {
  const headers = new Headers()
  applyCsrfToken(headers, "POST")

  const res = await fetch(refreshUrlFor(requestUrl), {
    method: "POST",
    headers,
    credentials: "include",
  })

  // Only a 401 says the refresh token is dead. A 403 from the CSRF guard, a
  // 5xx during a deploy, or a network/CORS failure leaves a live session
  // untouched: announcing "session expired" there would clear the cache and
  // the CSRF token and bounce a signed-in user to login.
  if (res.status === 401) return "expired"
  if (res.status !== 200) return "unavailable"

  const body = await res.text()
  // The csrf_token cookie sits on the backend's origin, so in the
  // cross-origin deployment this body is the only place the rotated token
  // can be read from — and the retry in customFetch needs it (see
  // setCsrfToken).
  // SAFETY: a 200 from /auth/refresh is the backend's EnvelopeAuthMessage
  // (the same envelope a successful `login` returns).
  const { data } = (body ? JSON.parse(body) : {}) as EnvelopeAuthMessage
  setCsrfToken(data.csrfToken ?? null)
  return "refreshed"
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

  // Cookie first: it is the value the backend compares against. The stored
  // copies are the fallback for cross-origin, where it cannot be read at all.
  const csrfToken =
    readCookie("csrf_token") ?? inMemoryCsrfToken ?? storedCsrfToken()
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

  // `unavailable` (network/CORS failure, 5xx, or the CSRF guard's 403) is
  // not a lost session: the caller gets the 401 it would have gotten without
  // this wrapper, and nothing is cleared or announced. Only `expired` — the
  // backend's own 401 on the refresh — means the refresh token is gone.
  const outcome = await refreshAccessToken(url).catch(
    () => "unavailable" as const
  )

  if (outcome !== "refreshed") {
    if (outcome === "expired" && pathname !== SESSION_PATH) notifyAuthLost()
    return asResponse<T>(response)
  }

  // Exactly one retry, with the rotated cookie; a second 401 is returned
  // as-is rather than retried again.
  return asResponse<T>(await sendRequest(url, method, options))
}

export default customFetch
