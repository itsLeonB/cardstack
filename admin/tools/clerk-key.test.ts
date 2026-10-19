import { describe, expect, it } from "vitest"
import { clerkKeyCheck } from "./clerk-key"

describe("clerkKeyCheck", () => {
  it("accepts a build that has the key, in any environment", () => {
    expect(clerkKeyCheck({ VITE_CLERK_PUBLISHABLE_KEY: "pk_test_x" })).toBe(
      "ok"
    )
    expect(
      clerkKeyCheck({
        VITE_CLERK_PUBLISHABLE_KEY: "pk_live_x",
        VERCEL_ENV: "production",
      })
    ).toBe("ok")
  })

  it("fails a Vercel production build without the key", () => {
    expect(clerkKeyCheck({ VERCEL_ENV: "production" })).toBe("fatal")
    expect(
      clerkKeyCheck({
        VITE_CLERK_PUBLISHABLE_KEY: "",
        VERCEL_ENV: "production",
      })
    ).toBe("fatal")
  })

  it("only warns elsewhere, so local, CI and preview builds still pass", () => {
    expect(clerkKeyCheck({})).toBe("missing")
    expect(clerkKeyCheck({ VERCEL_ENV: "preview" })).toBe("missing")
  })
})
