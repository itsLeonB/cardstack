import { randomUUID } from "node:crypto"
import { test as base, expect } from "playwright/test"
import { z } from "zod"
import {
  expectShowsTestUserEmail,
  signInThroughForm,
  SIGNED_IN_TAG,
  useSignedInSuite,
} from "./support/clerk-auth"

const test = base.extend<{ disposableEmail: string }>({
  disposableEmail: async ({ page }, use) => {
    void page
    const secret = process.env.CLERK_SECRET_KEY?.trim()
    if (!secret?.startsWith("sk_test_")) {
      throw new Error("Disposable users require a Clerk development secret key")
    }
    const email = `e2e-${randomUUID()}+clerk_test@example.com`
    const headers = {
      Authorization: `Bearer ${secret}`,
      "Content-Type": "application/json",
    }
    const response = await fetch("https://api.clerk.com/v1/users", {
      method: "POST",
      headers,
      signal: AbortSignal.timeout(10_000),
      body: JSON.stringify({
        email_address: [email],
        password: `E2e-${randomUUID()}-aA1!`,
      }),
    })
    if (!response.ok) {
      throw new Error(
        `Clerk test user creation failed: HTTP ${response.status}`
      )
    }
    const user = z
      .object({ id: z.string().min(1) })
      .parse(await response.json())
    let useError: unknown
    try {
      await use(email)
    } catch (error) {
      useError = error
    }
    try {
      const deletion = await fetch(
        `https://api.clerk.com/v1/users/${encodeURIComponent(user.id)}`,
        { method: "DELETE", headers, signal: AbortSignal.timeout(10_000) }
      )
      if (!deletion.ok) {
        throw new Error(
          `Clerk test user ${user.id} deletion failed: HTTP ${deletion.status}`
        )
      }
    } catch (error) {
      const cleanupError = new Error(
        `Clerk test user ${user.id} cleanup failed`,
        { cause: error }
      )
      if (useError)
        throw new AggregateError(
          [useError, cleanupError],
          "Test and Clerk cleanup failed"
        )
      throw cleanupError
    }
    if (useError) throw useError
  },
})

// The thin end-to-end proof of sign-in (ADR-0015): Clerk's real development
// instance, the dedicated test user, Clerk Testing Tokens. The happy path and
// the one key failure, not every branch of the form. Needs the Clerk
// credentials and a running backend; skips without them (see
// docs/agents/testing.md, "End-to-end authentication").
test.describe("Sign-in with Clerk", { tag: SIGNED_IN_TAG }, () => {
  useSignedInSuite()

  test("signs in with email and password and reaches a private page through the API", async ({
    page,
  }) => {
    await signInThroughForm(page, "/auth/login")

    // No redirect param, so Clerk's forced redirect lands on the account page.
    await expect(page).toHaveURL(/\/account$/, { timeout: 15_000 })
    const main = page.getByRole("main")
    await expect(
      main.getByRole("heading", { level: 1, name: "Account" })
    ).toBeVisible()
    await expectShowsTestUserEmail(main)

    // The Collections list is a real API call carrying the real session token;
    // a rejected token would send the user back to sign-in.
    await page
      .getByRole("banner")
      .getByRole("link", { name: "Collections" })
      .click()
    await expect(page).toHaveURL(/\/collections$/)
    await expect(
      page.getByRole("heading", { level: 1, name: "Collections" })
    ).toBeVisible()
  })

  test("a wrong password shows Clerk's error and gives no session", async ({
    page,
    disposableEmail,
  }) => {
    // A fixed dummy, never derived from the real password.
    await signInThroughForm(page, "/auth/login", {
      email: disposableEmail,
      password: "definitely-not-the-password-1A!",
    })

    // `.first()`: Clerk repeats the message in a screen-reader live region.
    await expect(page.getByText(/password is incorrect/i).first()).toBeVisible()
    // Still on sign-in, possibly on its explicit password step (`factor-one`).
    await expect(page).toHaveURL(/\/auth\/login(\/factor-one)?$/)

    // Still a guest: a private page bounces back to sign-in.
    await page.goto("/collections")
    await expect(page).toHaveURL(/\/auth\/login\?redirect=%2Fcollections$/, {
      timeout: 15_000,
    })
  })
})
