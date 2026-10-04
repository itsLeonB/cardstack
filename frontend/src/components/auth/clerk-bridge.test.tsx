import { afterEach, describe, expect, it, vi } from "vitest"
import { cleanup, render } from "@testing-library/react"
import { useAuth, useClerk } from "@clerk/react"
import { clerkAuth } from "@/lib/clerk-auth"
import { ClerkBridge } from "./clerk-bridge"

// oxlint-disable-next-line anti-slop/no-module-mocking
vi.mock("@clerk/react", () =>
  import("@/test-clerk").then((m) => m.clerkModule())
)

afterEach(() => {
  cleanup()
  vi.restoreAllMocks()
})

function clerkState(isLoaded: boolean) {
  // SAFETY: partial hook results covering only the fields ClerkBridge reads.
  vi.mocked(useAuth).mockReturnValue({ isLoaded } as any)
  const instance = { session: null }
  // SAFETY: stand-in for the Clerk instance; the bridge only passes it on.
  vi.mocked(useClerk).mockReturnValue(instance as any)
  return instance
}

describe("ClerkBridge", () => {
  it("publishes nothing until Clerk has loaded", () => {
    const publish = vi.spyOn(clerkAuth, "publish")
    clerkState(false)

    render(<ClerkBridge />)

    expect(publish).not.toHaveBeenCalled()
  })

  it("publishes the loaded instance, and withdraws it on unmount", () => {
    const publish = vi.spyOn(clerkAuth, "publish")
    const instance = clerkState(true)

    const view = render(<ClerkBridge />)
    expect(publish).toHaveBeenLastCalledWith(instance)

    view.unmount()
    expect(publish).toHaveBeenLastCalledWith(null)
  })
})
