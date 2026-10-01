# 04: Login and register polish, plus auth redirects

**What to build:** Branded, accessible login and register pages, and correct redirect behavior around them. This absorbs two existing tickets from the MVP issue folder: 17 (post-login redirect bug) and 18 (redirect signed-in users away from guest routes); read their briefs for the detailed current-behavior analysis. Spec: `.scratch/frontend-visual-identity/spec.md`.

**Blocked by:** 02

**Status:** ready-for-agent

- [ ] Centered card using the narrow layout, with the brand mark and a link between login and register; the existing "account created" notice is preserved
- [ ] Correct autocomplete hints (email, current password, new password); field-level errors tied to their inputs and announced to assistive tech; a pending state on submit; a show/hide password toggle. Form errors stay inline (no toasts)
- [ ] Signed-in users visiting login or register are redirected to `/` (absorbs ticket 18); unauthenticated visitors reach the forms normally
- [ ] A user sent to login from a protected page lands back on that page after signing in, including its query string (absorbs ticket 17); keep the same-origin safety check so an external redirect target is still rejected
- [ ] Unit tests cover the guest guard and the preserved path-plus-search redirect; Playwright covers unauthenticated navigation to a protected route then returning after login, and a signed-in user visiting login and register; axe check on both forms
- [ ] No social login and no forgot-password flow (ADR-0004)
- [ ] Mobile-first; lint, typecheck, tests and build pass; note anything not verified in a browser
- [ ] When done, tickets 17 and 18 in the MVP issue folder are already marked absorbed; add a comment to each pointing at this ticket's outcome
