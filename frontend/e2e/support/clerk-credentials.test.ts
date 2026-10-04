import { describe, expect, it } from "vitest"
import {
  decideClerkE2e,
  readClerkE2eCredentials,
  REQUIRE_SIGN_IN_VARIABLE,
} from "./clerk-credentials"

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

describe("decideClerkE2e", () => {
  it("runs when everything is set, required or not", () => {
    expect(decideClerkE2e(full)).toEqual({ action: "run" })
    expect(
      decideClerkE2e({ ...full, [REQUIRE_SIGN_IN_VARIABLE]: "true" })
    ).toEqual({ action: "run" })
  })

  it("skips with the missing names when credentials are absent and sign-in is not required", () => {
    for (const env of [
      {},
      { [REQUIRE_SIGN_IN_VARIABLE]: "false" },
      { [REQUIRE_SIGN_IN_VARIABLE]: "" },
    ]) {
      const decision = decideClerkE2e(env)
      expect(decision.action).toBe("skip")
      expect(decision).toMatchObject({
        reason: expect.stringContaining("CLERK_SECRET_KEY"),
      })
    }
  })

  it("fails when sign-in is required and anything is missing, naming only what is missing", () => {
    const decision = decideClerkE2e({
      ...full,
      E2E_CLERK_USER_PASSWORD: "",
      [REQUIRE_SIGN_IN_VARIABLE]: "true",
    })
    expect(decision.action).toBe("fail")
    const message = decision.action === "fail" ? decision.message : ""
    expect(message).toContain("E2E_CLERK_USER_PASSWORD")
    expect(message).not.toContain("CLERK_SECRET_KEY")
  })

  it("never puts a value in a message", () => {
    const env = {
      CLERK_SECRET_KEY: "sk_test_SECRETVALUE",
      E2E_CLERK_USER_EMAIL: "who@example.com",
    }
    for (const required of ["true", "false"]) {
      const decision = decideClerkE2e({
        ...env,
        [REQUIRE_SIGN_IN_VARIABLE]: required,
      })
      const text = JSON.stringify(decision)
      expect(text).not.toContain("SECRETVALUE")
      expect(text).not.toContain("who@example.com")
    }
  })

  it("only the exact string true requires sign-in", () => {
    expect(decideClerkE2e({ [REQUIRE_SIGN_IN_VARIABLE]: "1" }).action).toBe(
      "skip"
    )
  })
})
