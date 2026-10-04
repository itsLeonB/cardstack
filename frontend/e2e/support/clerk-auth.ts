import { clerk, setupClerkTestingToken } from "@clerk/testing/playwright"
import { expect, test } from "playwright/test"
import type { Locator, Page } from "playwright/test"
import {
  CLERK_E2E_SKIP_REASON,
  readClerkE2eCredentials,
} from "./clerk-credentials"

// Signs the dedicated Clerk test user in through the real development
// instance (ADR-0015, security-hardening ticket 09). Nothing here stubs Clerk:
// the session token the app sends to the API is a real one. How it works, the
// secrets it needs and how to run it locally: docs/agents/testing.md,
// "End-to-end authentication".

const credentials = readClerkE2eCredentials(process.env)

function requireCredentials() {
  if (!credentials) {
    throw new Error(`${CLERK_E2E_SKIP_REASON} Call useSignedInSuite() first.`)
  }
  requireSignedInTestWithoutTrace()
  return credentials
}

// Tag of every describe whose specs sign in. playwright.config.ts runs the
// tagged specs in their own project with traces off: a trace records request
// headers (the session cookie and bearer token) and the arguments of every
// action, and the HTML report that CI uploads on a failure would carry both.
// Trace is a worker option, so it cannot be switched off from a describe.
export const SIGNED_IN_TAG = "@signed-in"

// Call first inside every describe tagged SIGNED_IN_TAG: without the
// credentials its tests skip with a reason instead of failing.
export function useSignedInSuite() {
  test.skip(!credentials, CLERK_E2E_SKIP_REASON)
}

// The sign-in helpers refuse to run in an untagged test, because it would run
// in the traced project and record the session.
function requireSignedInTestWithoutTrace() {
  if (!test.info().tags.includes(SIGNED_IN_TAG)) {
    throw new Error(
      `A spec that signs in must sit in a describe tagged "${SIGNED_IN_TAG}" so it runs untraced.`
    )
  }
}

// Signs in without touching the sign-in form: a one-time sign-in token minted
// with the secret key, redeemed in the page. The password never enters the
// browser, so a spec that only needs a session is not affected by Clerk's
// verification or bot-protection steps. Register API stubs first, then call
// this, then navigate: it leaves the page on the landing (a guest page that
// calls no API) with the session active.
export async function signInAsTestUser(page: Page) {
  const { email } = requireCredentials()
  await setupClerkTestingToken({ page })
  await page.goto("/")
  await clerk.signIn({ page, emailAddress: email })
}

// Types a secret without Playwright printing it: `fill()` puts its value in
// the step title and in the call log of a failure, both of which end up in the
// HTML report. The native setter plus an `input` event is what React's
// controlled inputs (Clerk's form) listen for.
async function fillSecret(field: Locator, value: string) {
  await expect(field).toBeVisible()
  await field.evaluate((input, secret) => {
    if (!(input instanceof HTMLInputElement)) {
      throw new Error("Expected an input element")
    }
    Object.getOwnPropertyDescriptor(
      HTMLInputElement.prototype,
      "value"
    )?.set?.call(input, secret)
    input.dispatchEvent(new Event("input", { bubbles: true }))
  }, value)
}

// Signs in by typing into Clerk's real sign-in form at `url`, which is what
// the sign-in spec and the redirect-after-login specs exercise. A caller that
// only needs a session uses signInAsTestUser. `password` overrides the test
// user's, for the wrong-password failure.
export async function signInThroughForm(
  page: Page,
  url: string,
  { password }: { password?: string } = {}
) {
  const testUser = requireCredentials()
  await setupClerkTestingToken({ page })
  await page.goto(url)

  await fillSecret(
    page.getByRole("textbox", { name: /email address/i }),
    testUser.email
  )
  // Clerk shows the password on the same step or on a second one, depending on
  // the instance's settings: continue only when it is not already there.
  const passwordField = page.getByLabel("Password", { exact: true })
  const continueButton = page.getByRole("button", { name: "Continue" })
  await passwordField
    .waitFor({ state: "visible", timeout: 2000 })
    .catch(() => continueButton.click())
  await fillSecret(passwordField, password ?? testUser.password)
  await continueButton.click()
}

// Asserts the shell shows the test user's email without putting it in a
// failure message (a plain toContainText would print the expected text).
export async function expectShowsTestUserEmail(locator: Locator) {
  const { email } = requireCredentials()
  await expect
    .poll(async () =>
      (await locator.innerText()).toLowerCase().includes(email.toLowerCase())
    )
    .toBe(true)
}
