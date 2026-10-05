# 01: Clerk dashboard setup

**Parent:** `.scratch/security-hardening/spec.md`

**What to build:** A working Clerk development instance (and, later, a production instance on the owner's domain) configured exactly as the spec requires, with its keys stored where the backend, frontend and CI will read them. Only the owner can do this; deliver it as an interactive bash wizard (the `wizard` skill) that walks through each dashboard step and verifies what it can from the command line.

**Blocked by:** None (can start immediately). The production-instance half also needs the domain from ticket 02.

**Status:** done — wizard delivered in `scripts/clerk-setup.sh` (commits `870da19`, `47e5c38`); acceptance boxes stay unticked until it has been run against a real Clerk instance.

## Steps the wizard covers

- Create the development instance and, when the domain is ready, the production instance.
- Enable Google as the only social provider.
- Enable email and password with mandatory email verification (email code), and forgot/reset password.
- Enable account lockout, bulk user enumeration protection and bot protection. Note which settings turned out to be plan-gated.
- Add two custom claims to the session token: the primary email and the full name. Check the token stays within Clerk's size limit.
- Create a dedicated test user in the development instance for end-to-end tests.
- Record the keys: backend secret key and the allowed frontend origins on Railway, the publishable key on Vercel (production) and in the local env example, and the secret key plus test-user credentials as GitHub secrets.

## Acceptance criteria

- [ ] Development instance signs in with Google and with a verified email and password account.
- [ ] An unverified email signup cannot obtain a session.
- [ ] A decoded session token from the development instance contains the email and name claims.
- [ ] Lockout, enumeration protection and bot protection are enabled (or the plan limitation is recorded in the ticket).
- [ ] Keys are stored in Railway, Vercel, GitHub and the local env example, and no secret is committed.
- [ ] The wizard script is re-runnable and skips steps that are already done.

## Comments

**Wizard:** `scripts/clerk-setup.sh` (development, 9 stages) and `scripts/clerk-setup.sh production` (8 stages, after ticket 02). The wizard prints the plan-gated settings at the end; paste them here. Variable names it uses, for tickets 07 to 09 to adopt: `CLERK_SECRET_KEY`, `VITE_CLERK_PUBLISHABLE_KEY`, `APP_CLIENT_URLS` (existing), `E2E_CLERK_USER_EMAIL`, `E2E_CLERK_USER_PASSWORD`. GitHub: secrets `CLERK_SECRET_KEY`, `E2E_CLERK_USER_EMAIL`, `E2E_CLERK_USER_PASSWORD`, and a variable `VITE_CLERK_PUBLISHABLE_KEY`. Pull-request previews use the development instance, so the preview workflow (tickets 07 to 09) must read those two from GitHub and push them to the Railway PR environment and the Vercel preview. Not covered by the wizard: the unverified-signup and forgot-password checks are manual confirmations in the Account Portal.

**Clerk development settings (wizard run 2026-10-03):** lockout: enabled; enumeration: enabled; bot: enabled.

**Clerk production settings (wizard run 2026-10-05):** lockout: enabled; enumeration: enabled; bot: enabled.
