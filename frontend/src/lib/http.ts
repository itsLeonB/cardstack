// Shared fetch wrapper wired in as orval's fetch-client `mutator`
// (see orval.config.ts's `override.mutator` under the `cardstack` project).
// Every generated API call goes through this, so it's the one place that
// needs to know: cookies must ride along cross-origin (frontend on Vercel,
// backend on Railway in prod; different ports in dev), and mutating
// requests need the CSRF double-submit header the backend checks against
// the (non-HttpOnly) csrf_token cookie.
const MUTATING_METHODS = new Set(["POST", "PUT", "PATCH", "DELETE"])

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
    const csrfToken = readCookie("csrf_token")
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
