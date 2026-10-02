import { describe, expect, it, vi } from "vitest"
import { QueryClient } from "@tanstack/react-query"
import { isRedirect } from "@tanstack/react-router"
import { isSameOriginPath, requireAuth, requireGuest } from "./route-guard"

describe("requireAuth", () => {
  it("returns the user when the session is authenticated", async () => {
    const queryClient = new QueryClient()
    vi.spyOn(queryClient, "ensureQueryData").mockResolvedValue({
      status: 200,
      data: { data: { id: "1", email: "a@b.com" } },
    })

    const result = await requireAuth({
      context: { queryClient },
      location: { pathname: "/account", searchStr: "" },
    })

    expect(result).toEqual({ user: { id: "1", email: "a@b.com" } })
  })

  it("redirects to /auth/login, preserving the attempted URL, when unauthenticated", async () => {
    const queryClient = new QueryClient()
    vi.spyOn(queryClient, "ensureQueryData").mockResolvedValue({
      status: 401,
      data: { detail: "Unauthorized" },
    })

    try {
      await requireAuth({
        context: { queryClient },
        location: { pathname: "/account", searchStr: "" },
      })
      expect.unreachable("requireAuth should have thrown a redirect")
    } catch (err) {
      if (!isRedirect(err)) throw err
      expect(err.options).toMatchObject({
        to: "/auth/login",
        search: { redirect: "/account" },
      })
    }
  })
})

describe("requireAuth redirect target", () => {
  it("keeps the query string on the relative redirect path", async () => {
    const queryClient = new QueryClient()
    vi.spyOn(queryClient, "ensureQueryData").mockResolvedValue({
      status: 401,
      data: { detail: "Unauthorized" },
    })

    try {
      await requireAuth({
        context: { queryClient },
        location: { pathname: "/collections", searchStr: "?sort=name&page=2" },
      })
      expect.unreachable("requireAuth should have thrown a redirect")
    } catch (err) {
      if (!isRedirect(err)) throw err
      expect(err.options.search).toEqual({
        redirect: "/collections?sort=name&page=2",
      })
    }
  })
})

describe("requireGuest", () => {
  function guestContext(result: "signed-in" | "guest" | "unreachable") {
    const queryClient = new QueryClient()
    const spy = vi.spyOn(queryClient, "ensureQueryData")
    if (result === "unreachable") spy.mockRejectedValue(new Error("network"))
    else if (result === "guest")
      spy.mockResolvedValue({ status: 401, data: { detail: "Unauthorized" } })
    else
      spy.mockResolvedValue({
        status: 200,
        data: { data: { id: "1", email: "a@b.com" } },
      })
    return { queryClient }
  }

  it("redirects a signed-in user to /", async () => {
    try {
      await requireGuest({ context: guestContext("signed-in") })
      expect.unreachable("requireGuest should have thrown a redirect")
    } catch (err) {
      if (!isRedirect(err)) throw err
      expect(err.options).toMatchObject({ to: "/" })
    }
  })

  it("lets a guest through", async () => {
    await expect(
      requireGuest({ context: guestContext("guest") })
    ).resolves.toBeUndefined()
  })

  it("lets a visitor through when the session probe fails", async () => {
    await expect(
      requireGuest({ context: guestContext("unreachable") })
    ).resolves.toBeUndefined()
  })
})

describe("isSameOriginPath", () => {
  it.each(["/collections", "/collections?sort=name", "/"])("accepts %s", (path) => {
    expect(isSameOriginPath(path)).toBe(true)
  })

  it.each([
    undefined,
    "",
    "collections",
    "https://evil.example/x",
    "//evil.example",
    "/\\evil.example",
  ])("rejects %s", (path) => {
    expect(isSameOriginPath(path)).toBe(false)
  })
})
