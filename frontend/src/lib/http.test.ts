import { afterEach, describe, expect, it, vi } from "vitest"
import { customFetch, setCsrfToken } from "./http"

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
    const fetchMock = vi.fn().mockResolvedValue(new Response("{}", { status: 200 }))
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
    const fetchMock = vi.fn().mockResolvedValue(new Response("{}", { status: 200 }))
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

    const fetchMock = vi.fn().mockResolvedValue(new Response("{}", { status: 200 }))
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
