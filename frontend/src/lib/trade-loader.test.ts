import { QueryClient } from "@tanstack/react-query"
import { afterEach, describe, expect, it, vi } from "vitest"

import { clerkAuth } from "@/lib/clerk-auth"
import type { ClerkLike } from "@/lib/clerk-auth"

import { loadTradeForSession } from "./trade-loader"

function session(id: string): ClerkLike {
  // SAFETY: the loader only reads session; addListener and signOut are unused.
  return {
    session: { id, getToken: async () => "jwt" },
    addListener: () => () => {},
    signOut: async () => {},
  } as ClerkLike
}

type JsonBody = { data?: { slug: string }; code?: string }

function json(body: JsonBody, status: number) {
  return new Response(JSON.stringify(body), {
    status,
    headers: { "content-type": "application/json" },
  })
}

afterEach(() => {
  clerkAuth.onSessionChange(null)
  clerkAuth.publish(null)
  vi.unstubAllGlobals()
})

describe("loadTradeForSession", () => {
  it("returns the Trade and the session it was read in", async () => {
    clerkAuth.publish(session("sess_1"))
    vi.stubGlobal(
      "fetch",
      vi.fn(async () => json({ data: { slug: "t1" } }, 200))
    )

    const result = await loadTradeForSession(new QueryClient(), "t1")

    expect(result.trade).toMatchObject({ slug: "t1" })
    expect(result.sessionVersion).toBe(clerkAuth.currentSessionVersion())
  })

  it("reports a missing Trade", async () => {
    clerkAuth.publish(session("sess_1"))
    vi.stubGlobal(
      "fetch",
      vi.fn(async () => json({ code: "trade_not_found" }, 404))
    )

    const result = await loadTradeForSession(new QueryClient(), "t1")

    expect(result.trade).toBe("missing")
  })

  it("reads again when the session changed during the request", async () => {
    clerkAuth.publish(session("sess_1"))
    let gets = 0
    vi.stubGlobal(
      "fetch",
      vi.fn(async () => {
        gets += 1
        if (gets === 1) clerkAuth.publish(session("sess_2"))
        return gets === 1
          ? json({ data: { slug: "old" } }, 200)
          : json({ code: "trade_not_found" }, 404)
      })
    )

    const result = await loadTradeForSession(new QueryClient(), "t1")

    expect(gets).toBe(2)
    expect(result.trade).toBe("missing")
  })

  it("returns no Trade when every read crosses a session change", async () => {
    clerkAuth.publish(session("sess_1"))
    let gets = 0
    vi.stubGlobal(
      "fetch",
      vi.fn(async () => {
        gets += 1
        clerkAuth.publish(session(`sess_${gets + 1}`))
        return json({ data: { slug: "t1" } }, 200)
      })
    )

    const result = await loadTradeForSession(new QueryClient(), "t1")

    expect(gets).toBe(2)
    expect(result.trade).toBeNull()
  })
})
