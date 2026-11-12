# Keep wrong-password tests off the shared Clerk user

Use a new disposable Clerk development user for each wrong-password test attempt, including retries. Keep the shared user for successful sign-ins only. This keeps the real Clerk failure check without putting the shared account at risk.

Checked 2026-11-12 against official Clerk docs and first-party source. Read `../issues/02-investigate-local-sign-in-timeout.md` and `docs/agents/testing.md`. No auth tests, credential reads, settings changes, user creation, deletion, or unlock calls took place.

## Ranked recommendation

1. **Use a disposable user for the negative test.** In a test-scoped Node-side fixture, call `users.createUser({ emailAddress: [uniqueEmail], password })`, retain its returned ID, submit one wrong password through the real form, assert the password error and no session, and call `users.deleteUser(id)` during fixture teardown even if the assertion fails. Use a unique address such as `e2e-wrong-password-<uuid>+clerk_test@example.com`; create a fresh user on every retry and concurrent run. Do not reuse `E2E_CLERK_USER_EMAIL`. This is a recommendation based on Clerk's supported create/delete APIs and its own disposable sign-in example, not a claim that Clerk publishes this exact negative-test recipe. [S5][S6][S7]
2. **If disposable users cannot ship yet, isolate a dedicated negative-test user.** Never use that account for happy-path or application tests. Reuse still accumulates failed attempts and can lock that account, so this is containment, not a complete fix. Keep retries off for this test until the disposable fixture exists. An explicitly approved Node-side `unlockUser(id)` can recover a locked negative-test user, but it adds shared state and races; do not automatically unlock the shared user before every run. [S1][S4]
3. **Do not weaken instance-wide lockout as the test fix.** Clerk lets admins change the attempt limit, duration, or enabled state. Raising the limit only postpones the problem; disabling lockout changes security behavior for every affected user in that instance. No setting change is needed for option 1. [S1]

## Verified findings

### 1. Testing Tokens are not a documented lockout bypass

Clerk says: “Testing Tokens allow you to bypass bot detection mechanisms.” Current docs also name specific block/challenge protections that sign-ups skip and state that sign-ins skip “Require a minimum ClerkJS version.” Thus “bot detection only” is slightly too narrow for the current documented behavior, but the docs do **not** promise an account-lockout exemption. [S2]

The protections reference scopes its token exemption to the section **Protections that block or challenge**: a request with a token “isn't checked by these protections.” Its table lists account lockout separately with action **Lock account**, not **Block** or **Challenge**. Do not read that scoped exemption as a bypass of every Clerk protection. [S3]

The SDK's `setupClerkTestingToken` comment says “Bypasses bot protection by appending the testing token in the Frontend API requests.” Its implementation adds the token query parameter, sets `captcha_bypass` in responses, and handles request retries. It contains no unlock request or reset of user verification attempts. This confirms what the client helper does; it does not expose Clerk's private server-side lockout implementation. **Safe conclusion: keep Testing Tokens, but never rely on them to prevent lockout.** No live test of token/lockout behavior was performed. [S8]

### 2. A development test user can still face account lockout

Clerk says it “automatically tracks each user's failed verification attempts during sign-in, such as an incorrect password or code.” The settings table describes failed sign-ins “in a row.” Password attempts, password resets, backup code requests, email/phone codes, and TOTP attempts count. [S1]

Current docs give new instances a default of **10 attempts and one hour**. They explicitly say the default changed from **100 to 10 on July 6, 2026**, while existing instances kept their previous limit. The minimum configurable attempt limit is 5; duration can be a time limit or **Until unlocked**. Do not assume this project's instance uses 10 or 100. The reported 60-minute lock matches the documented default duration, but does not establish its configured threshold. [S1]

The docs describe instance settings and give no development-instance or test-address exemption from lockout. For a future authorized inspection, select the development instance and use **Protect → Protections → Lock accounts after failed sign-ins**; the current reference routes setting changes through each protection's settings icon. This research did not inspect the Dashboard or establish the live configuration. [S1][S3]

### 3. `+clerk_test` prevents email delivery; it does not promise lockout immunity

“Every development instance has Test mode enabled by default.” With Test mode enabled, “Any email with the `+clerk_test` subaddress is a test email address,” and Clerk “never sends email” to it, including security notifications. Email verification uses `424242`. None of these passages grants extra password attempts or disables lockout. A test user can therefore receive no lockout email; lack of email is not proof that the account remains usable. [S9]

Use the test address for disposable users to avoid mail delivery, not as a security bypass. The fixed code covers email verification, not passwords or arbitrary MFA methods. [S9]

### 4. `users.unlockUser` is recovery, not test isolation

The Backend SDK documents `unlockUser(userId: string): Promise<User>` as “Removes a sign-in lock ... allowing them to sign in again.” It wraps `POST /users/{user_id}/unlock` in BAPI (`https://api.clerk.com/v1/users/{user_id}/unlock`). The first-party `UserApi.unlockUser` checks the ID and sends that POST. [S4][S10]

This method needs backend authority; keep the secret key and any future create/delete/unlock work in the Node test process, not browser code or a new public app endpoint. The cited unlock contract does not specify that calling it on an already unlocked user clears a partial failed-attempt count. Do not use it as an assumed pre-test counter reset. No unlock call was made. [S4][S10]

### 5. Clerk's own examples support disposable users and separate auth setup

Clerk's first-party Playwright demo creates a “dedicated test user for this sign-in test via the Backend API,” uses a unique `+clerk_test` address, and tracks its ID for cleanup. Its teardown deletes tracked users; setup handles stale records from interrupted runs. The docs also recommend global teardown to prevent test users from accumulating. For this repo, delete only IDs created by the fixture; preserve a run-owned cleanup record if interrupted-run recovery is needed, and report cleanup errors without hiding the original test failure. Do not bulk-delete users by a loose email prefix. [S5][S6][S7]

For tests that need a session rather than proof of the form, Clerk recommends `clerk.signIn({ page, emailAddress })`: it creates a server-side sign-in token and skips first-factor verification. The SDK source looks up the user, creates that token, and uses the `ticket` strategy. It does not unlock the user. Keep the repo's existing token-based application tests and small real-form happy path; change the negative test's user ownership, not its real Clerk coverage. MFA and session-task limits still apply to the helper. [S11][S12]

## What this establishes for issue 02

The user's report establishes that repeat runs locked the shared user. The sources explain a supported way to avoid repeating that harm. They do not prove that lockout caused the earlier timeout while waiting for “Enter verification code.” Clerk documents locked authentication as HTTP `403`, code `user_locked`, with raw `meta.lockout_expires_in_seconds`; inspect that specific evidence in a later authorized run rather than treating every timeout as lockout. [S13]

Next: change only the wrong-password test to use a disposable-user fixture before running auth tests again. Keep the existing signed-in artifact restrictions and compose verification path in `docs/agents/testing.md`. No installed SDK-version check or runtime verification was done; the SDK citations below pin the source reviewed.

## Sources and cited passages

[S1]: https://clerk.com/docs/guides/protect/user-lockout "Lock accounts after failed sign-ins: How account lockout works; Settings; Unlock user accounts."
[S2]: https://clerk.com/docs/guides/development/testing/overview#testing-tokens "Testing Tokens: specific protection bypasses, instance scope, and query parameter."
[S3]: https://clerk.com/docs/guides/protect/protections/overview "Available protections; Protections that block or challenge; Manage protections in the Dashboard."
[S4]: https://clerk.com/docs/reference/backend/user/unlock-user "unlockUser: removes a sign-in lock; signature; Backend API endpoint."
[S5]: https://github.com/clerk/clerk-playwright-nextjs/blob/2c4f6e4deef7bd7e4035675b7c3a7b6e919a9de5/e2e/app.spec.ts "sign in test: dedicated user via createUser, unique address, trackCreatedUser."
[S6]: https://github.com/clerk/clerk-playwright-nextjs/blob/2c4f6e4deef7bd7e4035675b7c3a7b6e919a9de5/e2e/global.teardown.ts "cleanup test users: delete tracked IDs; email fallback."
[S7]: https://clerk.com/docs/guides/development/testing/playwright/test-sign-up-flows#cleaning-up-test-users "Cleaning up test users: use Playwright global teardown to delete users created during the run."
[S8]: https://github.com/clerk/javascript/blob/c9bdae3b0aef91699a73824fecc6726959196ab4/packages/testing/src/playwright/setupClerkTestingToken.ts "setupClerkTestingToken: appends token; overrides captcha_bypass; request retry handling."
[S9]: https://clerk.com/docs/guides/development/testing/test-emails-and-phones "Set up test mode; Email addresses; Testing email verification codes."
[S10]: https://github.com/clerk/javascript/blob/c9bdae3b0aef91699a73824fecc6726959196ab4/packages/backend/src/api/endpoints/UserApi.ts "UserApi.createUser; deleteUser; unlockUser implementations."
[S11]: https://clerk.com/docs/guides/development/testing/playwright/test-helpers#clerk-signin "Recommended emailAddress parameter: server-side token; first-factor bypass; MFA/session-task limits."
[S12]: https://github.com/clerk/javascript/blob/c9bdae3b0aef91699a73824fecc6726959196ab4/packages/testing/src/playwright/helpers.ts "signIn emailAddress branch: getUserList, createSignInToken, ticket strategy."
[S13]: https://clerk.com/docs/js-frontend/guides/development/custom-flows/error-handling "Account lockout: HTTP 403; user_locked; raw lockout_expires_in_seconds."

Interrupted-run cleanup source: [demo global setup](https://github.com/clerk/clerk-playwright-nextjs/blob/2c4f6e4deef7bd7e4035675b7c3a7b6e919a9de5/e2e/global.setup.ts). Delete API contract: [deleteUser](https://clerk.com/docs/reference/backend/user/delete-user), “Deletes the given User.”
