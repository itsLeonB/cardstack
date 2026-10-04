import { describe, expect, it } from "vitest"
import { isRedirect } from "@tanstack/react-router"
import type { AuthGate } from "./clerk-auth"
import { isSameOriginPath, requireAuth, requireGuest } from "./route-guard"

function gate(signedIn: boolean): AuthGate {
  return { isSignedIn: async () => signedIn }
}

/** A gate whose answer arrives when `finishLoading` is called, as Clerk's does. */
function loadingGate() {
  let finishLoading: (signedIn: boolean) => void = () => {}
  const loading = new Promise<boolean>((resolve) => {
    finishLoading = resolve
  })
  return { auth: { isSignedIn: () => loading }, finishLoading }
}

async function redirectOf(run: () => Promise<void>) {
  try {
    await run()
  } catch (err) {
    if (isRedirect(err)) return err.options
    throw err
  }
  return undefined
}

describe("requireAuth", () => {
  it("lets a signed-in user through", async () => {
    await expect(
      requireAuth({
        context: { auth: gate(true) },
        location: { pathname: "/account", searchStr: "" },
      })
    ).resolves.toBeUndefined()
  })

  it("redirects to /auth/login, preserving the attempted URL, when nobody is signed in", async () => {
    const options = await redirectOf(() =>
      requireAuth({
        context: { auth: gate(false) },
        location: { pathname: "/account", searchStr: "" },
      })
    )

    expect(options).toMatchObject({
      to: "/auth/login",
      search: { redirect: "/account" },
    })
  })

  it("keeps the query string on the relative redirect path", async () => {
    const options = await redirectOf(() =>
      requireAuth({
        context: { auth: gate(false) },
        location: { pathname: "/collections", searchStr: "?sort=name&page=2" },
      })
    )

    expect(options?.search).toEqual({
      redirect: "/collections?sort=name&page=2",
    })
  })

  it("waits for Clerk to finish loading before deciding", async () => {
    const { auth, finishLoading } = loadingGate()
    let decided = false
    const guard = requireAuth({
      context: { auth },
      location: { pathname: "/account", searchStr: "" },
    }).then(() => {
      decided = true
    })

    await Promise.resolve()
    expect(decided).toBe(false)

    finishLoading(true)
    await guard
    expect(decided).toBe(true)
  })
})

describe("requireGuest", () => {
  it("redirects a signed-in user to /", async () => {
    const options = await redirectOf(() =>
      requireGuest({ context: { auth: gate(true) } })
    )

    expect(options).toMatchObject({ to: "/" })
  })

  it("lets a guest through", async () => {
    await expect(
      requireGuest({ context: { auth: gate(false) } })
    ).resolves.toBeUndefined()
  })

  it("waits for Clerk to finish loading, so a signed-in user is not shown the form", async () => {
    const { auth, finishLoading } = loadingGate()
    const pending = redirectOf(() => requireGuest({ context: { auth } }))

    finishLoading(true)

    expect(await pending).toMatchObject({ to: "/" })
  })
})

describe("isSameOriginPath", () => {
  it.each(["/collections", "/collections?sort=name", "/"])(
    "accepts %s",
    (path) => {
      expect(isSameOriginPath(path)).toBe(true)
    }
  )

  it.each([
    undefined,
    "",
    "collections",
    "https://evil.example/x",
    "//evil.example",
    "/\\evil.example",
    "/\t/evil.example",
    "/\n/evil.example",
    "/\r/evil.example",
    "/ok\u0000",
  ])("rejects %s", (path) => {
    expect(isSameOriginPath(path)).toBe(false)
  })
})
