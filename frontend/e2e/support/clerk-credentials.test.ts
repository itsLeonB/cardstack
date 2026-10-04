import { describe, expect, it } from "vitest"
import { readClerkE2eCredentials } from "./clerk-credentials"

const full = {
  CLERK_SECRET_KEY: "sk_test_x",
  E2E_CLERK_USER_EMAIL: "user+clerk_test@example.com",
  E2E_CLERK_USER_PASSWORD: "pw",
}

describe("readClerkE2eCredentials", () => {
  it("returns the test user when everything is set", () => {
    expect(readClerkE2eCredentials(full)).toEqual({
      email: "user+clerk_test@example.com",
      password: "pw",
    })
  })

  it.each(Object.keys(full))("is null when %s is missing", (key) => {
    expect(readClerkE2eCredentials({ ...full, [key]: undefined })).toBeNull()
  })

  it("treats empty and blank values as missing", () => {
    // GitHub passes an unset secret to a step as an empty string.
    expect(
      readClerkE2eCredentials({ ...full, CLERK_SECRET_KEY: "" })
    ).toBeNull()
    expect(
      readClerkE2eCredentials({ ...full, E2E_CLERK_USER_EMAIL: "  " })
    ).toBeNull()
  })

  it("keeps the password exactly as given", () => {
    expect(
      readClerkE2eCredentials({ ...full, E2E_CLERK_USER_PASSWORD: " pw " })
        ?.password
    ).toBe(" pw ")
  })
})
