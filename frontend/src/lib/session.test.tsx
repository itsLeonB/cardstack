import { afterEach, describe, expect, it, vi } from "vitest"
import { act, renderHook } from "@testing-library/react"
import { QueryClient } from "@tanstack/react-query"
import { useClerk, useUser } from "@clerk/react"
import { toast } from "sonner"
import {
  LOGOUT_FAILED,
  createSessionChangeHandler,
  resetCache,
  useIsGuest,
  useSession,
  useSignOut,
} from "./session"
import { rearmAuthLost, reportAuthLost, setOnAuthLost } from "./http"

// Clerk's hooks need a mounted ClerkProvider talking to Clerk's servers, so
// they are the boundary to fake (as the generated client is elsewhere).
// oxlint-disable-next-line anti-slop/no-module-mocking
vi.mock("@clerk/react", () =>
  import("@/test-clerk").then((m) => m.clerkModule())
)

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

describe("useIsGuest", () => {
  it("is a Guest only once Clerk has loaded and nobody is signed in", () => {
    clerkUser("loading")
    expect(renderHook(() => useIsGuest()).result.current).toBe(false)
    clerkUser("signed-out")
    expect(renderHook(() => useIsGuest()).result.current).toBe(true)
    clerkUser("signed-in")
    expect(renderHook(() => useIsGuest()).result.current).toBe(false)
  })
})

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

describe("createSessionChangeHandler", () => {
  const KEY = ["/collections"]

  function setup({ onPrivatePage = true } = {}) {
    const queryClient = new QueryClient()
    queryClient.setQueryData(KEY, { status: 200 })
    const onLost = vi.fn()
    rearmAuthLost()
    setOnAuthLost(onLost)
    const handler = createSessionChangeHandler(queryClient, () => onPrivatePage)
    return { queryClient, onLost, handler }
  }

  afterEach(() => {
    setOnAuthLost(null)
  })

  it("drops cached data on sign-in, and leaves data loaded afterwards alone", () => {
    const { queryClient, handler } = setup()

    handler("sess_1", null)
    expect(queryClient.getQueryData(KEY)).toBeUndefined()

    // Clerk reports the change before it navigates, so what the next page
    // loads lands after the reset and is not touched by it.
    queryClient.setQueryData(KEY, { status: 200, fresh: true })
    expect(queryClient.getQueryData(KEY)).toEqual({ status: 200, fresh: true })
  })

  it("announces a session Clerk dropped under a private page, once", () => {
    const { queryClient, onLost, handler } = setup()

    handler(null, "sess_1")
    handler(null, "sess_1")

    expect(queryClient.getQueryData(KEY)).toBeUndefined()
    expect(onLost).toHaveBeenCalledTimes(1)
  })

  it("does not announce a session dropped while a public page is open", () => {
    const { queryClient, onLost, handler } = setup({ onPrivatePage: false })

    handler(null, "sess_1")

    expect(queryClient.getQueryData(KEY)).toBeUndefined()
    expect(onLost).not.toHaveBeenCalled()
  })

  it("does not re-open the notice on sign-out, so a lost session is not announced twice", () => {
    const { onLost, handler } = setup()
    reportAuthLost()

    // The auth-lost handler signs Clerk out, which is itself a sign-out change.
    handler(null, "sess_1")

    expect(onLost).toHaveBeenCalledTimes(1)
  })

  it("re-opens the notice when a new session starts", () => {
    const { onLost, handler } = setup()
    reportAuthLost()

    handler("sess_2", null)
    handler(null, "sess_2")

    expect(onLost).toHaveBeenCalledTimes(2)
  })

  it("does not announce the sign-out the user asked for, only the next one", async () => {
    const { onLost, handler } = setup()
    // SAFETY: partial hook result covering only `signOut`.
    mockUseClerk.mockReturnValue({ signOut: async () => {} } as any)
    const { result } = renderHook(() => useSignOut())

    await act(() => result.current.signOut())
    handler(null, "sess_1")
    expect(onLost).not.toHaveBeenCalled()

    handler(null, "sess_2")
    expect(onLost).toHaveBeenCalledTimes(1)
  })

  it("does not remember a sign-out that failed", async () => {
    const { onLost, handler } = setup()
    // SAFETY: partial hook result covering only `signOut`.
    mockUseClerk.mockReturnValue({
      signOut: async () => {
        throw new Error("network")
      },
    } as any)
    const { result } = renderHook(() => useSignOut())

    await act(() => result.current.signOut())
    handler(null, "sess_1")

    expect(onLost).toHaveBeenCalledTimes(1)
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
