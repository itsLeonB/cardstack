import { describe, expect, it } from "vitest"
import { apiAddressCheck } from "./api-address"

describe("apiAddressCheck", () => {
  it("accepts a build that has the API address, in any environment", () => {
    expect(
      apiAddressCheck({
        VITE_API_BASE_URL: "https://api.example.com",
        VERCEL_ENV: "production",
      })
    ).toBe("ok")
  })

  it("fails a production build without the API address", () => {
    expect(apiAddressCheck({ VERCEL_ENV: "production" })).toBe("fatal")
  })

  it("only warns about a local or preview build without the API address", () => {
    expect(apiAddressCheck({})).toBe("missing")
    expect(apiAddressCheck({ VERCEL_ENV: "preview" })).toBe("missing")
  })
})
