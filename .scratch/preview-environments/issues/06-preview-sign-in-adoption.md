# 06: Preview sign-in adopts the matching copied User

**What to build:** In a preview, signing in with a development Clerk account whose verified email matches exactly one copied User adopts that User, so the maintainer sees their own Collections and Inventory Entries. The adopted User keeps its profile, Collections and Inventory Entries; only its Auth Identity changes. Outside previews nothing changes: an unknown Auth Identity is a new User. This is the second pull request. It relies on the development Clerk instance requiring a verified email at sign-up (confirmed by the maintainer) and adds no schema change, no new token claim and no allowlist.

**Blocked by:** 05 (the first pull request is verified live before this begins).

**Status:** ready-for-agent

- [ ] With `APP_ENV` not `preview`, an unknown Auth Identity creates a new User exactly as today, even when a User with the same email exists.
- [ ] With `APP_ENV=preview` and exactly one User whose email matches (case-insensitively), resolving an unknown Auth Identity rewrites that User's auth provider and subject to the caller's and returns that User's profile; its Collections and Inventory Entries are still attached.
- [ ] With `APP_ENV=preview` and no matching User, a new User is created.
- [ ] With `APP_ENV=preview` and several matching Users, a new User is created and none is adopted.
- [ ] Two concurrent first requests for the same identity adopt once and never create a duplicate User or fail, using the existing per-identity advisory lock.
- [ ] After adoption, a later request with the same identity resolves directly, and the identity cache cannot resolve the old subject to the wrong User.
- [ ] The behaviour is covered against real Postgres in the style of the existing resolve-caller tests (seam 2).
- [ ] The Clerk deployment guide documents adoption, that it is preview-only, and that it depends on the development instance requiring a verified email.
