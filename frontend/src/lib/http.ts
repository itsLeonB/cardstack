// Shared fetch wrapper wired in as orval's fetch-client `mutator`
// (see orval.config.ts's `override.mutator` under the `cardstack` project).
// Every generated API call goes through this, so it's the one place that
// attaches the Clerk session token as `Authorization: Bearer` (ADR-0015;
// there are no cookies, so no `credentials` and no CSRF header). A 401 on a
// request that carried a token means it expired in flight: ask Clerk for a
// fresh one (skipping its cache) and retry once, and report a lost session if
// that does not help. A guest sends no header, and a guest's 401 just means
// "sign in", so it is neither retried nor reported.

export interface TokenOptions {
  skipCache?: boolean
}

/** A Clerk token with the id of the session it was issued for. */
export interface SessionToken {
  token: string
  sessionId: string
}

/** Resolves the current Clerk session's token, or null when nobody is signed in. */
export type TokenGetter = (
  options?: TokenOptions
) => Promise<SessionToken | null>

/** The id of the session Clerk holds right now, or null when there is none. */
export type CurrentSessionId = () => string | null

interface FetchResponse {
  data: object
  status: number
  headers: Headers
}

// A server render builds a fresh router per request, so a registration made
// there would leave these process-wide singletons holding one request's state
// for the next: only the browser realm may register.
function inBrowser(): boolean {
  return "document" in globalThis
}

let getToken: TokenGetter | null = null
let currentSessionId: CurrentSessionId = () => null

/**
 * This module is orval's mutator and has no React context, so the app
 * registers its Clerk token getter here (`router.tsx`), with a way to read the
 * current session id: a request's failure is only held against the session
 * that made it (a late 401 from a session that has since been replaced must
 * not sign out the new one).
 */
export function setTokenGetter(
  getter: TokenGetter | null,
  readSessionId: CurrentSessionId = () => null
): void {
  getToken = inBrowser() ? getter : null
  currentSessionId = inBrowser() ? readSessionId : () => null
}

// True when Clerk now holds a different, signed-in session than the one a
// request was made with. No session at all is not "replaced": that session
// ended, which is the case worth reporting.
function replacedBy(sessionId: string): boolean {
  const now = currentSessionId()
  return now !== null && now !== sessionId
}

let onAuthLost: (() => void) | null = null

/**
 * Registers what the app should do when a request that carried a token still
 * gets a 401 after a fresh one. The app owns that wiring (`router.tsx`)
 * because this module has no queryClient or router.
 */
export function setOnAuthLost(callback: (() => void) | null): void {
  onAuthLost = inBrowser() ? callback : null
}

// One expiry, one notice: a burst of concurrent 401s all fail the same way,
// and the user should be toasted and redirected once, not per request.
// `rearmAuthLost` re-opens this once a new session exists.
let authLostNotified = false

export function rearmAuthLost(): void {
  authLostNotified = false
}

/**
 * Announces a lost session once per expiry. Used here for a request that was
 * still refused after a fresh token, and by the app when Clerk itself drops
 * the session under a mounted private page.
 */
export function reportAuthLost(): void {
  if (authLostNotified) return
  authLostNotified = true
  onAuthLost?.()
}

// A token the caller cannot get (Clerk blocked, timed out) must not break
// public pages: the request goes out as a guest and the API decides.
async function currentToken(): Promise<SessionToken | null> {
  try {
    return (await getToken?.()) ?? null
  } catch {
    return null
  }
}

// Concurrent 401s after one expiry all want the same new token, so they share
// a single request. `undefined` means Clerk could not answer (offline, timeout),
// which is not the same as Clerk saying there is no session (`null`).
let inFlightFresh: Promise<SessionToken | null | undefined> | null = null

function freshToken(): Promise<SessionToken | null | undefined> {
  inFlightFresh ??= Promise.resolve(getToken?.({ skipCache: true }))
    .then((fresh) => fresh ?? null)
    .catch(() => undefined)
    .finally(() => {
      inFlightFresh = null
    })
  return inFlightFresh
}

async function sendRequest(
  url: string,
  method: string,
  options: RequestInit,
  token: string | null
): Promise<FetchResponse> {
  const headers = new Headers(options.headers)
  if (token) headers.set("Authorization", `Bearer ${token}`)

  const res = await fetch(url, { ...options, method, headers })

  const body = [204, 205, 304].includes(res.status) ? null : await res.text()
  const data: object = body ? JSON.parse(body) : {}

  return { data, status: res.status, headers: res.headers }
}

function asResponse<T>(response: FetchResponse): T {
  // SAFETY: `T` is always one of orval's generated `<operation>Response`
  // unions, which are shaped exactly `{ data, status, headers }` for every
  // status code declared in the OpenAPI spec — matching orval's own default
  // fetch-client response shape (see the generated `getHealth`/etc. in
  // src/generated/endpoints), just with the bearer token added.
  return response as T
}

export const customFetch = async <T>(
  url: string,
  options: RequestInit
): Promise<T> => {
  const method = (options.method ?? "GET").toUpperCase()
  const session = await currentToken()
  const response = await sendRequest(
    url,
    method,
    options,
    session?.token ?? null
  )

  if (response.status !== 401 || session === null) {
    return asResponse<T>(response)
  }

  const fresh = await freshToken()
  // Clerk could not answer, or the session this request was made with has been
  // replaced by another one in the meantime (neither the new token nor the
  // failure belongs to this request): the caller gets the 401 it would have
  // gotten without this wrapper, and nothing is announced.
  if (fresh === undefined || replacedBy(session.sessionId)) {
    return asResponse<T>(response)
  }
  if (fresh === null) {
    reportAuthLost()
    return asResponse<T>(response)
  }
  if (fresh.sessionId !== session.sessionId) return asResponse<T>(response)

  // Exactly one retry; a second 401 is returned as-is rather than retried.
  const retry = await sendRequest(url, method, options, fresh.token)
  if (retry.status === 401 && !replacedBy(session.sessionId)) reportAuthLost()
  return asResponse<T>(retry)
}

export default customFetch
