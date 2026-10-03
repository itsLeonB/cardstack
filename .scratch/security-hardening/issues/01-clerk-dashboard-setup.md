# 01: Clerk dashboard setup

**Parent:** `.scratch/security-hardening/spec.md`

**What to build:** A working Clerk development instance (and, later, a production instance on the owner's domain) configured exactly as the spec requires, with its keys stored where the backend, frontend and CI will read them. Only the owner can do this; deliver it as an interactive bash wizard (the `wizard` skill) that walks through each dashboard step and verifies what it can from the command line.

**Blocked by:** None (can start immediately). The production-instance half also needs the domain from ticket 02.

**Status:** ready-for-human

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
