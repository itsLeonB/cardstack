// Shared fetch wrapper wired in as orval's fetch-client `mutator`
// (see orval.config.ts's `override.mutator` under the `cardstack` project).
// Every generated API call goes through this, so it's the one place that
// needs to know: cookies must ride along cross-origin (frontend on Vercel,
// backend on Railway in prod; different ports in dev), and mutating
// requests need the CSRF double-submit header the backend checks against
// the (non-HttpOnly) csrf_token cookie.
const MUTATING_METHODS = new Set(["POST", "PUT", "PATCH", "DELETE"])

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

export const customFetch = async <T>(
  url: string,
  options: RequestInit
): Promise<T> => {
  const method = (options.method ?? "GET").toUpperCase()
  const headers = new Headers(options.headers)

  if (MUTATING_METHODS.has(method)) {
    const csrfToken =
      inMemoryCsrfToken ?? storedCsrfToken() ?? readCookie("csrf_token")
    if (csrfToken) headers.set("X-CSRF-Token", csrfToken)
  }

  const res = await fetch(url, {
    ...options,
    method,
    headers,
    credentials: "include",
  })

  const body = [204, 205, 304].includes(res.status) ? null : await res.text()
  const data = body ? JSON.parse(body) : {}

  // SAFETY: `T` is always one of orval's generated `<operation>Response`
  // unions, which are shaped exactly `{ data, status, headers }` for every
  // status code declared in the OpenAPI spec — matching orval's own default
  // fetch-client response shape (see the generated `getHealth`/`login`/etc.
  // in src/generated/endpoints), just with credentials/CSRF handling added.
  return { data, status: res.status, headers: res.headers } as T
}

export default customFetch
