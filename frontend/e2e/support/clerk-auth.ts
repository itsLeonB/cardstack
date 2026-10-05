import { clerk, setupClerkTestingToken } from "@clerk/testing/playwright"
import { expect, test } from "playwright/test"
import type { Locator, Page } from "playwright/test"
import { decideClerkE2e, readClerkE2eCredentials } from "./clerk-credentials"

// Signs the dedicated Clerk test user in through the real development instance
// (ADR-0015); nothing here stubs Clerk. Setup, secrets and local runs:
// docs/agents/testing.md, "End-to-end authentication".
//
// Keeping credentials out of the uploaded HTML report is why the specs that
// sign in are tagged SIGNED_IN_TAG and why the password is not `fill()`ed:
// - A trace records request headers (the session cookie and bearer token) and
//   every action's arguments. Trace is a worker option, so a describe cannot
//   turn it off; playwright.config.ts runs the tagged specs in their own
//   untraced project.
// - `fill()` prints its value in the step title and in failure messages, so
//   fillSecret sets the value through `evaluate` instead.

export const SIGNED_IN_TAG = "@signed-in"

const credentials = readClerkE2eCredentials(process.env)

function requireCredentials() {
  if (!credentials) {
    throw new Error("Call useSignedInSuite() before signing in.")
  }
  return credentials
}

function requireSignedInTag() {
  if (!test.info().tags.includes(SIGNED_IN_TAG)) {
    throw new Error(
      `A spec that signs in must sit in a describe tagged "${SIGNED_IN_TAG}" so it runs untraced.`
    )
  }
}

// Call first inside every describe tagged SIGNED_IN_TAG. Without credentials
// its tests skip, or fail where E2E_REQUIRE_SIGN_IN says they must exist.
export function useSignedInSuite() {
  const decision = decideClerkE2e(process.env)
  if (decision.action === "skip") test.skip(true, decision.reason)
  if (decision.action === "fail") {
    test.beforeEach(() => {
      throw new Error(decision.message)
    })
  }
}

// Redeems a one-time sign-in token minted with the secret key, so the password
// never enters the browser and Clerk's verification steps do not apply.
// Register API stubs first, then call this, then navigate: it leaves the page
// on the landing (a guest page that calls no API) with the session active.
export async function signInAsTestUser(page: Page) {
  requireSignedInTag()
  const { email } = requireCredentials()
  await setupClerkTestingToken({ page })
  await page.goto("/")
  await clerk.signIn({ page, emailAddress: email })
}

// The native setter plus an `input` event is what React's controlled inputs
// (Clerk's form) listen for.
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

// Types into Clerk's real sign-in form at `url`, for the specs where the form
// is the subject; a spec that only needs a session uses signInAsTestUser.
// `password` overrides the test user's, for the wrong-password failure.
export async function signInThroughForm(
  page: Page,
  url: string,
  { password }: { password?: string } = {}
) {
  requireSignedInTag()
  const testUser = requireCredentials()
  await setupClerkTestingToken({ page })
  await page.goto(url)

  await fillSecret(
    page.getByRole("textbox", { name: /email address/i }),
    testUser.email
  )
  // Clerk shows the password on the same step as the email or on a second one,
  // depending on the instance's settings.
  const passwordField = page.getByLabel("Password", { exact: true })
  const continueButton = page.getByRole("button", {
    name: "Continue",
    // Not a substring match: "Continue with Google" is on the same form.
    exact: true,
  })
  await passwordField.or(continueButton).first().waitFor({ state: "visible" })
  if (!(await passwordField.isVisible())) await continueButton.click()
  const typed = password ?? testUser.password
  await fillSecret(passwordField, typed)
  // Registered before the click so the response cannot be missed. The code
  // field shows up before Clerk has sent the code, and a code typed in that
  // gap is refused ("send a verification code before attempting to verify").
  const codeSent = page.waitForResponse((res) =>
    res.url().includes("/prepare_second_factor")
  )
  codeSent.catch(() => {}) // only awaited when a code is asked for
  await continueButton.click()

  // Clerk submits a password typed on the first step straight away. When it is
  // wrong, Clerk drops to an explicit password step with no error, so type it
  // again there to get the error. A device Clerk has not seen (every CI run)
  // asks for an emailed code; the test user's +clerk_test address accepts
  // 424242. Neither prompt shows when Clerk trusts the device.
  const passwordStep = page.getByRole("heading", {
    name: "Enter your password",
  })
  const codeField = page.getByRole("textbox", {
    name: "Enter verification code",
  })
  const prompted = await passwordStep
    .or(codeField)
    .waitFor({ state: "visible", timeout: 10_000 })
    .then(
      () => true,
      () => false
    )
  if (!prompted) return
  if (await passwordStep.isVisible()) {
    await fillSecret(passwordField, typed)
    await continueButton.click()
    return
  }
  await codeSent
  await codeField.fill("424242")
}

// A plain toContainText would print the email in a failure message.
export async function expectShowsTestUserEmail(locator: Locator) {
  const { email } = requireCredentials()
  await expect
    .poll(async () =>
      (await locator.innerText()).toLowerCase().includes(email.toLowerCase())
    )
    .toBe(true)
}
