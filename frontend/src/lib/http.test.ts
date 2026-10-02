import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"
import { customFetch, setCsrfToken, setOnAuthLost } from "./http"

// Regression coverage for the cross-origin CSRF fix: document.cookie can't
// read a csrf_token cookie scoped to a different site (Vercel frontend,
// Railway backend), so the in-memory token set from the login/refresh
// response body must be what customFetch actually sends.
describe("customFetch CSRF header", () => {
  afterEach(() => {
    setCsrfToken(null)
    vi.unstubAllGlobals()
  })

  it("sends X-CSRF-Token from the in-memory token on a mutating request", async () => {
    setCsrfToken("in-memory-token")
    const fetchMock = vi
      .fn()
      .mockResolvedValue(new Response("{}", { status: 200 }))
    vi.stubGlobal("fetch", fetchMock)

    await customFetch("https://api.example.com/auth/logout", { method: "POST" })

    // SAFETY: fetchMock is called exactly once per customFetch call above,
    // with (url, init) — asserted implicitly by indexing call 0.
    const [, init] = fetchMock.mock.calls[0] as [string, RequestInit]
    const headers = new Headers(init.headers)
    expect(headers.get("X-CSRF-Token")).toBe("in-memory-token")
  })

  it("does not set X-CSRF-Token on a GET request", async () => {
    setCsrfToken("in-memory-token")
    const fetchMock = vi
      .fn()
      .mockResolvedValue(new Response("{}", { status: 200 }))
    vi.stubGlobal("fetch", fetchMock)

    await customFetch("https://api.example.com/auth/me", { method: "GET" })

    // SAFETY: fetchMock is called exactly once per customFetch call above,
    // with (url, init) — asserted implicitly by indexing call 0.
    const [, init] = fetchMock.mock.calls[0] as [string, RequestInit]
    const headers = new Headers(init.headers)
    expect(headers.get("X-CSRF-Token")).toBeNull()
  })

  it("recovers the CSRF token from sessionStorage after a page reload", async () => {
    setCsrfToken("stored-token")

    // vi.resetModules + a fresh dynamic import simulates a page reload:
    // module-level state (inMemoryCsrfToken) resets, but sessionStorage
    // (a jsdom global, not module-scoped) survives — the one difference
    // that matters for this regression.
    vi.resetModules()
    const { customFetch: freshCustomFetch } = await import("./http")

    const fetchMock = vi
      .fn()
      .mockResolvedValue(new Response("{}", { status: 200 }))
    vi.stubGlobal("fetch", fetchMock)

    await freshCustomFetch("https://api.example.com/auth/logout", {
      method: "POST",
    })

    // SAFETY: fetchMock is called exactly once per customFetch call above,
    // with (url, init) — asserted implicitly by indexing call 0.
    const [, init] = fetchMock.mock.calls[0] as [string, RequestInit]
    const headers = new Headers(init.headers)
    expect(headers.get("X-CSRF-Token")).toBe("stored-token")

    sessionStorage.clear()
  })
})

const API = "https://api.example.com"
const REFRESH_URL = `${API}/auth/refresh`
const COLLECTIONS_URL = `${API}/collections`

/** Response shape the mutator resolves to for every status; only `status` is read here. */
interface ApiResult {
  status: number
}

// The second element is optional in `vi.fn`'s recorded calls, since
// customFetch's options argument is optional at the call site.
type FetchCall = [url: string, init?: RequestInit | undefined]

function jsonResponse(body: { data: object }, status = 200): Response {
  return new Response(JSON.stringify(body), { status })
}

function unauthorized(): Response {
  return new Response(null, { status: 401 })
}

function refreshSucceeds(csrfToken: string): Response {
  return jsonResponse({ data: { csrfToken, message: "ok" } })
}

function stubFetch(respond: (url: string, method: string) => Response) {
  const mock = vi.fn(async (url: string, init?: RequestInit) =>
    respond(url, (init?.method ?? "GET").toUpperCase())
  )
  vi.stubGlobal("fetch", mock)
  return mock
}

function callsTo(mock: ReturnType<typeof stubFetch>, url: string): FetchCall[] {
  return mock.mock.calls.filter(([called]) => called === url)
}

function csrfHeaderOf(call: FetchCall): string | null {
  return new Headers(call[1]?.headers).get("X-CSRF-Token")
}

// The backend rotates sessions on every /auth/refresh and the frontend used
// to ignore 401s entirely, so an expired 15-minute access token meant a
// silently dead session. These cover the retry flow that replaces it.
describe("customFetch refresh on 401", () => {
  beforeEach(() => {
    // Setting a token is also what re-arms the once-per-expiry notice, the
    // same way a login or a successful refresh does in the app.
    setCsrfToken("csrf-before")
  })

  afterEach(() => {
    setOnAuthLost(null)
    setCsrfToken(null)
    vi.unstubAllGlobals()
  })

  it("refreshes once and retries the original request with the rotated token", async () => {
    let attempts = 0
    const fetchMock = stubFetch((url) => {
      if (url === REFRESH_URL) return refreshSucceeds("rotated-token")
      attempts += 1
      return attempts === 1 ? unauthorized() : jsonResponse({ data: [] })
    })

    const response = await customFetch<ApiResult>(COLLECTIONS_URL, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
    })

    expect(response.status).toBe(200)
    const refreshes = callsTo(fetchMock, REFRESH_URL)
    expect(refreshes).toHaveLength(1)
    expect(refreshes[0]?.[1]?.method).toBe("POST")
    // Refreshing is itself a mutating call, so it needs the double-submit header.
    expect(csrfHeaderOf(refreshes[0]!)).toBe("csrf-before")

    const attemptsOnTarget = callsTo(fetchMock, COLLECTIONS_URL)
    expect(attemptsOnTarget).toHaveLength(2)
    expect(csrfHeaderOf(attemptsOnTarget[0]!)).toBe("csrf-before")
    expect(csrfHeaderOf(attemptsOnTarget[1]!)).toBe("rotated-token")
  })

  it("keeps the rotated token for later mutating requests", async () => {
    const fetchMock = stubFetch((url) =>
      url === REFRESH_URL ? refreshSucceeds("rotated-token") : unauthorized()
    )

    await customFetch(COLLECTIONS_URL, { method: "GET" })
    await customFetch(COLLECTIONS_URL, { method: "POST" })

    // The second request 401s too, refreshes again and is retried; whatever
    // the sequence, every mutating attempt after the first refresh must
    // carry the token the refresh handed back.
    const latest = callsTo(fetchMock, COLLECTIONS_URL).at(-1)
    expect(csrfHeaderOf(latest!)).toBe("rotated-token")
  })

  it("shares one refresh between concurrent 401s", async () => {
    let attempts = 0
    const fetchMock = stubFetch((url) => {
      if (url === REFRESH_URL) return refreshSucceeds("rotated-token")
      attempts += 1
      return attempts <= 2 ? unauthorized() : jsonResponse({ data: [] })
    })

    const [first, second] = await Promise.all([
      customFetch<ApiResult>(COLLECTIONS_URL, { method: "GET" }),
      customFetch<ApiResult>(COLLECTIONS_URL, { method: "GET" }),
    ])

    expect(first.status).toBe(200)
    expect(second.status).toBe(200)
    expect(callsTo(fetchMock, REFRESH_URL)).toHaveLength(1)
    expect(callsTo(fetchMock, COLLECTIONS_URL)).toHaveLength(4)
  })

  it.each(["/auth/login", "/auth/register", "/auth/logout", "/auth/refresh"])(
    "never refreshes or retries %s",
    async (path) => {
      const onAuthLost = vi.fn()
      setOnAuthLost(onAuthLost)
      const fetchMock = stubFetch(() => unauthorized())

      const response = await customFetch<ApiResult>(`${API}${path}`, {
        method: "POST",
      })

      expect(response.status).toBe(401)
      expect(fetchMock).toHaveBeenCalledTimes(1)
      expect(onAuthLost).not.toHaveBeenCalled()
    }
  )

  it("retries exactly once, returning the retry's 401", async () => {
    const fetchMock = stubFetch((url) =>
      url === REFRESH_URL ? refreshSucceeds("rotated-token") : unauthorized()
    )

    const response = await customFetch<ApiResult>(COLLECTIONS_URL, {
      method: "GET",
    })

    expect(response.status).toBe(401)
    expect(callsTo(fetchMock, REFRESH_URL)).toHaveLength(1)
    expect(callsTo(fetchMock, COLLECTIONS_URL)).toHaveLength(2)
  })

  it("returns the original 401 and reports a lost session when the refresh fails", async () => {
    const onAuthLost = vi.fn()
    setOnAuthLost(onAuthLost)
    const fetchMock = stubFetch(() => unauthorized())

    const response = await customFetch<ApiResult>(COLLECTIONS_URL, {
      method: "GET",
    })

    expect(response.status).toBe(401)
    expect(callsTo(fetchMock, COLLECTIONS_URL)).toHaveLength(1)
    expect(onAuthLost).toHaveBeenCalledTimes(1)
  })

  it("reports a lost session when the refresh request itself fails", async () => {
    const onAuthLost = vi.fn()
    setOnAuthLost(onAuthLost)
    stubFetch((url) => {
      if (url === REFRESH_URL) throw new TypeError("Failed to fetch")
      return unauthorized()
    })

    const response = await customFetch<ApiResult>(COLLECTIONS_URL, {
      method: "GET",
    })

    expect(response.status).toBe(401)
    expect(onAuthLost).toHaveBeenCalledTimes(1)
  })

  it("reports a lost session once for concurrent failed refreshes", async () => {
    const onAuthLost = vi.fn()
    setOnAuthLost(onAuthLost)
    const fetchMock = stubFetch(() => unauthorized())

    await Promise.all([
      customFetch(COLLECTIONS_URL, { method: "GET" }),
      customFetch(`${API}/inventory/cards`, { method: "GET" }),
      customFetch(COLLECTIONS_URL, { method: "GET" }),
    ])

    expect(callsTo(fetchMock, REFRESH_URL)).toHaveLength(1)
    expect(onAuthLost).toHaveBeenCalledTimes(1)
  })

  it("reports the next expiry after a new session is established", async () => {
    const onAuthLost = vi.fn()
    setOnAuthLost(onAuthLost)
    stubFetch(() => unauthorized())

    await customFetch(COLLECTIONS_URL, { method: "GET" })
    await customFetch(COLLECTIONS_URL, { method: "GET" })
    expect(onAuthLost).toHaveBeenCalledTimes(1)

    // Logging back in issues a fresh CSRF token, which re-arms the notice.
    setCsrfToken("new-session")
    await customFetch(COLLECTIONS_URL, { method: "GET" })

    expect(onAuthLost).toHaveBeenCalledTimes(2)
  })

  it("refreshes /auth/me so an expired access token still reads as signed in", async () => {
    let attempts = 0
    const fetchMock = stubFetch((url) => {
      if (url === REFRESH_URL) return refreshSucceeds("rotated-token")
      attempts += 1
      return attempts === 1
        ? unauthorized()
        : jsonResponse({ data: { id: "u1" } })
    })

    const response = await customFetch<ApiResult>(`${API}/auth/me`, {
      method: "GET",
    })

    expect(response.status).toBe(200)
    expect(callsTo(fetchMock, REFRESH_URL)).toHaveLength(1)
  })

  it("stays silent when /auth/me's refresh fails", async () => {
    const onAuthLost = vi.fn()
    setOnAuthLost(onAuthLost)
    stubFetch(() => unauthorized())

    // A visitor who was never logged in: the route guard owns this 401.
    const response = await customFetch<ApiResult>(`${API}/auth/me`, {
      method: "GET",
    })

    expect(response.status).toBe(401)
    expect(onAuthLost).not.toHaveBeenCalled()
  })

  it("passes a successful response straight through", async () => {
    const fetchMock = stubFetch(() => jsonResponse({ data: [] }))

    const response = await customFetch<ApiResult>(COLLECTIONS_URL, {
      method: "GET",
    })

    expect(response.status).toBe(200)
    expect(fetchMock).toHaveBeenCalledTimes(1)
  })

  it("does not refresh during a server-side render, which has no cookie jar", async () => {
    const savedDocument = globalThis.document
    const fetchMock = stubFetch(() => unauthorized())

    try {
      Reflect.deleteProperty(globalThis, "document")
      const response = await customFetch<ApiResult>(COLLECTIONS_URL, {
        method: "GET",
      })
      expect(response.status).toBe(401)
    } finally {
      Reflect.set(globalThis, "document", savedDocument)
    }

    expect(fetchMock).toHaveBeenCalledTimes(1)
  })

  it("ignores an auth-lost registration made outside the browser realm", async () => {
    const savedDocument = globalThis.document
    const onAuthLost = vi.fn()

    try {
      Reflect.deleteProperty(globalThis, "document")
      setOnAuthLost(onAuthLost)
    } finally {
      Reflect.set(globalThis, "document", savedDocument)
    }

    // The realm check gates the *registration*, not the refresh that follows:
    // a server render's per-request router must not reach this singleton.
    stubFetch(() => unauthorized())
    await customFetch<ApiResult>(COLLECTIONS_URL, { method: "GET" })

    expect(onAuthLost).not.toHaveBeenCalled()
  })
})
