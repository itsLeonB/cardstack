# 03: Admin app: sign-in and access gate

**What to build:** A new top-level `admin/` package that an Admin can run locally on its own port against the local API and the development Clerk instance. It is built like the user app (TanStack Start in SPA mode, Vite, Tailwind 4, orval, bun, oxlint, vitest) with its own copy of the shadcn components and design tokens it needs. A Guest, or a signed-in user who is not listed, sees the ordinary not-found page; an Admin sees an empty shell with sign-out. A failed access check (429, 5xx) shows an error, not not-found. It serves `noindex`, a `robots.txt` that disallows everything and no sitemap, and its production build fails when the Clerk publishable key or the API address is missing. It has its own orval target reading the same `openapi.json`, limited to the admin operations and the catalog operations it will call. A CI job runs typecheck, lint, tests and build on changes to it and is a required check.

Spec: `.scratch/admin-app-extraction/spec.md`. Decision: `docs/adr/0024-the-admin-is-its-own-app-and-its-routes-accept-only-its-origin.md`. Vocabulary: Admin app, Admin, Published in `GLOSSARY.md`. Conventions: `docs/agents/conventions/general.md` and `frontend.md`.

**Blocked by:** 01, 02

**Status:** resolved

- [ ] `bun run dev` in the package serves the app on its own port; with the local API and a listed email an Admin signs in and sees the shell
- [ ] A Guest and an unlisted signed-in user see the same not-found page, and a 429 or 5xx from the access check shows an error
- [ ] The access check asks the API and treats 204 as an Admin and 404 as not found, as the old `/admin` gate did; a Guest is not even asked
- [ ] Responses carry `noindex`, `robots.txt` disallows everything, and no sitemap is built
- [ ] A production build exits non-zero when the Clerk key or the API address is unset, and warns and continues elsewhere
- [ ] The generated client is produced by `bun run codegen` from the same `openapi.json` and holds only the operations the Admin app calls
- [ ] Component tests of the gate (Admin, Guest, unlisted, failed check) run with the generated client mocked
- [ ] A CI workflow for the package runs typecheck, lint, tests and build, and is listed as a required check
- [ ] The package's local settings example lists the Clerk key and the API address, and the local Admin origin is the one in the API's `ADMIN_CLIENT_URLS` example

## Comments

- For the human: make the `Admin CI` jobs (`Lint`, `Format`, `Typecheck`, `Test`, `Build` of `.github/workflows/admin-ci.yml`) required checks in GitHub. The agent cannot set it, and `main` has no branch protection or ruleset today. The workflow is path-filtered to `admin/**`, so a required check on it stays pending for PRs that do not touch `admin/`, unless the ruleset allows skipped checks.
- For the human: set the repository variables `VITE_CLERK_PUBLISHABLE_KEY` and `VITE_API_BASE_URL` if the CI build should bake them in (outside a Vercel production build they only warn).
- The Admin app has no sign-in page: a Guest sees not-found, so an Admin signs in through the user app (`http://localhost:3000`) and the session must carry over to `http://localhost:3001`. Not checked in a browser; verify it once with a listed email. If the dev instance does not share the session across ports, the ticket needs a sign-in route.
- `frontend/` tests `collection-card-results.pages.test.tsx` failed twice under load once while `scripts/verification/frontend.sh` ran next to other work, and passed on rerun; unrelated to this ticket.
