import { afterEach, describe, expect, it, vi } from "vitest"
import { act, renderHook } from "@testing-library/react"
import { QueryClient } from "@tanstack/react-query"
import { useClerk, useUser } from "@clerk/react"
import { toast } from "sonner"
import {
  LOGOUT_FAILED,
  handleSessionChange,
  resetCache,
  useSession,
  useSignOut,
} from "./session"
import { customFetch, setOnAuthLost, setTokenGetter } from "./http"

// Clerk's hooks need a mounted ClerkProvider talking to Clerk's servers, so
// they are the boundary to fake (as the generated client is elsewhere).
// oxlint-disable-next-line anti-slop/no-module-mocking
vi.mock("@clerk/react", () => ({ useUser: vi.fn(), useClerk: vi.fn() }))

const mockUseUser = vi.mocked(useUser)
const mockUseClerk = vi.mocked(useClerk)

afterEach(() => {
  vi.restoreAllMocks()
})

function clerkUser(state: "loading" | "signed-out" | "signed-in") {
  // SAFETY: partial hook results covering only the fields useSession reads.
  mockUseUser.mockReturnValue(
    (state === "loading"
      ? { isLoaded: false, isSignedIn: undefined, user: undefined }
      : state === "signed-out"
        ? { isLoaded: true, isSignedIn: false, user: null }
        : {
            isLoaded: true,
            isSignedIn: true,
            user: {
              id: "user_1",
              fullName: "Ada Lovelace",
              primaryEmailAddress: { emailAddress: "ada@example.com" },
            },
          }) as any
  )
}

describe("useSession", () => {
  it("is loading, and not authenticated, until Clerk has loaded", () => {
    clerkUser("loading")

    const { result } = renderHook(() => useSession())

    expect(result.current).toEqual({
      user: null,
      isAuthenticated: false,
      isLoading: true,
    })
  })

  it("reports a guest once Clerk has loaded with no session", () => {
    clerkUser("signed-out")

    const { result } = renderHook(() => useSession())

    expect(result.current).toEqual({
      user: null,
      isAuthenticated: false,
      isLoading: false,
    })
  })

  it("reports the signed-in user's name and email from Clerk's user data", () => {
    clerkUser("signed-in")

    const { result } = renderHook(() => useSession())

    expect(result.current).toEqual({
      user: { id: "user_1", name: "Ada Lovelace", email: "ada@example.com" },
      isAuthenticated: true,
      isLoading: false,
    })
  })

  it("has a null name and email when Clerk has none", () => {
    // SAFETY: partial hook result, see clerkUser().
    mockUseUser.mockReturnValue({
      isLoaded: true,
      isSignedIn: true,
      user: { id: "user_2", fullName: null, primaryEmailAddress: null },
    } as any)

    const { result } = renderHook(() => useSession())

    expect(result.current.user).toEqual({
      id: "user_2",
      name: null,
      email: null,
    })
  })
})

describe("resetCache", () => {
  it("drops every cached query, so the next session never sees the last one's data", () => {
    const queryClient = new QueryClient()
    queryClient.setQueryData(["/collections"], { status: 200 })
    queryClient.setQueryData(["/inventory"], { status: 200 })

    resetCache(queryClient)

    expect(queryClient.getQueryData(["/collections"])).toBeUndefined()
    expect(queryClient.getQueryData(["/inventory"])).toBeUndefined()
  })
})

describe("handleSessionChange", () => {
  it("drops cached data and lets the next expiry be announced again", async () => {
    const queryClient = new QueryClient()
    queryClient.setQueryData(["/collections"], { status: 200 })
    vi.stubGlobal(
      "fetch",
      vi.fn(async () => new Response("{}", { status: 401 }))
    )
    setTokenGetter(async () => "token")
    const onLost = vi.fn()
    setOnAuthLost(onLost)
    try {
      await customFetch("https://api.example.com/x", { method: "GET" })
      expect(onLost).toHaveBeenCalledTimes(1)

      handleSessionChange(queryClient)
      await customFetch("https://api.example.com/x", { method: "GET" })

      expect(queryClient.getQueryData(["/collections"])).toBeUndefined()
      expect(onLost).toHaveBeenCalledTimes(2)
    } finally {
      setTokenGetter(null)
      setOnAuthLost(null)
      vi.unstubAllGlobals()
    }
  })
})

describe("useSignOut", () => {
  function clerkWithSignOut(signOut: () => Promise<void>) {
    // SAFETY: partial hook result covering only `signOut`.
    mockUseClerk.mockReturnValue({ signOut } as any)
  }

  it("signs out of Clerk and sends the user to the login page", async () => {
    const signOut = vi.fn(async () => {})
    clerkWithSignOut(signOut)
    const error = vi.spyOn(toast, "error")
    const { result } = renderHook(() => useSignOut())

    await act(() => result.current.signOut())

    expect(signOut).toHaveBeenCalledWith({ redirectUrl: "/auth/login" })
    expect(error).not.toHaveBeenCalled()
    expect(result.current.isPending).toBe(false)
  })

  it("is pending while Clerk signs out", async () => {
    let finish: () => void = () => {}
    clerkWithSignOut(
      () =>
        new Promise<void>((resolve) => {
          finish = resolve
        })
    )
    const { result } = renderHook(() => useSignOut())

    let done: Promise<void> = Promise.resolve()
    act(() => {
      done = result.current.signOut()
    })
    expect(result.current.isPending).toBe(true)

    await act(async () => {
      finish()
      await done
    })
    expect(result.current.isPending).toBe(false)
  })

  it("toasts, rather than leaving, when Clerk fails to sign out", async () => {
    clerkWithSignOut(async () => {
      throw new Error("network")
    })
    const error = vi.spyOn(toast, "error")
    const { result } = renderHook(() => useSignOut())

    await act(() => result.current.signOut())

    expect(error).toHaveBeenCalledWith(LOGOUT_FAILED)
    expect(result.current.isPending).toBe(false)
  })
})
