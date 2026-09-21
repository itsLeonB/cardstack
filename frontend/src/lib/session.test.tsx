import { describe, expect, it, vi, beforeEach } from "vitest"
import { renderHook } from "@testing-library/react"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import type { ReactNode } from "react"
import type * as AuthModule from "@/generated/endpoints/auth/auth"
import {
  useGetCurrentUser,
  useLogin,
  useLogout,
  getGetCurrentUserQueryKey,
} from "@/generated/endpoints/auth/auth"
import type { loginResponse, logoutResponse } from "@/generated/endpoints/auth/auth"
import { useLoginMutation, useLogoutMutation, useSession } from "./session"

// The auth endpoints are generated orval/TanStack Query hooks with no
// service layer to inject; mocking the generated module is the standard way
// to isolate these wrappers from it in tests (see health-check.test.tsx).
// oxlint-disable-next-line anti-slop/no-module-mocking
vi.mock("@/generated/endpoints/auth/auth", async () => {
  const actual =
    await vi.importActual<typeof AuthModule>("@/generated/endpoints/auth/auth")
  return {
    ...actual,
    useGetCurrentUser: vi.fn(),
    useLogin: vi.fn(),
    useLogout: vi.fn(),
  }
})

const mockUseGetCurrentUser = vi.mocked(useGetCurrentUser)
const mockUseLogin = vi.mocked(useLogin)
const mockUseLogout = vi.mocked(useLogout)

function createWrapper() {
  const queryClient = new QueryClient()
  return function Wrapper({ children }: { children: ReactNode }) {
    return (
      <QueryClientProvider client={queryClient}>
        {children}
      </QueryClientProvider>
    )
  }
}

describe("useSession", () => {
  it("reports authenticated with the user when /auth/me returns 200", () => {
    // SAFETY: partial mock covering only the fields useSession reads
    // (data, isPending); the real hook return has more.
    mockUseGetCurrentUser.mockReturnValue({
      data: { status: 200, data: { data: { id: "1", email: "a@b.com" } } },
      isPending: false,
    } as any)

    const { result } = renderHook(() => useSession(), {
      wrapper: createWrapper(),
    })

    expect(result.current.isAuthenticated).toBe(true)
    expect(result.current.user).toEqual({ id: "1", email: "a@b.com" })
  })

  it("reports logged out (not an error) when /auth/me returns 401", () => {
    // SAFETY: partial mock covering only the fields useSession reads
    // (data, isPending); the real hook return has more.
    mockUseGetCurrentUser.mockReturnValue({
      data: { status: 401, data: { detail: "Unauthorized" } },
      isPending: false,
    } as any)

    const { result } = renderHook(() => useSession(), {
      wrapper: createWrapper(),
    })

    expect(result.current.isAuthenticated).toBe(false)
    expect(result.current.user).toBeNull()
  })

  it("tunes the underlying query so a 401 never retries", () => {
    // SAFETY: partial mock covering only the fields useSession reads
    // (data, isPending); the real hook return has more.
    mockUseGetCurrentUser.mockReturnValue({
      data: undefined,
      isPending: true,
    } as any)

    renderHook(() => useSession(), { wrapper: createWrapper() })

    expect(mockUseGetCurrentUser).toHaveBeenCalledWith(
      expect.objectContaining({
        query: expect.objectContaining({ retry: false }),
      }),
    )
  })
})

describe("useLoginMutation", () => {
  beforeEach(() => {
    mockUseLogin.mockReset()
  })

  it("invalidates the session query once login succeeds", () => {
    let capturedOnSuccess: ((response: loginResponse) => void) | undefined
    mockUseLogin.mockImplementation((options) => {
      // SAFETY: mutation.onSuccess is a known field on the real useLogin
      // options; only it is exercised by this mock.
      capturedOnSuccess = options?.mutation?.onSuccess as never
      // SAFETY: partial mock; only mutation.onSuccess is exercised here.
      return {} as any
    })

    const queryClient = new QueryClient()
    const invalidateSpy = vi.spyOn(queryClient, "invalidateQueries")

    function Wrapper({ children }: { children: ReactNode }) {
      return (
        <QueryClientProvider client={queryClient}>
          {children}
        </QueryClientProvider>
      )
    }

    renderHook(() => useLoginMutation(), { wrapper: Wrapper })

    capturedOnSuccess?.({
      status: 200,
      data: { data: { message: "ok" } },
      headers: new Headers(),
    })

    expect(invalidateSpy).toHaveBeenCalledWith({
      queryKey: getGetCurrentUserQueryKey(),
    })
  })

  it("does not invalidate the session query on a failed login (e.g. 401)", () => {
    let capturedOnSuccess: ((response: loginResponse) => void) | undefined
    mockUseLogin.mockImplementation((options) => {
      // SAFETY: mutation.onSuccess is a known field on the real useLogin
      // options; only it is exercised by this mock.
      capturedOnSuccess = options?.mutation?.onSuccess as never
      // SAFETY: partial mock; only mutation.onSuccess is exercised here.
      return {} as any
    })

    const queryClient = new QueryClient()
    const invalidateSpy = vi.spyOn(queryClient, "invalidateQueries")

    function Wrapper({ children }: { children: ReactNode }) {
      return (
        <QueryClientProvider client={queryClient}>
          {children}
        </QueryClientProvider>
      )
    }

    renderHook(() => useLoginMutation(), { wrapper: Wrapper })

    capturedOnSuccess?.({
      status: 401,
      data: { detail: "bad creds" },
      headers: new Headers(),
    })

    expect(invalidateSpy).not.toHaveBeenCalled()
  })
})

describe("useLogoutMutation", () => {
  beforeEach(() => {
    mockUseLogout.mockReset()
  })

  it("invalidates the session query once logout succeeds (204)", () => {
    let capturedOnSuccess: ((response: logoutResponse) => void) | undefined
    mockUseLogout.mockImplementation((options) => {
      // SAFETY: mutation.onSuccess is a known field on the real useLogout
      // options; only it is exercised by this mock.
      capturedOnSuccess = options?.mutation?.onSuccess as never
      // SAFETY: partial mock; only mutation.onSuccess is exercised here.
      return {} as any
    })

    const queryClient = new QueryClient()
    const invalidateSpy = vi.spyOn(queryClient, "invalidateQueries")

    function Wrapper({ children }: { children: ReactNode }) {
      return (
        <QueryClientProvider client={queryClient}>
          {children}
        </QueryClientProvider>
      )
    }

    renderHook(() => useLogoutMutation(), { wrapper: Wrapper })

    capturedOnSuccess?.({ status: 204, data: undefined, headers: new Headers() })

    expect(invalidateSpy).toHaveBeenCalledWith({
      queryKey: getGetCurrentUserQueryKey(),
    })
  })
})
