# 04: Login and register polish, plus auth redirects

**What to build:** Branded, accessible login and register pages, and correct redirect behavior around them. This absorbs two existing tickets from the MVP issue folder: 17 (post-login redirect bug) and 18 (redirect signed-in users away from guest routes); read their briefs for the detailed current-behavior analysis. Spec: `.scratch/frontend-visual-identity/spec.md`.

**Blocked by:** 02

**Status:** done

- [x] Centered card using the narrow layout, with the brand mark and a link between login and register; the existing "account created" notice is preserved
- [x] Correct autocomplete hints (email, current password, new password); field-level errors tied to their inputs and announced to assistive tech; a pending state on submit; a show/hide password toggle. Form errors stay inline (no toasts)
- [x] Signed-in users visiting login or register are redirected to `/` (absorbs ticket 18); unauthenticated visitors reach the forms normally
- [x] A user sent to login from a protected page lands back on that page after signing in, including its query string (absorbs ticket 17); keep the same-origin safety check so an external redirect target is still rejected
- [x] Unit tests cover the guest guard and the preserved path-plus-search redirect; Playwright covers unauthenticated navigation to a protected route then returning after login, and a signed-in user visiting login and register; axe check on both forms
- [x] No social login and no forgot-password flow (ADR-0004)
- [x] Mobile-first; lint, typecheck, tests and build pass; note anything not verified in a browser
- [x] When done, tickets 17 and 18 in the MVP issue folder are already marked absorbed; add a comment to each pointing at this ticket's outcome

## Outcome

Login and register use a shared `AuthPage`/`AuthField` card (narrow layout, `BrandMark`, show/hide password, field errors tied to inputs, pending state). `requireGuest` redirects signed-in users from `/login` and `/register` to `/`. `requireAuth` preserves path plus query string, and a shared `redirectSchema` plus `isSameOriginPath` rejects external targets (an invalid `?redirect=` is dropped and login falls back to `/account`). The login and register cross-links carry the validated `redirect`. Session cache is now reset (not just invalidated) and awaited after login and logout so the guards never read a stale session.

Not verified in a real browser: visual pass of the card (spacing, brand mark, dark mode) and login against the real backend; Playwright specs run with the API stubbed. Three pre-existing e2e failures in `breadcrumbs.spec.ts` and `app-shell.spec.ts` need a seeded catalog and were not rerun on the baseline.

Left for later: public catalog "Sign in" links (`card-holdings.tsx`, `nav-links.ts`) still drop the return path; `requireGuest` ignores `?redirect` and sends a signed-in user to `/`; the hash is dropped by `requireAuth`.
