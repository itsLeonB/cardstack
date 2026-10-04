import { afterEach, describe, expect, it, vi } from "vitest"
import { cleanup, render } from "@testing-library/react"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { useAuth } from "@clerk/react"
import { SessionSync } from "./session-sync"

// Clerk's hooks need a ClerkProvider talking to Clerk's servers.
// oxlint-disable-next-line anti-slop/no-module-mocking
vi.mock("@clerk/react", () => ({ useAuth: vi.fn() }))

afterEach(cleanup)

const KEY = ["/collections"]

function setup(initial: { isLoaded: boolean; sessionId: string | null }) {
  const queryClient = new QueryClient()
  queryClient.setQueryData(KEY, { status: 200 })
  const mockAuth = (state: { isLoaded: boolean; sessionId: string | null }) =>
    // SAFETY: partial hook result covering only the fields SessionSync reads.
    vi.mocked(useAuth).mockReturnValue(state as any)
  mockAuth(initial)
  // A new element each time: React skips re-rendering an identical one.
  const tree = () => (
    <QueryClientProvider client={queryClient}>
      <SessionSync />
    </QueryClientProvider>
  )
  const view = render(tree())
  return {
    queryClient,
    change(state: { isLoaded: boolean; sessionId: string | null }) {
      mockAuth(state)
      view.rerender(tree())
    },
  }
}

describe("SessionSync", () => {
  it("keeps the cache across the first load, whatever the session", () => {
    const { queryClient, change } = setup({ isLoaded: false, sessionId: null })

    change({ isLoaded: true, sessionId: "sess_1" })

    expect(queryClient.getQueryData(KEY)).toEqual({ status: 200 })
  })

  it("drops the cache when the user signs in", () => {
    const { queryClient, change } = setup({ isLoaded: true, sessionId: null })

    change({ isLoaded: true, sessionId: "sess_1" })

    expect(queryClient.getQueryData(KEY)).toBeUndefined()
  })

  it("drops the cache when the user signs out", () => {
    const { queryClient, change } = setup({
      isLoaded: true,
      sessionId: "sess_1",
    })

    change({ isLoaded: true, sessionId: null })

    expect(queryClient.getQueryData(KEY)).toBeUndefined()
  })

  it("drops the cache when another account takes over", () => {
    const { queryClient, change } = setup({
      isLoaded: true,
      sessionId: "sess_1",
    })

    change({ isLoaded: true, sessionId: "sess_2" })

    expect(queryClient.getQueryData(KEY)).toBeUndefined()
  })

  it("leaves the cache alone when the session is unchanged", () => {
    const { queryClient, change } = setup({
      isLoaded: true,
      sessionId: "sess_1",
    })

    change({ isLoaded: true, sessionId: "sess_1" })

    expect(queryClient.getQueryData(KEY)).toEqual({ status: 200 })
  })
})
