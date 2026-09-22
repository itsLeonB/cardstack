import { describe, expect, it, vi } from "vitest"
import { QueryClient } from "@tanstack/react-query"
import { isRedirect } from "@tanstack/react-router"
import { requireAuth } from "./route-guard"

describe("requireAuth", () => {
  it("returns the user when the session is authenticated", async () => {
    const queryClient = new QueryClient()
    vi.spyOn(queryClient, "ensureQueryData").mockResolvedValue({
      status: 200,
      data: { data: { id: "1", email: "a@b.com" } },
    })

    const result = await requireAuth({
      context: { queryClient },
      location: { href: "/account" },
    })

    expect(result).toEqual({ user: { id: "1", email: "a@b.com" } })
  })

  it("redirects to /login, preserving the attempted URL, when unauthenticated", async () => {
    const queryClient = new QueryClient()
    vi.spyOn(queryClient, "ensureQueryData").mockResolvedValue({
      status: 401,
      data: { detail: "Unauthorized" },
    })

    try {
      await requireAuth({
        context: { queryClient },
        location: { href: "/account" },
      })
      expect.unreachable("requireAuth should have thrown a redirect")
    } catch (err) {
      if (!isRedirect(err)) throw err
      expect(err.options).toMatchObject({
        to: "/login",
        search: { redirect: "/account" },
      })
    }
  })
})
