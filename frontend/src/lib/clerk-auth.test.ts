import { beforeEach, describe, expect, it, vi } from "vitest"
import { getToken } from "@clerk/react"
import { clerkAuth, getSessionToken } from "./clerk-auth"

// Clerk's standalone `getToken` reads window.Clerk, which only a real Clerk
// instance sets, so it is the boundary to fake.
// oxlint-disable-next-line anti-slop/no-module-mocking
vi.mock("@clerk/react", () => ({ getToken: vi.fn() }))

const mockGetToken = vi.mocked(getToken)

beforeEach(() => {
  mockGetToken.mockReset()
})

describe("getSessionToken", () => {
  it("forwards the options, so a retry can skip Clerk's cache", async () => {
    mockGetToken.mockResolvedValue("jwt")

    await expect(getSessionToken({ skipCache: true })).resolves.toBe("jwt")
    expect(mockGetToken).toHaveBeenCalledWith({ skipCache: true })
  })
})

describe("clerkAuth.isSignedIn", () => {
  it("is true when Clerk hands out a token", async () => {
    mockGetToken.mockResolvedValue("jwt")

    await expect(clerkAuth.isSignedIn()).resolves.toBe(true)
  })

  it("is false when nobody is signed in", async () => {
    mockGetToken.mockResolvedValue(null)

    await expect(clerkAuth.isSignedIn()).resolves.toBe(false)
  })

  it("is false when Clerk cannot load or answer", async () => {
    mockGetToken.mockRejectedValue(new Error("Timeout waiting for Clerk"))

    await expect(clerkAuth.isSignedIn()).resolves.toBe(false)
  })
})
