import { test, expect } from "playwright/test"
import {
  expectShowsTestUserEmail,
  signInThroughForm,
  SIGNED_IN_TAG,
  useSignedInSuite,
} from "./support/clerk-auth"

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
  }) => {
    // A fixed dummy, never derived from the real password.
    await signInThroughForm(page, "/auth/login", {
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
