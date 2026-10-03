# 09: End-to-end sign-in on Testing Tokens

**Parent:** `.scratch/security-hardening/spec.md`

**What to build:** The Playwright end-to-end layer signs in through the real Clerk development instance using Clerk Testing Tokens and a dedicated test user, replacing every use of the old register form. A thin sign-in spec proves the flow; other specs sign in the same way. CI runs it with the right secrets and skips fork pull requests.

**Blocked by:** 08, 01.

**Status:** ready-for-agent

- [ ] A sign-in spec covers the happy path and one key failure (wrong password), not every branch.
- [ ] Existing specs that registered a user through the old form sign in through the new helper instead, and still pass.
- [ ] CI provides the Clerk secret key and the test user's credentials from repository secrets, and the e2e job is skipped on fork pull requests.
- [ ] No secret appears in the repo or logs.
- [ ] The frontend testing doc describes how e2e authenticates and what secrets it needs.
- [ ] The e2e specs that need a seeded backend are documented as such, and the ticket records which ones were actually run.

## Comments
