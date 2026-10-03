# 08: Clerk frontend

**Parent:** `.scratch/security-hardening/spec.md` (see ADR-0015)

**What to build:** Sign-in and sign-up run on Clerk's prebuilt components, themed to the visual identity, at their current addresses. Every API call carries the Clerk token as a bearer header, and signed-in state across the shell, route guards and redirects comes from Clerk. A user can sign up with Google or email, verify, sign in, use the app and sign out.

**Blocked by:** 07, 01.

**Status:** ready-for-agent

- [ ] The Clerk provider wraps the app, which stays an SPA (ADR-0014 is not reopened). Confirm Clerk's SDK for TanStack Start works in the app's SPA mode.
- [ ] The sign-in and sign-up pages stay at their current addresses and render the prebuilt components themed for light and dark, keeping the noindex behavior those pages have today.
- [ ] The shared request wrapper used by the generated client attaches the current token as a bearer header. On a 401 it requests a fresh token and retries once; a second 401 triggers the existing auth-lost handling.
- [ ] The shell (header, user menu, sign-out), route guards and the redirect rules (return to the original page after sign-in, restricted to same-origin paths; signed-in users redirected away from guest-only pages) derive from Clerk. The name and email shown come from Clerk's user data.
- [ ] Removed: the cookie and CSRF token handling, the refresh-on-401 flow, the current-user query, and the old login and register forms, with their tests.
- [ ] Feature tests (generated client and Clerk hooks mocked) cover the bearer header, retry-once, guards and redirects, and shell state for guest and signed-in users.
- [ ] The Clerk publishable key is added to the frontend env example and the deployment doc; frontend build, type check, lint and tests pass. UI in a real browser is verified manually or by ticket 09's e2e layer, and the ticket says which.

## Comments
