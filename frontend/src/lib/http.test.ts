import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"
import type { TokenGetter } from "./http"
import {
  customFetch,
  rearmAuthLost,
  setOnAuthLost,
  setTokenGetter,
} from "./http"

const URL = "https://api.example.com/collections"

function respond(status: number) {
  return new Response("{}", { status })
}

/** What the token getter resolves for a signed-in user. */
function tok(token: string, sessionId = "sess_1") {
  return { token, sessionId }
}

function fetchStub(status: number) {
  return vi.fn(async (_url: string, _init?: RequestInit) => respond(status))
}

/** The Authorization header each fetch call carried, in call order. */
function sentAuthorization(fetchMock: ReturnType<typeof fetchStub>) {
  return fetchMock.mock.calls.map(([, init]) =>
    new Headers(init?.headers).get("Authorization")
  )
}

beforeEach(() => {
  // A new session re-arms the one-notice-per-expiry dedupe between tests.
  rearmAuthLost()
})

afterEach(() => {
  setTokenGetter(null)
  setOnAuthLost(null)
  vi.unstubAllGlobals()
})

describe("customFetch bearer token", () => {
  it("sends the current Clerk token as a bearer header", async () => {
    const fetchMock = fetchStub(200)
    vi.stubGlobal("fetch", fetchMock)
    setTokenGetter(async () => tok("token-1"))

    await customFetch(URL, { method: "GET" })

    expect(sentAuthorization(fetchMock)).toEqual(["Bearer token-1"])
  })

  it("sends no Authorization header for a guest", async () => {
    const fetchMock = fetchStub(200)
    vi.stubGlobal("fetch", fetchMock)
    setTokenGetter(async () => null)

    await customFetch(URL, { method: "GET" })

    expect(sentAuthorization(fetchMock)).toEqual([null])
  })

  it("sends no Authorization header before the app registered a token getter", async () => {
    const fetchMock = fetchStub(200)
    vi.stubGlobal("fetch", fetchMock)

    await customFetch(URL, { method: "GET" })

    expect(sentAuthorization(fetchMock)).toEqual([null])
  })

  it("falls back to a guest request when Clerk cannot answer", async () => {
    const fetchMock = fetchStub(200)
    vi.stubGlobal("fetch", fetchMock)
    setTokenGetter(async () => {
      throw new Error("Timeout waiting for Clerk to load.")
    })

    const response = await customFetch<{ status: number }>(URL, {
      method: "GET",
    })

    expect(response.status).toBe(200)
    expect(sentAuthorization(fetchMock)).toEqual([null])
  })

  it("sends no cookies and no CSRF header", async () => {
    const fetchMock = fetchStub(200)
    vi.stubGlobal("fetch", fetchMock)
    setTokenGetter(async () => tok("token-1"))

    await customFetch(URL, { method: "POST", body: "{}" })

    const init = fetchMock.mock.calls[0]?.[1]
    expect(init?.credentials).toBeUndefined()
    expect(new Headers(init?.headers).get("X-CSRF-Token")).toBeNull()
  })

  it("passes the caller's headers and body through", async () => {
    const fetchMock = fetchStub(200)
    vi.stubGlobal("fetch", fetchMock)
    setTokenGetter(async () => tok("token-1"))

    await customFetch(URL, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: '{"a":1}',
    })

    const init = fetchMock.mock.calls[0]?.[1]
    expect(new Headers(init?.headers).get("Content-Type")).toBe(
      "application/json"
    )
    expect(init?.body).toBe('{"a":1}')
  })
})

describe("customFetch retry on 401", () => {
  it("asks Clerk for a fresh token (skipping its cache) and retries once", async () => {
    const fetchMock = vi
      .fn<() => Promise<Response>>()
      .mockResolvedValueOnce(respond(401))
      .mockResolvedValueOnce(respond(200))
    vi.stubGlobal("fetch", fetchMock)
    const getter = vi
      .fn<TokenGetter>()
      .mockResolvedValueOnce(tok("stale"))
      .mockResolvedValueOnce(tok("fresh"))
    setTokenGetter(getter)
    const onLost = vi.fn()
    setOnAuthLost(onLost)

    const response = await customFetch<{ status: number }>(URL, {
      method: "GET",
    })

    expect(response.status).toBe(200)
    expect(getter.mock.calls).toEqual([[], [{ skipCache: true }]])
    expect(sentAuthorization(fetchMock)).toEqual([
      "Bearer stale",
      "Bearer fresh",
    ])
    expect(onLost).not.toHaveBeenCalled()
  })

  it("reports a lost session when the retry is 401 too, and does not retry again", async () => {
    const fetchMock = fetchStub(401)
    vi.stubGlobal("fetch", fetchMock)
    setTokenGetter(async (options) =>
      tok(options?.skipCache ? "fresh" : "stale")
    )
    const onLost = vi.fn()
    setOnAuthLost(onLost)

    const response = await customFetch<{ status: number }>(URL, {
      method: "GET",
    })

    expect(response.status).toBe(401)
    expect(fetchMock).toHaveBeenCalledTimes(2)
    expect(onLost).toHaveBeenCalledTimes(1)
  })

  it("reports a lost session, without retrying, when Clerk has no session left", async () => {
    const fetchMock = fetchStub(401)
    vi.stubGlobal("fetch", fetchMock)
    setTokenGetter(async (options) =>
      options?.skipCache ? null : tok("stale")
    )
    const onLost = vi.fn()
    setOnAuthLost(onLost)

    const response = await customFetch<{ status: number }>(URL, {
      method: "GET",
    })

    expect(response.status).toBe(401)
    expect(fetchMock).toHaveBeenCalledTimes(1)
    expect(onLost).toHaveBeenCalledTimes(1)
  })

  it("stays silent when Clerk cannot be reached for a fresh token", async () => {
    const fetchMock = fetchStub(401)
    vi.stubGlobal("fetch", fetchMock)
    setTokenGetter(async (options) => {
      if (options?.skipCache) throw new Error("offline")
      return tok("stale")
    })
    const onLost = vi.fn()
    setOnAuthLost(onLost)

    const response = await customFetch<{ status: number }>(URL, {
      method: "GET",
    })

    expect(response.status).toBe(401)
    expect(fetchMock).toHaveBeenCalledTimes(1)
    expect(onLost).not.toHaveBeenCalled()
  })

  it("shares one fresh-token request between concurrent 401s", async () => {
    const fetchMock = vi.fn(async (_url: string, init?: RequestInit) =>
      respond(
        new Headers(init?.headers).get("Authorization") === "Bearer fresh"
          ? 200
          : 401
      )
    )
    vi.stubGlobal("fetch", fetchMock)
    const getter = vi.fn(async (options?: { skipCache?: boolean }) =>
      tok(options?.skipCache ? "fresh" : "stale")
    )
    setTokenGetter(getter)

    const responses = await Promise.all([
      customFetch<{ status: number }>(URL, { method: "GET" }),
      customFetch<{ status: number }>(URL, { method: "GET" }),
      customFetch<{ status: number }>(URL, { method: "GET" }),
    ])

    expect(responses.map((r) => r.status)).toEqual([200, 200, 200])
    expect(getter.mock.calls.filter(([o]) => o?.skipCache)).toHaveLength(1)
  })

  it("reports a lost session once for concurrent failures", async () => {
    vi.stubGlobal("fetch", fetchStub(401))
    setTokenGetter(async () => tok("token"))
    const onLost = vi.fn()
    setOnAuthLost(onLost)

    await Promise.all([
      customFetch(URL, { method: "GET" }),
      customFetch(URL, { method: "GET" }),
    ])

    expect(onLost).toHaveBeenCalledTimes(1)
  })

  it("reports the next expiry once a new session re-arms the notice", async () => {
    vi.stubGlobal("fetch", fetchStub(401))
    setTokenGetter(async () => tok("token"))
    const onLost = vi.fn()
    setOnAuthLost(onLost)

    await customFetch(URL, { method: "GET" })
    await customFetch(URL, { method: "GET" })
    expect(onLost).toHaveBeenCalledTimes(1)

    rearmAuthLost()
    await customFetch(URL, { method: "GET" })
    expect(onLost).toHaveBeenCalledTimes(2)
  })

  it("never retries or reports a lost session for a guest's 401", async () => {
    const fetchMock = fetchStub(401)
    vi.stubGlobal("fetch", fetchMock)
    const getter = vi.fn(async () => null)
    setTokenGetter(getter)
    const onLost = vi.fn()
    setOnAuthLost(onLost)

    const response = await customFetch<{ status: number }>(URL, {
      method: "GET",
    })

    expect(response.status).toBe(401)
    expect(fetchMock).toHaveBeenCalledTimes(1)
    expect(getter).toHaveBeenCalledTimes(1)
    expect(onLost).not.toHaveBeenCalled()
  })

  it("passes a non-401 error straight through", async () => {
    const fetchMock = fetchStub(500)
    vi.stubGlobal("fetch", fetchMock)
    setTokenGetter(async () => tok("token"))
    const onLost = vi.fn()
    setOnAuthLost(onLost)

    const response = await customFetch<{ status: number }>(URL, {
      method: "GET",
    })

    expect(response.status).toBe(500)
    expect(fetchMock).toHaveBeenCalledTimes(1)
    expect(onLost).not.toHaveBeenCalled()
  })
})

describe("customFetch when the session changes under a request", () => {
  // Session A's request is in flight when the user ends up in session B.
  function switchableSession() {
    let current = "sess_A"
    // Tokens are minted for whoever is signed in at the time, as Clerk does.
    setTokenGetter(
      async (options) =>
        tok(`${options?.skipCache ? "fresh" : "token"}-${current}`, current),
      () => current
    )
    return {
      switchTo(id: string) {
        current = id
      },
    }
  }

  it("does not retry A's request as B, and does not announce anything", async () => {
    const fetchMock = fetchStub(401)
    vi.stubGlobal("fetch", fetchMock)
    const session = switchableSession()
    const onLost = vi.fn()
    setOnAuthLost(onLost)
    fetchMock.mockImplementationOnce(async () => {
      session.switchTo("sess_B")
      return respond(401)
    })

    const response = await customFetch<{ status: number }>(URL, {
      method: "GET",
    })

    expect(response.status).toBe(401)
    expect(fetchMock).toHaveBeenCalledTimes(1)
    expect(onLost).not.toHaveBeenCalled()
  })

  it("ignores A's second 401 when B took over during the retry, leaving the notice unspent", async () => {
    const fetchMock = fetchStub(401)
    vi.stubGlobal("fetch", fetchMock)
    const session = switchableSession()
    const onLost = vi.fn()
    setOnAuthLost(onLost)
    fetchMock
      .mockImplementationOnce(async () => respond(401))
      .mockImplementationOnce(async () => {
        session.switchTo("sess_B")
        return respond(401)
      })

    await customFetch(URL, { method: "GET" })

    expect(sentAuthorization(fetchMock)).toEqual([
      "Bearer token-sess_A",
      "Bearer fresh-sess_A",
    ])
    expect(onLost).not.toHaveBeenCalled()

    // B's own refusal is still announced: A's did not spend the notice.
    await customFetch(URL, { method: "GET" })
    expect(onLost).toHaveBeenCalledTimes(1)
  })

  it("still announces a session that ended with nobody signed in after it", async () => {
    vi.stubGlobal("fetch", fetchStub(401))
    setTokenGetter(
      async (options) => (options?.skipCache ? null : tok("token-A")),
      () => null
    )
    const onLost = vi.fn()
    setOnAuthLost(onLost)

    await customFetch(URL, { method: "GET" })

    expect(onLost).toHaveBeenCalledTimes(1)
  })
})

describe("customFetch outside the browser realm", () => {
  it("ignores token-getter and auth-lost registrations made during a server render", async () => {
    const savedDocument = globalThis.document
    const getter = vi.fn(async () => tok("token"))
    const onLost = vi.fn()
    try {
      Reflect.deleteProperty(globalThis, "document")
      setTokenGetter(getter)
      setOnAuthLost(onLost)
    } finally {
      Reflect.set(globalThis, "document", savedDocument)
    }
    // The realm check gates the registration, not the request that follows:
    // a server render's per-request router must not reach this singleton.
    const fetchMock = fetchStub(401)
    vi.stubGlobal("fetch", fetchMock)

    await customFetch(URL, { method: "GET" })

    expect(getter).not.toHaveBeenCalled()
    expect(onLost).not.toHaveBeenCalled()
    expect(sentAuthorization(fetchMock)).toEqual([null])
  })
})
